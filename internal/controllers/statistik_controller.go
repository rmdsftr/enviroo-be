package controllers

import (
	"enviroo-be/internal/models"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type StatistikController struct {
	DB *gorm.DB
}

func NewStatistikController(db *gorm.DB) *StatistikController {
	return &StatistikController{DB: db}
}

func (sc *StatistikController) GetBankSampahStatistik(c *gin.Context) {
	var results []struct {
		JenisBank string `gorm:"column:jenis_bank"`
		Count     int64  `gorm:"column:count"`
	}

	if err := sc.DB.Model(&models.BankSampah{}).
		Select("jenis_bank, count(*) as count").
		Group("jenis_bank").
		Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get statistik: " + err.Error()})
		return
	}

	stats := map[string]int64{
		string(models.BSI): 0,
		string(models.BSM): 0,
		string(models.BSU): 0,
	}

	for _, r := range results {
		stats[r.JenisBank] = r.Count
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Statistik retrieved successfully",
		"data":    stats,
	})
}

func (sc *StatistikController) GetStatistikSetoranSampahBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	// Filter range: bulan_mulai+tahun_mulai s/d bulan_selesai+tahun_selesai (inklusif bulan selesai)
	var startDate, endDate time.Time
	var filterTanggal bool

	bmStr := c.Query("bulan_mulai")
	tmStr := c.Query("tahun_mulai")
	bsStr := c.Query("bulan_selesai")
	tsStr := c.Query("tahun_selesai")

	if bmStr != "" || tmStr != "" || bsStr != "" || tsStr != "" {
		bm, e1 := strconv.Atoi(bmStr)
		tm, e2 := strconv.Atoi(tmStr)
		bs, e3 := strconv.Atoi(bsStr)
		ts, e4 := strconv.Atoi(tsStr)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil ||
			bm < 1 || bm > 12 || bs < 1 || bs > 12 || tm < 2000 || ts < 2000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter bulan_mulai, tahun_mulai, bulan_selesai, tahun_selesai tidak valid"})
			return
		}
		startDate = time.Date(tm, time.Month(bm), 1, 0, 0, 0, 0, time.UTC)
		// endDate = awal bulan setelah bulan_selesai (eksklusif) agar inklusif s/d akhir bulan_selesai
		endDate = time.Date(ts, time.Month(bs+1), 1, 0, 0, 0, 0, time.UTC)
		if endDate.Before(startDate) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Rentang tanggal tidak valid: bulan_selesai lebih awal dari bulan_mulai"})
			return
		}
		filterTanggal = true
	}

	var bank models.BankSampah
	if err := sc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	type StatistikItem struct {
		SampahID    string  `gorm:"column:sampah_id" json:"sampah_id"`
		NamaSampah  string  `gorm:"column:nama_sampah" json:"nama_sampah"`
		Satuan      string  `gorm:"column:satuan" json:"satuan"`
		JenisReward string  `gorm:"column:jenis_reward" json:"jenis_reward"`
		Kategori    string  `gorm:"column:kategori" json:"kategori"`
		TotalQty    float64 `gorm:"column:total_qty" json:"total_qty"`
	}

	type GroupKey struct {
		SampahID    string
		Satuan      string
		JenisReward string
		Kategori    string
	}

	aggregated := make(map[GroupKey]StatistikItem)

	addToAggregated := func(item StatistikItem) {
		key := GroupKey{item.SampahID, item.Satuan, item.JenisReward, item.Kategori}
		if existing, ok := aggregated[key]; ok {
			existing.TotalQty += item.TotalQty
			aggregated[key] = existing
		} else {
			aggregated[key] = item
		}
	}

	// Setoran nasabah langsung ke bank ini
	nasabahQuery := `
		SELECT
			ks.sampah_id,
			s.nama_sampah,
			s.satuan,
			r.nama_reward AS jenis_reward,
			k.kategori,
			SUM(dsn.qty) AS total_qty
		FROM setoran_nasabah sn
		JOIN admin a ON sn.admin_id = a.admin_id
		JOIN detail_setoran_nasabah dsn ON sn.setoran_id = dsn.setoran_id
		JOIN katalog_sampah ks ON dsn.sampah_id = ks.sampah_id
		JOIN sampah s ON s.sarok_id = ks.sarok_id
		JOIN reward r ON ks.reward_id = r.reward_id
		JOIN kategori_sampah k ON ks.kategori_id = k.kategori_id
		WHERE sn.status_setoran = 'berhasil'
		  AND a.bank_id = ?`
	nasabahArgs := []interface{}{bankID}
	if filterTanggal {
		nasabahQuery += ` AND sn.created_at >= ? AND sn.created_at < ?`
		nasabahArgs = append(nasabahArgs, startDate, endDate)
	}
	nasabahQuery += ` GROUP BY ks.sampah_id, s.nama_sampah, s.satuan, r.nama_reward, k.kategori`

	var hasilNasabah []StatistikItem
	if err := sc.DB.Raw(nasabahQuery, nasabahArgs...).Scan(&hasilNasabah).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data setoran: " + err.Error()})
		return
	}

	for _, item := range hasilNasabah {
		addToAggregated(item)
	}

	// Setoran dari BSU via paket pengangkutan (khusus BSI)
	if bank.JenisBank == models.BSI {
		paketQuery := `
			SELECT
				ks.sampah_id,
				s.nama_sampah,
				s.satuan,
				r.nama_reward AS jenis_reward,
				k.kategori,
				SUM(dp.qty) AS total_qty
			FROM pengangkutan_sampah ps
			JOIN detail_pengangkutan dp ON dp.pengangkutan_id = ps.pengangkutan_id
			JOIN katalog_sampah ks ON ks.sampah_id = dp.sampah_id
			JOIN sampah s ON s.sarok_id = ks.sarok_id
			JOIN reward r ON r.reward_id = ks.reward_id
			JOIN kategori_sampah k ON k.kategori_id = ks.kategori_id
			JOIN riwayat_pengangkutan rp ON rp.pengangkutan_id = ps.pengangkutan_id
				AND rp.status_pengangkutan = 'completed'
			WHERE ps.bsi_id = ?`
		paketArgs := []interface{}{bankID}
		if filterTanggal {
			paketQuery += ` AND rp.changed_at >= ? AND rp.changed_at < ?`
			paketArgs = append(paketArgs, startDate, endDate)
		}
		paketQuery += ` GROUP BY ks.sampah_id, s.nama_sampah, s.satuan, r.nama_reward, k.kategori`

		var hasilPaket []StatistikItem
		if err := sc.DB.Raw(paketQuery, paketArgs...).Scan(&hasilPaket).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data paket setoran: " + err.Error()})
			return
		}

		for _, item := range hasilPaket {
			addToAggregated(item)
		}
	}

	results := make([]StatistikItem, 0, len(aggregated))
	for _, item := range aggregated {
		results = append(results, item)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Satuan != results[j].Satuan {
			return results[i].Satuan < results[j].Satuan
		}
		if results[i].JenisReward != results[j].JenisReward {
			return results[i].JenisReward < results[j].JenisReward
		}
		return results[i].Kategori < results[j].Kategori
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Statistik setoran sampah berhasil diambil",
		"bank": gin.H{
			"bank_id":    bank.BankID,
			"nama_bank":  bank.NamaBank,
			"jenis_bank": bank.JenisBank,
		},
		"data": results,
	})
}

