package controllers

import (
	"encoding/json"
	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PenjualanController struct {
	DB        *gorm.DB
	CFStorage *storage.CloudflareStorage
}

func NewPenjualanController(db *gorm.DB, cfStorage *storage.CloudflareStorage) *PenjualanController {
	return &PenjualanController{
		DB:        db,
		CFStorage: cfStorage,
	}
}

type HasilKalkulasiPenjualan struct {
	TotalPenjualan float64
	SatuanReward   models.SatuanRewardEnum
	PersenNasabah  float64
	DetailItems    []DetailKalkulasiItem
}

type DetailKalkulasiItem struct {
	SampahID             string
	NamaSampah           string
	Qty                  float64
	HargaJual            float64
	Subtotal             float64
	HargaNasabahSnapshot float64
}

// hitungPenjualan adalah pure kalkulasi — tidak menyentuh DB write sama sekali.
// db boleh berisi tx (untuk POST) atau pc.DB biasa (untuk GET preview).
func (pc *PenjualanController) hitungPenjualan(
	db *gorm.DB,
	bankID string,
	bank models.BankSampah,
	rewardID int,
	itemsSampah []ItemSampahDijual,
) (*HasilKalkulasiPenjualan, error) {

	// ── 1. Ambil persentase bagi hasil ───────────────────────────────────────
	var persenNasabah float64

	var nilaiNasabah models.NilaiRewardBank
	if err := db.Where("bank_id = ? AND reward_id = ? AND level_user = ? AND is_active = true",
		bankID, rewardID, models.LevelNasabah).First(&nilaiNasabah).Error; err == nil {
		persenNasabah = nilaiNasabah.PersenBagiHasil
	}

	// ── 2. Ambil reward ──────────────────────────────────────────────────────
	var reward models.Reward
	if err := db.Where("reward_id = ?", rewardID).First(&reward).Error; err != nil {
		return nil, fmt.Errorf("reward tidak ditemukan")
	}

	// ── 3. Proses setiap item ────────────────────────────────────────────────
	satuanReward := models.SatuanRewardEnumRp
	if reward.NamaReward == models.RewardEnumSembako {
		satuanReward = models.SatuanRewardEnumPoin
	}

	var totalPenjualan float64
	var detailItems []DetailKalkulasiItem

	for _, item := range itemsSampah {
		// Validasi sampah
		var sampah models.KatalogSampah
		if err := db.Where("sampah_id = ?", item.SampahID).First(&sampah).Error; err != nil {
			return nil, fmt.Errorf("sampah tidak ditemukan: %s", item.SampahID)
		}

		// Validasi stok
		var stok models.StokSampah
		if err := db.Where("bank_id = ? AND sampah_id = ?", bankID, item.SampahID).First(&stok).Error; err != nil {
			return nil, fmt.Errorf("stok sampah tidak ada di bank: %s", sampah.NamaSampah)
		}
		if stok.Stok < item.Qty {
			return nil, fmt.Errorf("stok %s tidak mencukupi (stok: %s, dibutuhkan: %s)",
				sampah.NamaSampah, formatFloat(stok.Stok), formatFloat(item.Qty))
		}



		subtotal := item.HargaJual * item.Qty
		totalPenjualan += subtotal

		detailItems = append(detailItems, DetailKalkulasiItem{
			SampahID:             item.SampahID,
			NamaSampah:           sampah.NamaSampah,
			Qty:                  item.Qty,
			HargaJual:            item.HargaJual,
			Subtotal:             subtotal,
			HargaNasabahSnapshot: item.HargaJual * (persenNasabah / 100),
		})
	}

	return &HasilKalkulasiPenjualan{
		TotalPenjualan: totalPenjualan,
		SatuanReward:   satuanReward,
		PersenNasabah:  persenNasabah,
		DetailItems:    detailItems,
	}, nil
}

func (pc *PenjualanController) PreviewPenjualanEksternal(c *gin.Context) {
	bankID := c.Param("bank_id")

	// ── 1. Parse form ────────────────────────────────────────────────────────
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal membaca form data"})
		return
	}

	rewardIDStr := c.PostForm("reward_id")
	var rewardID int
	if _, err := fmt.Sscan(rewardIDStr, &rewardID); err != nil || rewardID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reward_id tidak valid"})
		return
	}

	itemsSampahStr := c.PostForm("items_sampah")
	var itemsSampah []ItemSampahDijual
	if err := json.Unmarshal([]byte(itemsSampahStr), &itemsSampah); err != nil || len(itemsSampah) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format items_sampah tidak valid atau kosong"})
		return
	}
	for _, item := range itemsSampah {
		if item.HargaJual <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "harga_jual untuk setiap item harus lebih dari 0"})
			return
		}
	}

	// ── 2. Validasi bank ─────────────────────────────────────────────────────
	var bank models.BankSampah
	if err := pc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}
	if bank.JenisBank == models.BSU {
		c.JSON(http.StatusForbidden, gin.H{"error": "BSU tidak diizinkan melakukan penjualan eksternal. Hanya BSI/BSM."})
		return
	}

	// ── 3. Hitung kalkulasi (read-only, tidak ada DB write) ──────────────────
	hasil, err := pc.hitungPenjualan(pc.DB, bankID, bank, rewardID, itemsSampah)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ── 4. Susun response preview ────────────────────────────────────────────
	response := gin.H{
		"total_penjualan": hasil.TotalPenjualan,
		"satuan":          hasil.SatuanReward,
		"persen_nasabah":  hasil.PersenNasabah,
		"detail_items":    hasil.DetailItems,
	}

	c.JSON(http.StatusOK, response)
}

