package controllers

import (
	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"net/http"

	"enviroo-be/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type KatalogController struct {
	DB        *gorm.DB
	CFStorage *storage.CloudflareStorage
}

func NewKatalogController(db *gorm.DB, cfStorage *storage.CloudflareStorage) *KatalogController {
	return &KatalogController{
		DB:        db,
		CFStorage: cfStorage,
	}
}

// ─── AddKatalog ─────────────────────────────────────────────────────────────
// POST /katalog/add-sampah/:bank_id
// Membuat katalog baru untuk BSI atau BSM beserta stok awal.
// BSU tidak diperbolehkan menambah katalog (katalog diwarisi dari BSI induknya).
// Harga sampah TIDAK diinput di sini; akan otomatis terupdate saat penjualan ke pengepul.
func (kc *KatalogController) AddKatalog(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := kc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	// BSU tidak diperbolehkan mengelola katalog secara mandiri
	if bank.JenisBank == models.BSU {
		c.JSON(http.StatusForbidden, gin.H{"error": "BSU tidak dapat menambah katalog sampah. Katalog diwarisi dari BSI induk."})
		return
	}

	var req struct {
		NamaSampah      string            `form:"nama_sampah" binding:"required"`
		Satuan          models.SatuanEnum `form:"satuan" binding:"required"`
		KategoriID      int               `form:"kategori_id" binding:"required"`
		RewardID        int               `form:"reward_id" binding:"required"`
		SyaratPemilahan string            `form:"syarat_pemilahan"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var fotoURL string
	if fileHeader, err := c.FormFile("foto"); err == nil {
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca foto"})
			return
		}
		defer file.Close()
		fotoURL, err = kc.CFStorage.UploadFile(file, fileHeader, "katalog_sampah")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupload foto: " + err.Error()})
			return
		}
	}

	sampahID := utils.GenerateBankRelatedID(bankID)

	tx := kc.DB.Begin()

	newKatalog := models.KatalogSampah{
		SampahID:        sampahID,
		BankID:          bankID,
		NamaSampah:      req.NamaSampah,
		PhotoURL:        fotoURL,
		Satuan:          req.Satuan,
		KategoriID:      req.KategoriID,
		RewardID:        req.RewardID,
		SyaratPemilahan: req.SyaratPemilahan,
	}
	if err := tx.Create(&newKatalog).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat katalog: " + err.Error()})
		return
	}

	stokAwal := models.StokSampah{BankID: bankID, SampahID: sampahID, Stok: 0}
	if err := tx.Create(&stokAwal).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat stok awal: " + err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Katalog berhasil ditambahkan",
		"data": gin.H{
			"katalog": newKatalog,
			"stok":    stokAwal,
		},
	})
}

// ─── EditKatalog ─────────────────────────────────────────────────────────────
// PATCH /katalog/edit-sampah/:sampah_id
// BSU tidak dapat mengedit katalog.
func (kc *KatalogController) EditKatalog(c *gin.Context) {
	kc.editKatalog(c)
}

// editKatalog adalah helper internal untuk EditKatalog.
func (kc *KatalogController) editKatalog(c *gin.Context) {
	sampahID := c.Param("sampah_id")

	var katalog models.KatalogSampah
	if err := kc.DB.Where("sampah_id = ?", sampahID).First(&katalog).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Katalog sampah tidak ditemukan"})
		return
	}

	// Verifikasi kepemilikan: BSU tidak boleh edit katalog
	var bank models.BankSampah
	if err := kc.DB.Where("bank_id = ?", katalog.BankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank pemilik katalog tidak ditemukan"})
		return
	}
	if bank.JenisBank == models.BSU {
		c.JSON(http.StatusForbidden, gin.H{"error": "BSU tidak dapat mengedit katalog sampah"})
		return
	}

	var req struct {
		NamaSampah      string            `form:"nama_sampah" binding:"required"`
		Satuan          models.SatuanEnum `form:"satuan" binding:"required"`
		KategoriID      int               `form:"kategori_id" binding:"required"`
		RewardID        int               `form:"reward_id" binding:"required"`
		SyaratPemilahan string            `form:"syarat_pemilahan"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	katalog.NamaSampah = req.NamaSampah
	katalog.Satuan = req.Satuan
	katalog.KategoriID = req.KategoriID
	katalog.RewardID = req.RewardID
	katalog.SyaratPemilahan = req.SyaratPemilahan

	if fileHeader, err := c.FormFile("foto"); err == nil {
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca foto"})
			return
		}
		defer file.Close()
		fotoURL, err := kc.CFStorage.UploadFile(file, fileHeader, "katalog_sampah")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupload foto: " + err.Error()})
			return
		}
		katalog.PhotoURL = fotoURL
	}

	if err := kc.DB.Save(&katalog).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate katalog: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Katalog sampah berhasil diupdate",
		"data":    katalog,
	})
}