// GET /statistik/superadmin/ringkasan
func (sc *StatistikController) GetRingkasanSuperadmin(c *gin.Context) {
	// ── Bank per jenis + status aktif ────────────────────────────────────────
	type bankStatRow struct {
		JenisBank string `gorm:"column:jenis_bank"`
		IsActive  bool   `gorm:"column:is_active"`
		Count     int64  `gorm:"column:count"`
	}
	var bankRows []bankStatRow
	if err := sc.DB.Model(&models.BankSampah{}).
		Select("jenis_bank, is_active, COUNT(*) as count").
		Group("jenis_bank, is_active").
		Scan(&bankRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data bank: " + err.Error()})
		return
	}

	type bankStat struct {
		Aktif    int64 `json:"aktif"`
		Nonaktif int64 `json:"nonaktif"`
		Total    int64 `json:"total"`
	}
	bankStats := map[string]*bankStat{
		string(models.BSI): {},
		string(models.BSM): {},
		string(models.BSU): {},
	}
	for _, r := range bankRows {
		s := bankStats[r.JenisBank]
		if r.IsActive {
			s.Aktif += r.Count
		} else {
			s.Nonaktif += r.Count
		}
		s.Total += r.Count
	}

	// ── Nasabah per status ────────────────────────────────────────────────────
	type nasabahStatRow struct {
		Status string `gorm:"column:status"`
		Count  int64  `gorm:"column:count"`
	}
	var nasabahRows []nasabahStatRow
	if err := sc.DB.Model(&models.Nasabah{}).
		Select("status_nasabah as status, COUNT(*) as count").
		Group("status_nasabah").
		Scan(&nasabahRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data nasabah: " + err.Error()})
		return
	}

	nasabahStats := map[string]int64{
		string(models.Aktif):    0,
		string(models.Pending):  0,
		string(models.Nonaktif): 0,
	}
	var totalNasabah int64
	for _, r := range nasabahRows {
		nasabahStats[r.Status] = r.Count
		totalNasabah += r.Count
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Ringkasan statistik berhasil diambil",
		"bank": gin.H{
			"bsi": bankStats[string(models.BSI)],
			"bsm": bankStats[string(models.BSM)],
			"bsu": bankStats[string(models.BSU)],
		},
		"nasabah": gin.H{
			"aktif":    nasabahStats[string(models.Aktif)],
			"pending":  nasabahStats[string(models.Pending)],
			"nonaktif": nasabahStats[string(models.Nonaktif)],
			"total":    totalNasabah,
		},
	})
}

