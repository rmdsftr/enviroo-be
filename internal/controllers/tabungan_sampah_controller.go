package controllers

import (
	"enviroo-be/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ────────────────────────────────────────────────────────────
// Response structs
// ────────────────────────────────────────────────────────────

type ItemBukuTabungan struct {
	TabunganID string   `json:"tabungan_id"`
	NamaSampah string   `json:"nama_sampah"`
	Satuan     string   `json:"satuan"`
	RewardID   int      `json:"reward_id"`
	NamaReward string   `json:"nama_reward"`
	QtySetoran float64  `json:"qty_setoran"`
	SisaQty    float64  `json:"sisa_qty"`
	HargaItem  *float64 `json:"harga_item"`  // nil kalau belum cair
	NilaiTotal *float64 `json:"nilai_total"` // nil kalau belum cair
	Status     string   `json:"status"`
}

type SetoranBukuTabungan struct {
	SourceID       string             `json:"source_id"`
	TanggalSetoran string             `json:"tanggal_setoran"`
	Items          []ItemBukuTabungan `json:"items"`
}

type BukuTabunganResponse struct {
	NasabahID string                `json:"nasabah_id"`
	Setoran   []SetoranBukuTabungan `json:"setoran"`
}

// ────────────────────────────────────────────────────────────
// Struct bantu untuk hasil raw query join
// ────────────────────────────────────────────────────────────

type hargaTabungan struct {
	TabunganID string
	SampahID   string
	HargaItem  float64
}

// ────────────────────────────────────────────────────────────
// Controller
// ────────────────────────────────────────────────────────────

type TabunganSampahController struct {
	DB *gorm.DB
}

func NewTabunganSampahController(db *gorm.DB) *TabunganSampahController {
	return &TabunganSampahController{DB: db}
}

// GetBukuTabunganSampahNasabah godoc
// GET /tabungan-sampah/buku-tabungan/:nasabah_id
//
// Harga item diambil dari detail_bagi_hasil melalui jalur:
//
//	tabungan_sampah.tabungan_id
//	  → pencairan_tabungan.tabungan_id
//	  → pencairan_tabungan.penerima_id = penerima_bagi_hasil.penerima_id
//	  → penerima_bagi_hasil.penerima_id = detail_bagi_hasil.penerima_id
//	  → detail_bagi_hasil.sampah_id = tabungan_sampah.sampah_id
//
// RewardID dan NamaReward diambil dari:
//
//	katalog_sampah.reward_id → reward.reward_id
//
// Kalau tabungan belum cair, harga_item dan nilai_total akan null.
//
// Status pencairan:
//
//	"Cair"          → sisa_qty == 0
//	"Cair Sebagian" → 0 < sisa_qty < qty
//	"Belum Cair"    → sisa_qty == qty
func (tc *TabunganSampahController) GetBukuTabunganSampahNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")
	if nasabahID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nasabah_id wajib diisi"})
		return
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// ── 1. Ambil semua tabungan milik nasabah ──────────────────────
	var tabungans []models.TabunganSampah
	q := tc.DB.
		Where("nasabah_id = ? AND entitas = ?", nasabahID, models.EntitasNasabah).
		Preload("KatalogSampah.Reward").
		Preload("KatalogSampah.Sarok")

	if startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			q = q.Where("created_at <= ?", t.Add(24*time.Hour-time.Second))
		}
	}

	if err := q.Order("created_at ASC").Find(&tabungans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data tabungan: " + err.Error()})
		return
	}

	if len(tabungans) == 0 {
		c.JSON(http.StatusOK, BukuTabunganResponse{
			NasabahID: nasabahID,
			Setoran:   []SetoranBukuTabungan{},
		})
		return
	}

	// ── 2. Kumpulkan tabungan_id untuk query harga ─────────────────
	tabunganIDs := make([]string, 0, len(tabungans))
	for _, t := range tabungans {
		tabunganIDs = append(tabunganIDs, t.TabunganID)
	}

	// ── 3. Join ke detail_bagi_hasil untuk dapat harga_item ─────────
	//
	// Jalur join:
	//   pencairan_tabungan (tabungan_id)
	//   → penerima_bagi_hasil (penerima_id)
	//   → detail_bagi_hasil (penerima_id + sampah_id)
	//   → match sampah_id dengan tabungan_sampah
	//
	// Kalau satu tabungan pernah cair di beberapa bagi hasil (cair
	// sebagian berkali-kali), kita ambil harga dari pencairan pertama.
	var hargaRows []hargaTabungan
	if err := tc.DB.Raw(`
		SELECT DISTINCT ON (pt.tabungan_id)
			pt.tabungan_id,
			dbh.sampah_id,
			dbh.harga_item
		FROM pencairan_tabungan pt
		JOIN penerima_bagi_hasil pbh ON pbh.penerima_id = pt.penerima_id
		JOIN detail_bagi_hasil   dbh ON dbh.penerima_id = pbh.penerima_id
		JOIN tabungan_sampah     ts  ON ts.tabungan_id  = pt.tabungan_id
		                            AND ts.sampah_id    = dbh.sampah_id
		WHERE pt.tabungan_id IN ?
		ORDER BY pt.tabungan_id, pt.penerima_id ASC
	`, tabunganIDs).Scan(&hargaRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data harga: " + err.Error()})
		return
	}

	// ── 4. Bangun map tabungan_id → harga ───────────────────────────
	hargaMap := make(map[string]hargaTabungan, len(hargaRows))
	for _, row := range hargaRows {
		hargaMap[row.TabunganID] = row
	}

	// ── 5. Kumpulkan source_id unik untuk query tanggal setoran ─────
	sourceIDSet := make(map[string]struct{})
	for _, t := range tabungans {
		if t.SourceID != nil && *t.SourceID != "" {
			sourceIDSet[*t.SourceID] = struct{}{}
		}
	}

	sourceIDs := make([]string, 0, len(sourceIDSet))
	for id := range sourceIDSet {
		sourceIDs = append(sourceIDs, id)
	}

	setoranMap := make(map[string]models.SetoranNasabah)
	if len(sourceIDs) > 0 {
		var setorans []models.SetoranNasabah
		if err := tc.DB.
			Where("setoran_id IN ?", sourceIDs).
			Find(&setorans).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data setoran: " + err.Error()})
			return
		}
		for _, s := range setorans {
			setoranMap[s.SetoranID] = s
		}
	}

	// ── 6. Grouping tabungan berdasarkan source_id ───────────────────
	orderSlice := []string{}
	groupMap := make(map[string][]models.TabunganSampah)

	for _, t := range tabungans {
		key := ""
		if t.SourceID != nil {
			key = *t.SourceID
		}
		if _, exists := groupMap[key]; !exists {
			orderSlice = append(orderSlice, key)
		}
		groupMap[key] = append(groupMap[key], t)
	}

	// ── 7. Bangun response ───────────────────────────────────────────
	setoranList := make([]SetoranBukuTabungan, 0, len(orderSlice))

	for _, sourceID := range orderSlice {
		tanggal := ""
		if s, ok := setoranMap[sourceID]; ok {
			tanggal = s.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		}

		items := groupMap[sourceID]
		itemList := make([]ItemBukuTabungan, 0, len(items))

		for _, tab := range items {
			namaSampah := ""
			satuan := ""
			rewardID := 0
			namaReward := ""

			if tab.KatalogSampah != nil {
				namaSampah = tab.KatalogSampah.Sarok.NamaSampah
				satuan = string(tab.KatalogSampah.Sarok.Satuan)
				rewardID = tab.KatalogSampah.RewardID
				namaReward = string(tab.KatalogSampah.Reward.NamaReward)
			}

			status := statusCair(tab.Qty, tab.SisaQty)

			// Harga dari detail_bagi_hasil — nil kalau belum ada pencairan
			var hargaItem *float64
			var nilaiTotal *float64
			if h, ok := hargaMap[tab.TabunganID]; ok {
				hi := h.HargaItem
				nt := tab.Qty * h.HargaItem
				hargaItem = &hi
				nilaiTotal = &nt
			}

			itemList = append(itemList, ItemBukuTabungan{
				TabunganID: tab.TabunganID,
				NamaSampah: namaSampah,
				Satuan:     satuan,
				RewardID:   rewardID,
				NamaReward: namaReward,
				QtySetoran: tab.Qty,
				SisaQty:    tab.SisaQty,
				HargaItem:  hargaItem,
				NilaiTotal: nilaiTotal,
				Status:     status,
			})
		}

		setoranList = append(setoranList, SetoranBukuTabungan{
			SourceID:       sourceID,
			TanggalSetoran: tanggal,
			Items:          itemList,
		})
	}

	c.JSON(http.StatusOK, BukuTabunganResponse{
		NasabahID: nasabahID,
		Setoran:   setoranList,
	})
}