// ─── Input Structs ────────────────────────────────────────────────────────────

// ItemSampahDijual mewakili satu item sampah dalam penjualan.
// HargaJual diisi oleh admin berdasarkan harga kesepakatan dengan pembeli eksternal.
// Sistem akan mencocokkan nilai ini dengan schema_harga_sampah level "eksternal":
//   - Jika sama  → tidak ada perubahan schema
//   - Jika beda  → schema di-update + history perubahan harga dicatat
//
// Snapshot harga nasabah dihitung dari persentase di nilai_reward_bank:
//   harga_nasabah_snapshot = harga_jual × (persen_nasabah / 100)
type ItemSampahDijual struct {
	SampahID  string  `json:"sampah_id" binding:"required"`
	Qty       float64 `json:"qty" binding:"required"`
	HargaJual float64 `json:"harga_jual" binding:"required"` // harga jual ke pembeli eksternal (per satuan)
}

type InputPenjualan struct {
	RewardID         int                `json:"reward_id" binding:"required"` // jenis reward untuk menentukan persentase bagi hasil
	IdentitasPembeli string             `json:"identitas_pembeli" binding:"required"`
	ItemsSampah      []ItemSampahDijual `json:"items_sampah" binding:"required"`
}

// ─── AddNewPenjualanEksternal ─────────────────────────────────────────────────
// POST /penjualan/add-eksternal/:bank_id/:admin_id
//
// Alur per item sampah:
//  1. Validasi stok cukup
//  2. Cek schema_harga_sampah level "eksternal":
//     - Jika harga berubah → update schema + catat history
//     - Jika sama          → skip update
//  3. Baca snapshot harga "nasabah" dari schema_harga_sampah saat ini
//     (digunakan nanti saat proses bagi hasil)
//  4. Kurangi stok sampah
//  5. Simpan Penjualan + DetailPenjualan
//  6. status_bagi_hasil = "pending" (bagi hasil diproses via endpoint terpisah)
func (pc *PenjualanController) AddNewPenjualanEksternal(c *gin.Context) {
	bankID := c.Param("bank_id")
	adminID := c.Param("admin_id")

	// ── 1. Parse multipart form ──────────────────────────────────────────────
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal membaca form data"})
		return
	}

	rewardIDStr := c.PostForm("reward_id")
	var rewardID int
	if _, err := fmt.Sscan(rewardIDStr, &rewardID); err != nil || rewardID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reward_id tidak valid"})
		return
	}

	identitasPembeli := c.PostForm("identitas_pembeli")
	if identitasPembeli == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Identitas pembeli wajib diisi"})
		return
	}

	itemsSampahStr := c.PostForm("items_sampah")
	var itemsSampah []ItemSampahDijual
	if err := json.Unmarshal([]byte(itemsSampahStr), &itemsSampah); err != nil || len(itemsSampah) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format items_sampah tidak valid atau kosong"})
		return
	}
	for _, item := range itemsSampah {
		if item.HargaJual <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "harga_jual untuk setiap item harus lebih dari 0"})
			return
		}
	}

	// ── 2. Upload bukti foto ─────────────────────────────────────────────────
	fileHeader, err := c.FormFile("bukti_foto")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bukti foto wajib diupload"})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca file foto"})
		return
	}
	defer file.Close()

	buktiFotoURL, err := pc.CFStorage.UploadFile(file, fileHeader, "penjualan_eksternal")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal upload bukti foto: " + err.Error()})
		return
	}

	// ── 3. Validasi bank (hanya BSI/BSM) ─────────────────────────────────────
	var bank models.BankSampah
	if err := pc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}
	if bank.JenisBank == models.BSU {
		c.JSON(http.StatusForbidden, gin.H{"error": "BSU tidak diizinkan melakukan penjualan eksternal. Hanya BSI/BSM."})
		return
	}

	// ── 3b. Ambil persentase bagi hasil dari nilai_reward_bank ────────────────
	// Snapshot disesuaikan dengan jenis bank yang melakukan penjualan
	var persenNasabah float64 = 0

	var nilaiNasabah models.NilaiRewardBank
	if err := pc.DB.
		Where("bank_id = ? AND reward_id = ? AND level_user = ? AND is_active = true", bankID, rewardID, models.LevelNasabah).
		First(&nilaiNasabah).Error; err == nil {
		persenNasabah = nilaiNasabah.PersenBagiHasil
	}

	// ── 3c. Ambil data Reward ────────────────────────────────────────────────
	var reward models.Reward
	if err := pc.DB.Where("reward_id = ?", rewardID).First(&reward).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reward tidak ditemukan"})
		return
	}

	satuanReward := models.SatuanRewardEnumRp
	if reward.NamaReward == models.RewardEnumSembako {
		satuanReward = models.SatuanRewardEnumPoin
	}

	// ── 4. Mulai transaksi DB ─────────────────────────────────────────────────
	tx := pc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
		return
	}

	penjualanID := utils.GenerateBankRelatedID(bankID)
	var totalPenjualan float64 = 0
	var detailPenjualans []models.DetailPenjualan

	// ── 5. Proses setiap item sampah ──────────────────────────────────────────
	for _, item := range itemsSampah {

		// 5a. Ambil data katalog sampah
		var sampah models.KatalogSampah
		if err := tx.Where("sampah_id = ?", item.SampahID).First(&sampah).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Sampah tidak ditemukan: " + item.SampahID})
			return
		}

		// 5b. Validasi stok
		var stok models.StokSampah
		if err := tx.Where("bank_id = ? AND sampah_id = ?", bankID, item.SampahID).First(&stok).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Stok sampah tidak ada di bank: " + sampah.NamaSampah})
			return
		}
		if stok.Stok < item.Qty {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Stok %s tidak mencukupi (stok: %s, dibutuhkan: %s)",
					sampah.NamaSampah, formatFloat(stok.Stok), formatFloat(item.Qty)),
			})
			return
		}

		// 5c. Cek schema harga EKSTERNAL → update jika harga berubah
		var schemaEksternal models.SchemaHargaSampah
		errSchema := tx.Where("sampah_id = ? AND level_user = ?", item.SampahID, models.LevelEksternal).
			First(&schemaEksternal).Error

		if errSchema != nil {
			// Schema belum ada → buat baru
			schemaEksternal = models.SchemaHargaSampah{
				SampahID:     item.SampahID,
				LevelUser:    models.LevelEksternal,
				Harga:        item.HargaJual,
				SatuanReward: models.SatuanRewardEnum(sampah.Reward.Satuan),
			}

			// Re-load reward untuk dapat SatuanReward
			var rewardSampah models.Reward
			if err := tx.Where("reward_id = ?", sampah.RewardID).First(&rewardSampah).Error; err == nil {
				schemaEksternal.SatuanReward = models.SatuanRewardEnum(rewardSampah.Satuan)
			}

			if err := tx.Create(&schemaEksternal).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat schema harga: " + err.Error()})
				return
			}
		} else if schemaEksternal.Harga != item.HargaJual {
			// Schema sudah ada tapi harga BERUBAH → catat history dulu, lalu update
			history := models.KatalogSampahHistory{
				SchemaID:  int(schemaEksternal.SchemaID),
				HargaLama: schemaEksternal.Harga,
				HargaBaru: item.HargaJual,
				ChangedBy: adminID,
			}
			if err := tx.Create(&history).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencatat history harga: " + err.Error()})
				return
			}

			schemaEksternal.Harga = item.HargaJual
			if err := tx.Save(&schemaEksternal).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update schema harga: " + err.Error()})
				return
			}
		}
		// Jika harga sama → tidak ada yang perlu dilakukan



		// 5d. Hitung snapshot harga NASABAH berdasarkan persentase bagi hasil
		// Rumus: harga_snapshot = harga_jual × (persen / 100)
		hargaNasabahSnapshot := item.HargaJual * (persenNasabah / 100)

		// 5f. Kurangi stok
		if err := tx.Model(&models.StokSampah{}).
			Where("bank_id = ? AND sampah_id = ?", bankID, item.SampahID).
			Update("stok", stok.Stok-item.Qty).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengurangi stok sampah"})
			return
		}

		subtotal := item.HargaJual * item.Qty
		totalPenjualan += subtotal

		detailPenjualans = append(detailPenjualans, models.DetailPenjualan{
			PenjualanID:          penjualanID,
			SampahID:             item.SampahID,
			Qty:                  item.Qty,
			HargaJual:            item.HargaJual,
			SubtotalPenjualan:    subtotal,
			HargaNasabahSnapshot: hargaNasabahSnapshot,
		})
	}

	// ── 6. Simpan record Penjualan ────────────────────────────────────────────
	penjualan := models.Penjualan{
		PenjualanID:      penjualanID,
		BankID:           bankID,
		RewardID:         rewardID,
		IdentitasPembeli: identitasPembeli,
		TotalItem:        len(itemsSampah),
		TotalPenjualan:   totalPenjualan,
		SatuanReward:     satuanReward,
		SoldBy:           adminID,
		BuktiFoto:        buktiFotoURL,
		CreatedAt:        time.Now(),
		StatusBagiHasil:  models.BagiHasilPending,
	}
	if err := tx.Create(&penjualan).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data penjualan: " + err.Error()})
		return
	}

	// ── 7. Simpan Detail Penjualan ────────────────────────────────────────────
	if err := tx.Create(&detailPenjualans).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan detail penjualan: " + err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal commit transaksi: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "Penjualan eksternal berhasil dicatat",
		"penjualan_id":      penjualanID,
		"total_penjualan":   totalPenjualan,
		"satuan":            satuanReward,
		"status_bagi_hasil": models.BagiHasilPending,
	})
}