// GET /statistik/superadmin/tren-penjualan?tahun=2026
func (sc *StatistikController) GetTrenPenjualan(c *gin.Context) {
	tahunStr := c.Query("tahun")
	tahun := time.Now().Year()
	if tahunStr != "" {
		if t, err := strconv.Atoi(tahunStr); err == nil && t >= 2000 {
			tahun = t
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter tahun tidak valid"})
			return
		}
	}

	type trenRow struct {
		Bulan  int     `gorm:"column:bulan"`
		Satuan string  `gorm:"column:satuan"`
		Total  float64 `gorm:"column:total"`
	}
	var rows []trenRow
	if err := sc.DB.Raw(`
		SELECT
			EXTRACT(MONTH FROM created_at)::int AS bulan,
			satuan_reward AS satuan,
			SUM(total_penjualan) AS total
		FROM penjualan
		WHERE EXTRACT(YEAR FROM created_at) = ?
		GROUP BY bulan, satuan_reward
		ORDER BY bulan ASC
	`, tahun).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil tren penjualan: " + err.Error()})
		return
	}

	// Susun ke 12 bulan, isi 0 untuk bulan yang tidak ada data
	type bulanData struct {
		Bulan    int     `json:"bulan"`
		Uang     float64 `json:"uang"`
		Sembako  float64 `json:"sembako"`
	}
	tren := make([]bulanData, 12)
	for i := range tren {
		tren[i].Bulan = i + 1
	}
	for _, r := range rows {
		if r.Bulan < 1 || r.Bulan > 12 {
			continue
		}
		switch r.Satuan {
		case string(models.SatuanRewardEnumRp):
			tren[r.Bulan-1].Uang = r.Total
		case string(models.SatuanRewardEnumPoin):
			tren[r.Bulan-1].Sembako = r.Total
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tren penjualan berhasil diambil",
		"tahun":   tahun,
		"data":    tren,
	})
}