// ────────────────────────────────────────────────────────────
// Helper
// ────────────────────────────────────────────────────────────

func statusCair(qty, sisaQty float64) string {
	switch {
	case sisaQty == 0:
		return "Cair"
	case sisaQty >= qty:
		return "Belum Cair"
	default:
		return "Cair Sebagian"
	}
}

type BukuTabunganBSUResponse struct {
	BankID      string                `json:"bank_id"`
	Pengangkutan []SetoranBukuTabungan `json:"pengangkutan"`
}

type tanggalPengangkutan struct {
	SourceID  string
	ChangedAt string
}
 
func buildHargaMap(db *gorm.DB, tabunganIDs []string, c *gin.Context) map[string]hargaTabungan {
	var hargaRows []hargaTabungan
	if err := db.Raw(`
		SELECT DISTINCT ON (pt.tabungan_id)
			pt.tabungan_id,
			dbh.sampah_id,
			dbh.harga_item
		FROM pencairan_tabungan pt
		JOIN penerima_bagi_hasil pbh ON pbh.penerima_id = pt.penerima_id
		JOIN detail_bagi_hasil   dbh ON dbh.penerima_id = pbh.penerima_id
		JOIN tabungan_sampah     ts  ON ts.tabungan_id  = pt.tabungan_id
		                            AND ts.sampah_id    = dbh.sampah_id
		WHERE pt.tabungan_id IN ?
		ORDER BY pt.tabungan_id, pt.penerima_id ASC
	`, tabunganIDs).Scan(&hargaRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data harga: " + err.Error()})
		return nil
	}
 
	hargaMap := make(map[string]hargaTabungan, len(hargaRows))
	for _, row := range hargaRows {
		hargaMap[row.TabunganID] = row
	}
	return hargaMap
}