// ─── Helper: satuan saldo berdasarkan jenis reward ────────────────────────────

// satuanSaldoNominal mengembalikan satuan untuk NominalSaldo sesuai jenis reward
// (Rp untuk Uang, gram untuk Emas, poin untuk Sembako/Poin)
func satuanSaldoNominal(r models.Reward) models.SatuanRewardEnum {
	if r.NamaReward == models.RewardEnumUang {
		return models.SatuanRewardEnumRp
	}
	return models.SatuanRewardEnumPoin
}

func satuanSaldoSisa(_ models.Reward) models.SatuanRewardEnum {
	return models.SatuanRewardEnumRp
}

// ─── GetRiwayatPenjualanEksternal ─────────────────────────────────────────────

// GET /penjualan/riwayat-eksternal/:bank_id
func (pc *PenjualanController) GetRiwayatPenjualanEksternal(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := pc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}
	if bank.JenisBank != models.BSI && bank.JenisBank != models.BSM {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur ini hanya untuk BSI dan BSM"})
		return
	}

	type RiwayatResponse struct {
		PenjualanID      string                      `json:"penjualan_id"`
		IdentitasPembeli string                      `json:"identitas_pembeli"`
		TotalItem        int                         `json:"total_item"`
		TotalPenjualan   float64                     `json:"total_penjualan"`
		NamaReward       string                      `json:"nama_reward"`
		SatuanReward     models.SatuanRewardEnum     `json:"satuan_reward"`
		BuktiFoto        string                      `json:"bukti_foto"`
		CreatedAt        time.Time                   `json:"created_at"`
		AdminName        string                      `json:"admin_name"`
		StatusBagiHasil  models.StatusBagiHasilEnum  `json:"status_bagi_hasil"`
	}

	var riwayats []RiwayatResponse
	if err := pc.DB.Table("penjualan").
		Select(`penjualan.penjualan_id, penjualan.identitas_pembeli,
		        penjualan.total_item, penjualan.total_penjualan, penjualan.satuan_reward,
		        penjualan.bukti_foto, penjualan.created_at, penjualan.status_bagi_hasil,
		        users.nama as admin_name, reward.nama_reward as nama_reward`).
		Joins("LEFT JOIN admin ON penjualan.sold_by = admin.admin_id").
		Joins("LEFT JOIN users ON admin.user_id = users.user_id").
		Joins("LEFT JOIN reward ON penjualan.reward_id = reward.reward_id").
		Where("penjualan.bank_id = ?", bankID).
		Order("penjualan.created_at DESC").
		Find(&riwayats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil riwayat penjualan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Riwayat penjualan berhasil diambil",
		"data":    riwayats,
	})
}