// ─── GetKatalogSampahBank ────────────────────────────────────────────────────
// GET /katalog/get-sampah/:bank_id
// Mengambil seluruh katalog sampah milik bank beserta stok saat ini.
// Untuk BSU, akan mengambil katalog dari BSI induknya.
func (kc *KatalogController) GetKatalogSampahBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := kc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	// BSU mengambil katalog dari BSI induknya
	katalogBankID := bankID
	if bank.JenisBank == models.BSU {
		if bank.ParentBankID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "BSU ini tidak memiliki BSI induk"})
			return
		}
		katalogBankID = *bank.ParentBankID
	}

	type KatalogResponseItem struct {
		SampahID        string                `json:"sampah_id"`
		NamaSampah      string                `json:"nama_sampah"`
		PhotoURL        string                `json:"photo_url"`
		Satuan          models.SatuanEnum     `json:"satuan"`
		BankID          string                `json:"bank_id"`
		KategoriID      int                   `json:"kategori_id"`
		RewardID        int                   `json:"reward_id"`
		SyaratPemilahan string                `json:"syarat_pemilahan"`
		Stok            float64               `json:"stok"`
		Kategori        models.KategoriSampah `json:"kategori"`
		Reward          models.Reward         `json:"reward"`
	}

	var katalogs []models.KatalogSampah
	if err := kc.DB.Preload("Kategori").Preload("Reward").Where("bank_id = ?", katalogBankID).Find(&katalogs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil katalog: " + err.Error()})
		return
	}

	var result []KatalogResponseItem
	for _, k := range katalogs {
		// Ambil stok untuk bank yang meminta (bankID asli, bukan katalogBankID)
		var stok models.StokSampah
		var stokVal float64 = 0
		if err := kc.DB.Where("bank_id = ? AND sampah_id = ?", bankID, k.SampahID).Limit(1).Find(&stok).Error; err == nil {
			stokVal = stok.Stok
		}

		result = append(result, KatalogResponseItem{
			SampahID:        k.SampahID,
			NamaSampah:      k.NamaSampah,
			PhotoURL:        k.PhotoURL,
			Satuan:          k.Satuan,
			BankID:          k.BankID,
			KategoriID:      k.KategoriID,
			RewardID:        k.RewardID,
			SyaratPemilahan: k.SyaratPemilahan,
			Stok:            stokVal,
			Kategori:        k.Kategori,
			Reward:          k.Reward,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Katalog berhasil diambil",
		"data":    result,
	})
}