// GET /statistik/superadmin/ranking-bank?tahun=2026&limit=10
func (sc *StatistikController) GetRankingBank(c *gin.Context) {
	limit := 10
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 {
		limit = l
	}

	tahunStr := c.Query("tahun")
	var tahunFilter int
	if tahunStr != "" {
		t, err := strconv.Atoi(tahunStr)
		if err != nil || t < 2000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter tahun tidak valid"})
			return
		}
		tahunFilter = t
	}

	type rankingRow struct {
		BankID          string  `gorm:"column:bank_id"`
		NamaBank        string  `gorm:"column:nama_bank"`
		JenisBank       string  `gorm:"column:jenis_bank"`
		TotalUang       float64 `gorm:"column:total_uang"`
		TotalSembako    float64 `gorm:"column:total_sembako"`
		JumlahPenjualan int     `gorm:"column:jumlah_penjualan"`
	}

	query := `
		SELECT
			p.bank_id,
			bs.nama_bank,
			bs.jenis_bank,
			SUM(CASE WHEN p.satuan_reward = 'Rp'   THEN p.total_penjualan ELSE 0 END) AS total_uang,
			SUM(CASE WHEN p.satuan_reward = 'poin' THEN p.total_penjualan ELSE 0 END) AS total_sembako,
			COUNT(*) AS jumlah_penjualan
		FROM penjualan p
		JOIN bank_sampah bs ON bs.bank_id = p.bank_id`

	args := []interface{}{}
	if tahunFilter > 0 {
		query += ` WHERE EXTRACT(YEAR FROM p.created_at) = ?`
		args = append(args, tahunFilter)
	}
	query += `
		GROUP BY p.bank_id, bs.nama_bank, bs.jenis_bank
		ORDER BY total_uang DESC, total_sembako DESC
		LIMIT ?`
	args = append(args, limit)

	var ranking []rankingRow
	if err := sc.DB.Raw(query, args...).Scan(&ranking).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil ranking bank: " + err.Error()})
		return
	}

	label := "all-time"
	if tahunFilter > 0 {
		label = strconv.Itoa(tahunFilter)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Ranking bank berhasil diambil",
		"periode": label,
		"data":    ranking,
	})
}

// GET /statistik/penjualan-sampah/:bank_id
//
// Query params (optional):
//
//	bulan_mulai, tahun_mulai, bulan_selesai, tahun_selesai
func (sc *StatistikController) GetStatistikPenjualanSampahBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := sc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}
	if bank.JenisBank == models.BSU {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BSU tidak memiliki data penjualan"})
		return
	}

	var startDate, endDate time.Time
	var filterTanggal bool

	bmStr := c.Query("bulan_mulai")
	tmStr := c.Query("tahun_mulai")
	bsStr := c.Query("bulan_selesai")
	tsStr := c.Query("tahun_selesai")

	if bmStr != "" || tmStr != "" || bsStr != "" || tsStr != "" {
		bm, e1 := strconv.Atoi(bmStr)
		tm, e2 := strconv.Atoi(tmStr)
		bs, e3 := strconv.Atoi(bsStr)
		ts, e4 := strconv.Atoi(tsStr)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil ||
			bm < 1 || bm > 12 || bs < 1 || bs > 12 || tm < 2000 || ts < 2000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter bulan_mulai, tahun_mulai, bulan_selesai, tahun_selesai tidak valid"})
			return
		}
		startDate = time.Date(tm, time.Month(bm), 1, 0, 0, 0, 0, time.UTC)
		endDate = time.Date(ts, time.Month(bs+1), 1, 0, 0, 0, 0, time.UTC)
		if endDate.Before(startDate) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Rentang tanggal tidak valid: bulan_selesai lebih awal dari bulan_mulai"})
			return
		}
		filterTanggal = true
	}

	type penjualanSampahItem struct {
		SampahID    string  `gorm:"column:sampah_id"    json:"sampah_id"`
		NamaSampah  string  `gorm:"column:nama_sampah"  json:"nama_sampah"`
		Satuan      string  `gorm:"column:satuan"       json:"satuan"`
		Kategori    string  `gorm:"column:kategori"     json:"kategori"`
		JenisReward string  `gorm:"column:jenis_reward" json:"jenis_reward"`
		TotalQty    float64 `gorm:"column:total_qty"    json:"total_qty"`
		TotalNilai  float64 `gorm:"column:total_nilai"  json:"total_nilai"`
		SatuanNilai string  `gorm:"column:satuan_nilai" json:"satuan_nilai"`
	}

	query := `
		SELECT
			ks.sampah_id,
			s.nama_sampah,
			s.satuan,
			k.kategori,
			r.nama_reward  AS jenis_reward,
			SUM(dp.qty)                   AS total_qty,
			SUM(dp.subtotal_penjualan)    AS total_nilai,
			p.satuan_reward               AS satuan_nilai
		FROM detail_penjualan dp
		JOIN penjualan p     ON p.penjualan_id = dp.penjualan_id
		JOIN katalog_sampah ks ON ks.sampah_id = dp.sampah_id
		JOIN sampah s ON s.sarok_id = ks.sarok_id
		JOIN reward r          ON r.reward_id  = ks.reward_id
		JOIN kategori_sampah k ON k.kategori_id = ks.kategori_id
		WHERE p.bank_id = ?`

	args := []interface{}{bankID}
	if filterTanggal {
		query += ` AND p.created_at >= ? AND p.created_at < ?`
		args = append(args, startDate, endDate)
	}
	query += `
		GROUP BY ks.sampah_id, s.nama_sampah, s.satuan, k.kategori, r.nama_reward, p.satuan_reward
		ORDER BY total_qty DESC`

	var hasil []penjualanSampahItem
	if err := sc.DB.Raw(query, args...).Scan(&hasil).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil statistik penjualan: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Statistik penjualan sampah berhasil diambil",
		"bank": gin.H{
			"bank_id":    bank.BankID,
			"nama_bank":  bank.NamaBank,
			"jenis_bank": bank.JenisBank,
		},
		"data": hasil,
	})
}