func buildSetoranList(
	tabungans []models.TabunganSampah,
	tanggalMap map[string]string,
	hargaMap map[string]hargaTabungan,
) []SetoranBukuTabungan {
	orderSlice := []string{}
	groupMap := make(map[string][]models.TabunganSampah)
 
	for _, t := range tabungans {
		key := ""
		if t.SourceID != nil {
			key = *t.SourceID
		}
		if _, exists := groupMap[key]; !exists {
			orderSlice = append(orderSlice, key)
		}
		groupMap[key] = append(groupMap[key], t)
	}
 
	setoranList := make([]SetoranBukuTabungan, 0, len(orderSlice))
 
	for _, sourceID := range orderSlice {
		tanggal := tanggalMap[sourceID] // kosong kalau belum completed
 
		items := groupMap[sourceID]
		itemList := make([]ItemBukuTabungan, 0, len(items))
 
		for _, tab := range items {
			namaSampah := ""
			satuan := ""
			rewardID := 0
			namaReward := ""
 
			if tab.KatalogSampah != nil {
				namaSampah = tab.KatalogSampah.Sarok.NamaSampah
				satuan = string(tab.KatalogSampah.Sarok.Satuan)
				rewardID = tab.KatalogSampah.RewardID
				namaReward = string(tab.KatalogSampah.Reward.NamaReward)
			}
 
			status := statusCair(tab.Qty, tab.SisaQty)
 
			var hargaItem *float64
			var nilaiTotal *float64
			if h, ok := hargaMap[tab.TabunganID]; ok {
				hi := h.HargaItem
				nt := tab.Qty * h.HargaItem
				hargaItem = &hi
				nilaiTotal = &nt
			}
 
			itemList = append(itemList, ItemBukuTabungan{
				TabunganID: tab.TabunganID,
				NamaSampah: namaSampah,
				Satuan:     satuan,
				RewardID:   rewardID,
				NamaReward: namaReward,
				QtySetoran: tab.Qty,
				SisaQty:    tab.SisaQty,
				HargaItem:  hargaItem,
				NilaiTotal: nilaiTotal,
				Status:     status,
			})
		}
 
		setoranList = append(setoranList, SetoranBukuTabungan{
			SourceID:       sourceID,
			TanggalSetoran: tanggal,
			Items:          itemList,
		})
	}
 
	return setoranList
}