// ─── DetailPenjualanEksternal ─────────────────────────────────────────────────
// GET /penjualan/detail-eksternal/:penjualan_id
func (pc *PenjualanController) DetailPenjualanEksternal(c *gin.Context) {
	penjualanID := c.Param("penjualan_id")

	var penjualan models.Penjualan
	if err := pc.DB.Preload("BankSampah").Preload("Reward").
		Where("penjualan_id = ?", penjualanID).
		First(&penjualan).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data penjualan tidak ditemukan"})
		return
	}
	if penjualan.BankSampah.JenisBank != models.BSI && penjualan.BankSampah.JenisBank != models.BSM {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur ini hanya untuk BSI dan BSM"})
		return
	}

	// Ambil detail item sampah
	type SampahDetail struct {
		SampahID             string  `json:"sampah_id"`
		NamaSampah           string  `json:"nama_sampah"`
		Qty                  float64 `json:"qty"`
		HargaJual            float64 `json:"harga_jual"`
		SubtotalPenjualan    float64 `json:"subtotal_penjualan"`
		HargaNasabahSnapshot float64 `json:"harga_nasabah_snapshot"`
	}

	var rawDetails []models.DetailPenjualan
	pc.DB.Preload("Sampah").Where("penjualan_id = ?", penjualanID).Find(&rawDetails)

	var itemsSampah []SampahDetail
	for _, d := range rawDetails {
		itemsSampah = append(itemsSampah, SampahDetail{
			SampahID:             d.SampahID,
			NamaSampah:           d.Sampah.NamaSampah,
			Qty:                  d.Qty,
			HargaJual:            d.HargaJual,
			SubtotalPenjualan:    d.SubtotalPenjualan,
			HargaNasabahSnapshot: d.HargaNasabahSnapshot,
		})
	}

	// Ambil nama admin
	var adminName string
	pc.DB.Table("users").Select("users.nama").
		Joins("JOIN admin ON admin.user_id = users.user_id").
		Where("admin.admin_id = ?", penjualan.SoldBy).
		Scan(&adminName)

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail penjualan berhasil diambil",
		"data": gin.H{
			"penjualan_id":      penjualan.PenjualanID,
			"bank_id":           penjualan.BankID,
			"identitas_pembeli": penjualan.IdentitasPembeli,
			"total_item":        penjualan.TotalItem,
			"total_penjualan":   penjualan.TotalPenjualan,
			"satuan_reward":     penjualan.SatuanReward,
			"nama_reward":       penjualan.Reward.NamaReward,
			"bukti_foto":        penjualan.BuktiFoto,
			"created_at":        penjualan.CreatedAt,
			"admin_name":        adminName,
			"status_bagi_hasil": penjualan.StatusBagiHasil,
			"items_sampah":      itemsSampah,
		}, 
	})
}