// GET /statistik/masuk-sampah/:bank_id
//
// Query params (optional):
//
//	bulan_mulai, tahun_mulai, bulan_selesai, tahun_selesai
func (sc *StatistikController) GetStatistikMasukSampahBSU(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := sc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}
	if bank.JenisBank != models.BSU {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Endpoint ini hanya tersedia untuk BSU"})
		return
	}

	var startDate, endDate time.Time
	var filterTanggal bool

	bmStr := c.Query("bulan_mulai")
	tmStr := c.Query("tahun_mulai")
	bsStr := c.Query("bulan_selesai")
	tsStr := c.Query("tahun_selesai")

	if bmStr != "" || tmStr != "" || bsStr != "" || tsStr != "" {
		bm, e1 := strconv.Atoi(bmStr)
		tm, e2 := strconv.Atoi(tmStr)
		bs, e3 := strconv.Atoi(bsStr)
		ts, e4 := strconv.Atoi(tsStr)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil ||
			bm < 1 || bm > 12 || bs < 1 || bs > 12 || tm < 2000 || ts < 2000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter bulan_mulai, tahun_mulai, bulan_selesai, tahun_selesai tidak valid"})
			return
		}
		startDate = time.Date(tm, time.Month(bm), 1, 0, 0, 0, 0, time.UTC)
		endDate = time.Date(ts, time.Month(bs+1), 1, 0, 0, 0, 0, time.UTC)
		if endDate.Before(startDate) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Rentang tanggal tidak valid: bulan_selesai lebih awal dari bulan_mulai"})
			return
		}
		filterTanggal = true
	}

	type masukItem struct {
		SampahID    string  `gorm:"column:sampah_id"    json:"sampah_id"`
		NamaSampah  string  `gorm:"column:nama_sampah"  json:"nama_sampah"`
		Satuan      string  `gorm:"column:satuan"       json:"satuan"`
		Kategori    string  `gorm:"column:kategori"     json:"kategori"`
		TotalMasuk  float64 `gorm:"column:total_masuk"  json:"total_masuk"`
		StokTersisa float64 `gorm:"column:stok_tersisa" json:"stok_tersisa"`
	}

	query := `
		SELECT
			ks.sampah_id,
			s.nama_sampah,
			s.satuan,
			k.kategori,
			SUM(ts.qty)      AS total_masuk,
			SUM(ts.sisa_qty) AS stok_tersisa
		FROM tabungan_sampah ts
		JOIN katalog_sampah ks  ON ks.sampah_id  = ts.sampah_id
		JOIN sampah s ON s.sarok_id = ks.sarok_id
		JOIN kategori_sampah k  ON k.kategori_id = ks.kategori_id
		WHERE ts.bank_id = ? AND ts.entitas = ?`

	args := []interface{}{bankID, models.EntitasBankSampah}
	if filterTanggal {
		query += ` AND ts.created_at >= ? AND ts.created_at < ?`
		args = append(args, startDate, endDate)
	}
	query += `
		GROUP BY ks.sampah_id, s.nama_sampah, s.satuan, k.kategori
		ORDER BY total_masuk DESC`

	var hasil []masukItem
	if err := sc.DB.Raw(query, args...).Scan(&hasil).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil statistik masuk sampah: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Statistik masuk sampah BSU berhasil diambil",
		"bank": gin.H{
			"bank_id":   bank.BankID,
			"nama_bank": bank.NamaBank,
		},
		"data": hasil,
	})
}