func (tc *TabunganSampahController) GetBukuTabunganSampahBSU(c *gin.Context) {
	bankID := c.Param("bsu_id")
	if bankID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bsu_id wajib diisi"})
		return
	}
 
	// ── 1. Ambil semua tabungan milik BSU ─────────────────────────
	var tabungans []models.TabunganSampah
	if err := tc.DB.
		Where("bank_id = ? AND entitas = ?", bankID, models.EntitasBankSampah).
		Preload("KatalogSampah.Reward").
		Preload("KatalogSampah.Sarok").
		Order("created_at ASC").
		Find(&tabungans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data tabungan: " + err.Error()})
		return
	}
 
	if len(tabungans) == 0 {
		c.JSON(http.StatusOK, BukuTabunganBSUResponse{
			BankID:       bankID,
			Pengangkutan: []SetoranBukuTabungan{},
		})
		return
	}
 
	// ── 2. Kumpulkan tabungan_id untuk query harga ────────────────
	tabunganIDs := make([]string, 0, len(tabungans))
	for _, t := range tabungans {
		tabunganIDs = append(tabunganIDs, t.TabunganID)
	}
 
	// ── 3. Ambil harga_item dari detail_bagi_hasil ────────────────
	hargaMap := buildHargaMap(tc.DB, tabunganIDs, c)
	if hargaMap == nil {
		return
	}
 
	// ── 4. Kumpulkan source_id unik (= pengangkutan_id) ──────────
	sourceIDSet := make(map[string]struct{})
	for _, t := range tabungans {
		if t.SourceID != nil && *t.SourceID != "" {
			sourceIDSet[*t.SourceID] = struct{}{}
		}
	}
 
	pengangkutanIDs := make([]string, 0, len(sourceIDSet))
	for id := range sourceIDSet {
		pengangkutanIDs = append(pengangkutanIDs, id)
	}
 
	// ── 5. Ambil tanggal completed dari riwayat_pengangkutan ──────
	//
	// Diambil changed_at dari baris dengan status = 'completed'.
	// Kalau pengangkutan belum completed, tanggal akan kosong.
	tanggalMap := make(map[string]string) // pengangkutan_id → tanggal
	if len(pengangkutanIDs) > 0 {
		var riwayats []tanggalPengangkutan
		if err := tc.DB.Raw(`
			SELECT DISTINCT ON (pengangkutan_id)
				pengangkutan_id AS source_id,
				TO_CHAR(changed_at, 'YYYY-MM-DD"T"HH24:MI:SSOF') AS changed_at
			FROM riwayat_pengangkutan
			WHERE pengangkutan_id IN ?
			  AND status_pengangkutan = ?
			ORDER BY pengangkutan_id, changed_at DESC
		`, pengangkutanIDs, models.StatusCompleted).Scan(&riwayats).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil tanggal pengangkutan: " + err.Error()})
			return
		}
		for _, r := range riwayats {
			tanggalMap[r.SourceID] = r.ChangedAt
		}
	}
 
	// ── 6. Grouping & bangun response ────────────────────────────
	pengangkutanList := buildSetoranList(tabungans, tanggalMap, hargaMap)
 
	c.JSON(http.StatusOK, BukuTabunganBSUResponse{
		BankID:       bankID,
		Pengangkutan: pengangkutanList,
	})
}