// ─── GetDetailSampah ─────────────────────────────────────────────────────────
// GET /katalog/get-detail/:sampah_id
// Mengambil detail lengkap satu item sampah:
// - Info dasar (nama, foto, satuan, kategori, reward)
// - Stok saat ini di bank pemilik
// - Daftar harga per level_user (nasabah, bsu, eksternal) beserta satuan reward
// - History perubahan harga per level_user, diurutkan dari terbaru
func (kc *KatalogController) GetDetailSampah(c *gin.Context) {
	sampahID := c.Param("sampah_id")

	// Ambil data katalog utama beserta relasi
	var katalog models.KatalogSampah
	if err := kc.DB.
		Preload("Kategori").
		Preload("Reward").
		Preload("Bank").
		Where("sampah_id = ?", sampahID).
		First(&katalog).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Katalog sampah tidak ditemukan"})
		return
	}

	// Ambil stok bank pemilik
	var stok models.StokSampah
	var stokVal float64 = 0
	if err := kc.DB.Where("bank_id = ? AND sampah_id = ?", katalog.BankID, sampahID).Limit(1).Find(&stok).Error; err == nil {
		stokVal = stok.Stok
	}

	// Ambil semua schema harga per level_user
	type HargaPerLevel struct {
		SchemaID     uint                      `json:"schema_id"`
		LevelUser    models.LevelUser          `json:"level_user"`
		Harga        float64                   `json:"harga"`
		SatuanReward models.SatuanRewardEnum   `json:"satuan_reward"`
	}
	var schemas []models.SchemaHargaSampah
	kc.DB.Where("sampah_id = ?", sampahID).Find(&schemas)

	hargaPerLevel := []HargaPerLevel{}
	for _, s := range schemas {
		hargaPerLevel = append(hargaPerLevel, HargaPerLevel{
			SchemaID:     s.SchemaID,
			LevelUser:    s.LevelUser,
			Harga:        s.Harga,
			SatuanReward: s.SatuanReward,
		})
	}

	// Ambil history perubahan harga, join dengan admin untuk nama pengubah
	type HistoryItem struct {
		HistoryID    int                     `json:"history_id"    gorm:"column:history_sampah_id"`
		SchemaID     int                     `json:"schema_id"     gorm:"column:schema_id"`
		LevelUser    models.LevelUser        `json:"level_user"    gorm:"column:level_user"`
		HargaLama    float64                 `json:"harga_lama"    gorm:"column:harga_lama"`
		HargaBaru    float64                 `json:"harga_baru"    gorm:"column:harga_baru"`
		ChangedAt    string                  `json:"changed_at"    gorm:"column:changed_at"`
		ChangedByID  string                  `json:"changed_by_id" gorm:"column:changed_by"`
		ChangedByNama string                 `json:"changed_by_nama" gorm:"column:changed_by_nama"`
	}
	var histories []HistoryItem
	kc.DB.Model(&models.KatalogSampahHistory{}).
		Select(`katalog_sampah_history.history_sampah_id,
			katalog_sampah_history.schema_id,
			schema_harga_sampah.level_user,
			katalog_sampah_history.harga_lama,
			katalog_sampah_history.harga_baru,
			katalog_sampah_history.changed_at,
			katalog_sampah_history.changed_by,
			users.nama as changed_by_nama`).
		Joins("JOIN schema_harga_sampah ON schema_harga_sampah.schema_id = katalog_sampah_history.schema_id").
		Joins("JOIN admin ON admin.admin_id = katalog_sampah_history.changed_by").
		Joins("JOIN users ON users.user_id = admin.user_id").
		Where("schema_harga_sampah.sampah_id = ?", sampahID).
		Order("katalog_sampah_history.changed_at DESC").
		Find(&histories)

	// Susun response akhir
	response := gin.H{
		"sampah_id":        katalog.SampahID,
		"nama_sampah":      katalog.NamaSampah,
		"photo_url":        katalog.PhotoURL,
		"satuan":           katalog.Satuan,
		"syarat_pemilahan": katalog.SyaratPemilahan,
		"bank_id":          katalog.BankID,
		"kategori":         katalog.Kategori,
		"reward":           katalog.Reward,
		"stok":             stokVal,
		"harga_per_level":  hargaPerLevel,
		"history_harga":    histories,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail sampah berhasil diambil",
		"data":    response,
	})
}