// GET /statistik/superadmin/volume-sampah?bulan=3&tahun=2026
func (sc *StatistikController) GetVolumeSampahNasabah(c *gin.Context) {
	type VolumeRow struct {
		Bulan    int     `gorm:"column:bulan"     json:"bulan"`
		Tahun    int     `gorm:"column:tahun"     json:"tahun"`
		Satuan   string  `gorm:"column:satuan"    json:"satuan"`
		TotalQty float64 `gorm:"column:total_qty" json:"total_qty"`
	}

	query := `
		SELECT
			EXTRACT(MONTH FROM ts.created_at)::int AS bulan,
			EXTRACT(YEAR  FROM ts.created_at)::int AS tahun,
			s.satuan,
			SUM(ts.qty) AS total_qty
		FROM tabungan_sampah ts
		JOIN katalog_sampah ks ON ks.sampah_id = ts.sampah_id
		JOIN sampah s          ON s.sarok_id   = ks.sarok_id
		WHERE ts.entitas = ?`
	args := []interface{}{models.EntitasNasabah}

	if bulan, err := strconv.Atoi(c.Query("bulan")); err == nil && bulan >= 1 && bulan <= 12 {
		query += ` AND EXTRACT(MONTH FROM ts.created_at) = ?`
		args = append(args, bulan)
	}
	if tahun, err := strconv.Atoi(c.Query("tahun")); err == nil && tahun >= 2000 {
		query += ` AND EXTRACT(YEAR FROM ts.created_at) = ?`
		args = append(args, tahun)
	}

	query += `
		GROUP BY tahun, bulan, s.satuan
		ORDER BY tahun ASC, bulan ASC, s.satuan ASC`

	var results []VolumeRow
	if err := sc.DB.Raw(query, args...).Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil volume sampah: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Volume sampah nasabah berhasil diambil",
		"data":    results,
	})
}