// ─── GetListMitraPenjualan ────────────────────────────────────────────────────
// GET /penjualan/mitra/:bank_id
func (pc *PenjualanController) GetListMitraPenjualan(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := pc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}
	if bank.JenisBank != models.BSI && bank.JenisBank != models.BSM {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur ini hanya untuk BSI dan BSM"})
		return
	}

	var rawNames []string
	if err := pc.DB.Table("penjualan").
		Select("DISTINCT identitas_pembeli").
		Where("bank_id = ? AND identitas_pembeli IS NOT NULL AND identitas_pembeli != ''", bankID).
		Pluck("identitas_pembeli", &rawNames).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data mitra: " + err.Error()})
		return
	}

	// Normalisasi duplikat (case-insensitive + karakter non-alfanumerik)
	uniqueMap := make(map[string]string)
	reg := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	for _, name := range rawNames {
		key := reg.ReplaceAllString(strings.ToLower(name), "")
		if _, exists := uniqueMap[key]; !exists {
			uniqueMap[key] = name
		}
	}

	type MitraResponse struct {
		Nama string `json:"nama"`
	}
	var mitraList []MitraResponse
	for _, originalName := range uniqueMap {
		mitraList = append(mitraList, MitraResponse{Nama: originalName})
	}
	sort.Slice(mitraList, func(i, j int) bool {
		return mitraList[i].Nama < mitraList[j].Nama
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "List mitra berhasil diambil",
		"data":    mitraList,
	})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func formatFloat(f float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", f), "0"), ".")
}