// ─── DeleteKatalogSampah ─────────────────────────────────────────────────────
// DELETE /katalog/delete-sampah/:sampah_id
func (kc *KatalogController) DeleteKatalogSampah(c *gin.Context) {
	sampahID := c.Param("sampah_id")

	var katalog models.KatalogSampah
	if err := kc.DB.Where("sampah_id = ?", sampahID).First(&katalog).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Katalog sampah tidak ditemukan"})
		return
	}

	tx := kc.DB.Begin()

	// Hapus stok sampah terkait
	if err := tx.Where("sampah_id = ?", sampahID).Delete(&models.StokSampah{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus stok sampah: " + err.Error()})
		return
	}

	// Hapus katalog utama
	if err := tx.Delete(&katalog).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus katalog: " + err.Error()})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Katalog sampah berhasil dihapus"})
}

// ─── AddNewKategori ──────────────────────────────────────────────────────────
// POST /katalog/add-kategori
func (kc *KatalogController) AddNewKategori(c *gin.Context) {
	var req struct {
		Kategori string `form:"kategori" binding:"required"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var dupCount int64
	kc.DB.Model(&models.KategoriSampah{}).
		Where("LOWER(REGEXP_REPLACE(kategori, '[^a-zA-Z0-9]', '', 'g')) = ?", utils.NormalizeString(req.Kategori)).
		Count(&dupCount)
	if dupCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Kategori dengan nama tersebut sudah ada"})
		return
	}

	newKategori := models.KategoriSampah{Kategori: req.Kategori}

	if err := kc.DB.Create(&newKategori).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat kategori: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Kategori berhasil ditambahkan",
		"data":    newKategori,
	})
}

// ─── GetKategori ─────────────────────────────────────────────────────────────
// GET /katalog/get-kategori
func (kc *KatalogController) GetKategori(c *gin.Context) {
	var kategori []models.KategoriSampah
	if err := kc.DB.Find(&kategori).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil kategori: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Kategori berhasil diambil",
		"data":    kategori,
	})
}

// ─── UpdateKategori ──────────────────────────────────────────────────────────
// PATCH /katalog/update-kategori/:kategori_id
func (kc *KatalogController) UpdateKategori(c *gin.Context) {
	kategoriID := c.Param("kategori_id")

	var kategori models.KategoriSampah
	if err := kc.DB.Where("kategori_id = ?", kategoriID).First(&kategori).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kategori tidak ditemukan"})
		return
	}

	var req struct {
		Kategori string `json:"kategori" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var dupCount int64
	kc.DB.Model(&models.KategoriSampah{}).
		Where("LOWER(REGEXP_REPLACE(kategori, '[^a-zA-Z0-9]', '', 'g')) = ? AND kategori_id != ?", utils.NormalizeString(req.Kategori), kategoriID).
		Count(&dupCount)
	if dupCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Kategori dengan nama tersebut sudah ada"})
		return
	}

	kategori.Kategori = req.Kategori
	if err := kc.DB.Save(&kategori).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate kategori: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Kategori berhasil diupdate",
		"data":    kategori,
	})
}

// ─── DeleteKategori ──────────────────────────────────────────────────────────
// DELETE /katalog/delete-kategori/:kategori_id
func (kc *KatalogController) DeleteKategori(c *gin.Context) {
	kategoriID := c.Param("kategori_id")

	var kategori models.KategoriSampah
	if err := kc.DB.Where("kategori_id = ?", kategoriID).First(&kategori).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kategori tidak ditemukan"})
		return
	}

	var count int64
	kc.DB.Model(&models.KatalogSampah{}).Where("kategori_id = ?", kategoriID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Kategori masih digunakan oleh katalog sampah dan tidak dapat dihapus"})
		return
	}

	if err := kc.DB.Delete(&kategori).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus kategori: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Kategori berhasil dihapus"})
}