func (sc *StatistikController) GetKontribusiNasabah(c *gin.Context) {
	bankID := c.Param("bank_id")

	// ── Date filter ──────────────────────────────────────────────────────────
	var startDate, endDate time.Time
	var filterTanggal bool

	bmStr := c.Query("bulan_mulai")
	tmStr := c.Query("tahun_mulai")
	bsStr := c.Query("bulan_selesai")
	tsStr := c.Query("tahun_selesai")

	if bmStr != "" || tmStr != "" || bsStr != "" || tsStr != "" {
		bm, e1 := strconv.Atoi(bmStr)
		tm, e2 := strconv.Atoi(tmStr)
		bs, e3 := strconv.Atoi(bsStr)
		ts, e4 := strconv.Atoi(tsStr)
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil ||
			bm < 1 || bm > 12 || bs < 1 || bs > 12 || tm < 2000 || ts < 2000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter bulan_mulai, tahun_mulai, bulan_selesai, tahun_selesai tidak valid"})
			return
		}
		startDate = time.Date(tm, time.Month(bm), 1, 0, 0, 0, 0, time.UTC)
		endDate = time.Date(ts, time.Month(bs+1), 1, 0, 0, 0, 0, time.UTC)
		if endDate.Before(startDate) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Rentang tanggal tidak valid: bulan_selesai lebih awal dari bulan_mulai"})
			return
		}
		filterTanggal = true
	}

	// ── Sort & pagination ────────────────────────────────────────────────────
	validSorts := map[string]string{
		"setoran": "jumlah_setoran",
		"kg":      "total_kg",
		"pcs":     "total_pcs",
		"liter":   "total_liter",
	}
	sortBy := c.DefaultQuery("sort_by", "setoran")
	sortCol, ok := validSorts[sortBy]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sort_by tidak valid. Gunakan: setoran, kg, pcs, liter"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	// ── Bank validation ──────────────────────────────────────────────────────
	var bank models.BankSampah
	if err := sc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	// ── Build subqueries ─────────────────────────────────────────────────────
	selectCols := `
		n.nasabah_id,
		u.nama AS nama_nasabah,
		n.bank_id,
		b.nama_bank,
		COUNT(DISTINCT sn.setoran_id) AS jumlah_setoran,
		COALESCE(SUM(CASE WHEN s.satuan = 'kg'    THEN dsn.qty ELSE 0 END), 0) AS total_kg,
		COALESCE(SUM(CASE WHEN s.satuan = 'pcs'   THEN dsn.qty ELSE 0 END), 0) AS total_pcs,
		COALESCE(SUM(CASE WHEN s.satuan = 'liter' THEN dsn.qty ELSE 0 END), 0) AS total_liter`

	joinClause := `
		FROM setoran_nasabah sn
		JOIN nasabah n ON sn.nasabah_id = n.nasabah_id
		JOIN users u ON n.user_id = u.user_id
		JOIN bank_sampah b ON n.bank_id = b.bank_id
		JOIN detail_setoran_nasabah dsn ON sn.setoran_id = dsn.setoran_id
		JOIN katalog_sampah ks ON dsn.sampah_id = ks.sampah_id
		JOIN sampah s ON s.sarok_id = ks.sarok_id`

	groupBy := ` GROUP BY n.nasabah_id, u.nama, n.bank_id, b.nama_bank`

	dateFilter := ""
	dateArgs := []interface{}{}
	if filterTanggal {
		dateFilter = ` AND sn.created_at >= ? AND sn.created_at < ?`
		dateArgs = []interface{}{startDate, endDate}
	}

	// q1: nasabah langsung bank ini
	q1 := fmt.Sprintf(`SELECT %s %s JOIN admin a ON sn.admin_id = a.admin_id WHERE sn.status_setoran = 'berhasil' AND a.bank_id = ?%s%s`,
		selectCols, joinClause, dateFilter, groupBy)
	args1 := append([]interface{}{bankID}, dateArgs...)

	// q2: nasabah BSU di bawah BSI (khusus BSI)
	q2 := ""
	args2 := []interface{}{}
	if bank.JenisBank == models.BSI {
		q2 = fmt.Sprintf(`SELECT %s %s WHERE sn.status_setoran = 'berhasil' AND b.parent_bank_id = ? AND b.jenis_bank = 'bsu'%s%s`,
			selectCols, joinClause, dateFilter, groupBy)
		args2 = append([]interface{}{bankID}, dateArgs...)
	}

	// ── Paginated data query ─────────────────────────────────────────────────
	type KontribusiItem struct {
		NasabahID     string  `gorm:"column:nasabah_id" json:"nasabah_id"`
		NamaNasabah   string  `gorm:"column:nama_nasabah" json:"nama_nasabah"`
		BankID        string  `gorm:"column:bank_id" json:"bank_id"`
		NamaBank      string  `gorm:"column:nama_bank" json:"nama_bank"`
		JumlahSetoran int     `gorm:"column:jumlah_setoran" json:"jumlah_setoran"`
		TotalKg       float64 `gorm:"column:total_kg" json:"total_kg"`
		TotalPcs      float64 `gorm:"column:total_pcs" json:"total_pcs"`
		TotalLiter    float64 `gorm:"column:total_liter" json:"total_liter"`
		TotalCount    int     `gorm:"column:total_count" json:"-"`
	}

	var dataQuery string
	var dataArgs []interface{}
	if q2 != "" {
		dataQuery = fmt.Sprintf(`
			WITH combined AS (%s UNION ALL %s)
			SELECT *, COUNT(*) OVER() AS total_count
			FROM combined
			ORDER BY %s DESC
			LIMIT ? OFFSET ?`, q1, q2, sortCol)
		dataArgs = append(append(args1, args2...), limit, offset)
	} else {
		dataQuery = fmt.Sprintf(`
			WITH combined AS (%s)
			SELECT *, COUNT(*) OVER() AS total_count
			FROM combined
			ORDER BY %s DESC
			LIMIT ? OFFSET ?`, q1, sortCol)
		dataArgs = append(args1, limit, offset)
	}

	var hasil []KontribusiItem
	if err := sc.DB.Raw(dataQuery, dataArgs...).Scan(&hasil).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data kontribusi: " + err.Error()})
		return
	}

	// ── Bank totals (untuk kalkulasi persentase di frontend) ─────────────────
	type BankTotals struct {
		SumKg    float64 `gorm:"column:sum_kg"`
		SumPcs   float64 `gorm:"column:sum_pcs"`
		SumLiter float64 `gorm:"column:sum_liter"`
	}

	// totalsSelectRaw: dipakai langsung di query dengan table alias (untuk non-BSI)
	totalsSelectRaw := `
		COALESCE(SUM(CASE WHEN s.satuan = 'kg'    THEN dsn.qty ELSE 0 END), 0) AS sum_kg,
		COALESCE(SUM(CASE WHEN s.satuan = 'pcs'   THEN dsn.qty ELSE 0 END), 0) AS sum_pcs,
		COALESCE(SUM(CASE WHEN s.satuan = 'liter' THEN dsn.qty ELSE 0 END), 0) AS sum_liter`

	// totalsSelectCTE: dipakai pada outer SELECT dari CTE combined (kolom tanpa alias tabel)
	totalsSelectCTE := `
		COALESCE(SUM(CASE WHEN satuan = 'kg'    THEN qty ELSE 0 END), 0) AS sum_kg,
		COALESCE(SUM(CASE WHEN satuan = 'pcs'   THEN qty ELSE 0 END), 0) AS sum_pcs,
		COALESCE(SUM(CASE WHEN satuan = 'liter' THEN qty ELSE 0 END), 0) AS sum_liter`

	var totalsQuery string
	var totalsArgs []interface{}
	if bank.JenisBank == models.BSI {
		totalsQuery = fmt.Sprintf(`
			WITH t1 AS (
				SELECT dsn.qty, s.satuan %s JOIN admin a ON sn.admin_id = a.admin_id
				WHERE sn.status_setoran = 'berhasil' AND a.bank_id = ?%s
			), t2 AS (
				SELECT dsn.qty, s.satuan %s
				WHERE sn.status_setoran = 'berhasil' AND b.parent_bank_id = ? AND b.jenis_bank = 'bsu'%s
			), combined AS (SELECT * FROM t1 UNION ALL SELECT * FROM t2)
			SELECT %s FROM combined`, joinClause, dateFilter, joinClause, dateFilter, totalsSelectCTE)
		totalsArgs = append(append([]interface{}{bankID}, dateArgs...), append([]interface{}{bankID}, dateArgs...)...)
	} else {
		totalsQuery = fmt.Sprintf(`
			SELECT %s %s JOIN admin a ON sn.admin_id = a.admin_id
			WHERE sn.status_setoran = 'berhasil' AND a.bank_id = ?%s`,
			totalsSelectRaw, joinClause, dateFilter)
		totalsArgs = append([]interface{}{bankID}, dateArgs...)
	}

	var totals BankTotals
	if err := sc.DB.Raw(totalsQuery, totalsArgs...).Scan(&totals).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil total kontribusi bank"})
		return
	}

	// ── Pagination metadata ──────────────────────────────────────────────────
	totalCount := 0
	if len(hasil) > 0 {
		totalCount = hasil[0].TotalCount
	}
	totalPages := (totalCount + limit - 1) / limit

	c.JSON(http.StatusOK, gin.H{
		"message": "Kontribusi nasabah berhasil diambil",
		"bank": gin.H{
			"bank_id":    bank.BankID,
			"nama_bank":  bank.NamaBank,
			"jenis_bank": bank.JenisBank,
		},
		"sort_by": sortBy,
		"totals": gin.H{
			"total_kg":    totals.SumKg,
			"total_pcs":   totals.SumPcs,
			"total_liter": totals.SumLiter,
		},
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       totalCount,
			"total_pages": totalPages,
		},
		"data": hasil,
	})
}