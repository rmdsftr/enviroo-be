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

// ─── AddKatalogBSI ──────────────────────────────────────────────────────────
// POST /katalog/bsi/add-sampah/:bank_id
// Membuat katalog baru khusus untuk BSI lengkap dengan 3 skema harga dan stok awal.
func (kc *KatalogController) AddKatalogBSI(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := kc.DB.Where("bank_id = ? AND jenis_bank = ?", bankID, models.BSI).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "BSI tidak ditemukan"})
		return
	}

	var req struct {
		NamaSampah      string            `form:"nama_sampah" binding:"required"`
		Satuan          models.SatuanEnum `form:"satuan" binding:"required"`
		KategoriID      int               `form:"kategori_id" binding:"required"`
		HargaNasabah    float64           `form:"harga_nasabah" binding:"required"` // poin yang didapat nasabah
		HargaBSU        float64           `form:"harga_bsu" binding:"required"`     // poin yang didapat BSU dari BSI
		HargaEksternal  float64           `form:"harga_eksternal" binding:"required"` // harga jual BSI ke luar
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
		SampahID:   sampahID,
		BankID:     bankID,
		NamaSampah: req.NamaSampah,
		PhotoURL:   fotoURL,
		Satuan:     req.Satuan,
		KategoriID: req.KategoriID,
	}
	if err := tx.Create(&newKatalog).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat katalog: " + err.Error()})
		return
	}

	schemaHargas := []models.SchemaHargaSampah{
		{SampahID: sampahID, LevelUser: models.LevelNasabah, PoinHarga: req.HargaNasabah},
		{SampahID: sampahID, LevelUser: models.LevelBSU, PoinHarga: req.HargaBSU},
		{SampahID: sampahID, LevelUser: models.LevelEksternal, PoinHarga: req.HargaEksternal},
	}
	if err := tx.Create(&schemaHargas).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat skema harga: " + err.Error()})
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
		"message": "Katalog BSI berhasil ditambahkan",
		"data": gin.H{
			"katalog":       newKatalog,
			"schema_harga":  schemaHargas,
			"stok":          stokAwal,
		},
	})
}

// ─── AddKatalogBSM ──────────────────────────────────────────────────────────
// POST /katalog/bsm/add-sampah/:bank_id
// Membuat katalog baru khusus untuk BSM lengkap dengan 2 skema harga dan stok awal.
// BSM tidak punya BSU sehingga tidak ada level harga 'bsu'.
func (kc *KatalogController) AddKatalogBSM(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := kc.DB.Where("bank_id = ? AND jenis_bank = ?", bankID, models.BSM).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "BSM tidak ditemukan"})
		return
	}

	var req struct {
		NamaSampah     string            `form:"nama_sampah" binding:"required"`
		Satuan         models.SatuanEnum `form:"satuan" binding:"required"`
		KategoriID     int               `form:"kategori_id" binding:"required"`
		HargaNasabah   float64           `form:"harga_nasabah" binding:"required"`
		HargaEksternal float64           `form:"harga_eksternal" binding:"required"`
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
		SampahID:   sampahID,
		BankID:     bankID,
		NamaSampah: req.NamaSampah,
		PhotoURL:   fotoURL,
		Satuan:     req.Satuan,
		KategoriID: req.KategoriID,
	}
	if err := tx.Create(&newKatalog).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat katalog: " + err.Error()})
		return
	}

	schemaHargas := []models.SchemaHargaSampah{
		{SampahID: sampahID, LevelUser: models.LevelNasabah, PoinHarga: req.HargaNasabah},
		{SampahID: sampahID, LevelUser: models.LevelEksternal, PoinHarga: req.HargaEksternal},
	}
	if err := tx.Create(&schemaHargas).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat skema harga: " + err.Error()})
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
		"message": "Katalog BSM berhasil ditambahkan",
		"data": gin.H{
			"katalog":      newKatalog,
			"schema_harga": schemaHargas,
			"stok":         stokAwal,
		},
	})
}

// ─── EditKatalogBSI ──────────────────────────────────────────────────────────
// PATCH /katalog/bsi/edit-sampah/:sampah_id
// Edit field non-harga untuk katalog milik BSI.
func (kc *KatalogController) EditKatalogBSI(c *gin.Context) {
	kc.editKatalog(c, models.BSI)
}

// ─── EditKatalogBSM ──────────────────────────────────────────────────────────
// PATCH /katalog/bsm/edit-sampah/:sampah_id
// Edit field non-harga untuk katalog milik BSM.
func (kc *KatalogController) EditKatalogBSM(c *gin.Context) {
	kc.editKatalog(c, models.BSM)
}

// editKatalog adalah helper internal untuk EditKatalogBSI dan EditKatalogBSM.
func (kc *KatalogController) editKatalog(c *gin.Context, jenisBank models.JenisBank) {
	sampahID := c.Param("sampah_id")

	var katalog models.KatalogSampah
	if err := kc.DB.Where("sampah_id = ?", sampahID).First(&katalog).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Katalog sampah tidak ditemukan"})
		return
	}

	// Verifikasi kepemilikan berdasarkan jenis bank
	var bank models.BankSampah
	if err := kc.DB.Where("bank_id = ? AND jenis_bank = ?", katalog.BankID, jenisBank).First(&bank).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak: katalog ini bukan milik " + string(jenisBank)})
		return
	}

	var req struct {
		NamaSampah string            `form:"nama_sampah" binding:"required"`
		Satuan     models.SatuanEnum `form:"satuan" binding:"required"`
		KategoriID int               `form:"kategori_id" binding:"required"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	katalog.NamaSampah = req.NamaSampah
	katalog.Satuan = req.Satuan
	katalog.KategoriID = req.KategoriID

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

// ─── UpdateHargaSchema ───────────────────────────────────────────────────────
// PATCH /katalog/update-harga/:sampah_id
// Mengupdate satu skema harga berdasarkan level_user, dan mencatat history-nya.
func (kc *KatalogController) UpdateHargaSchema(c *gin.Context) {
	sampahID := c.Param("sampah_id")

	var req struct {
		LevelUser    models.LevelUser `json:"level_user" binding:"required"`
		PoinHargaBaru float64         `json:"poin_harga_baru" binding:"required"`
		ChangedBy    string           `json:"changed_by" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var schema models.SchemaHargaSampah
	if err := kc.DB.Where("sampah_id = ? AND level_user = ?", sampahID, req.LevelUser).First(&schema).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Skema harga tidak ditemukan untuk level tersebut"})
		return
	}

	oldPoin := schema.PoinHarga

	tx := kc.DB.Begin()

	schema.PoinHarga = req.PoinHargaBaru
	if err := tx.Save(&schema).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate skema harga: " + err.Error()})
		return
	}

	history := models.KatalogSampahHistory{
		SampahID:  sampahID,
		LevelUser: req.LevelUser,
		OldPoin:   oldPoin,
		NewPoin:   req.PoinHargaBaru,
		ChangedBy: req.ChangedBy,
	}
	if err := tx.Create(&history).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencatat history harga: " + err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Skema harga berhasil diupdate",
		"data":    schema,
	})
}

// ─── GetKatalogSampahBank ────────────────────────────────────────────────────
// GET /katalog/get-sampah/:bank_id
// Mengambil seluruh katalog sampah milik bank. Untuk BSU, akan mengambil
// katalog dari BSI induknya beserta stok yang dimiliki BSU tersebut.
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

	type SchemaHargaDTO struct {
		LevelUser models.LevelUser `json:"level_user"`
		PoinHarga float64          `json:"poin_harga"`
	}

	type KatalogResponseItem struct {
		SampahID   string                    `json:"sampah_id" gorm:"column:sampah_id"`
		NamaSampah string                    `json:"nama_sampah" gorm:"column:nama_sampah"`
		PhotoURL   string                    `json:"photo_url" gorm:"column:photo_url"`
		Satuan     models.SatuanEnum         `json:"satuan" gorm:"column:satuan"`
		BankID     string                    `json:"bank_id" gorm:"column:bank_id"`
		KategoriID int                       `json:"kategori_id" gorm:"column:kategori_id"`
		Stok       float64                   `json:"stok" gorm:"column:stok"`
		Kategori   models.KategoriSampah     `json:"kategori"`
		Harga      []SchemaHargaDTO          `json:"harga"`
	}

	var katalogs []models.KatalogSampah
	if err := kc.DB.Preload("Kategori").Where("bank_id = ?", katalogBankID).Find(&katalogs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil katalog: " + err.Error()})
		return
	}

	var result []KatalogResponseItem
	for _, k := range katalogs {
		// Ambil skema harga
		var schemas []models.SchemaHargaSampah
		kc.DB.Where("sampah_id = ?", k.SampahID).Find(&schemas)

		hargas := []SchemaHargaDTO{}
		for _, s := range schemas {
			hargas = append(hargas, SchemaHargaDTO{
				LevelUser: s.LevelUser,
				PoinHarga: s.PoinHarga,
			})
		}

		// Ambil stok untuk bank yang meminta (bankID asli, bukan katalogBankID)
		var stok models.StokSampah
		var stokVal float64 = 0
		// Menggunakan Find alih-alih First untuk menghindari log "record not found" jika BSU belum punya record stok
		if err := kc.DB.Where("bank_id = ? AND sampah_id = ?", bankID, k.SampahID).Limit(1).Find(&stok).Error; err == nil {
			stokVal = stok.Stok
		}

		result = append(result, KatalogResponseItem{
			SampahID:   k.SampahID,
			NamaSampah: k.NamaSampah,
			PhotoURL:   k.PhotoURL,
			Satuan:     k.Satuan,
			BankID:     k.BankID,
			KategoriID: k.KategoriID,
			Stok:       stokVal,
			Kategori:   k.Kategori,
			Harga:      hargas,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Katalog berhasil diambil",
		"data":    result,
	})
}

// ─── GetSchemaHarga ──────────────────────────────────────────────────────────
// GET /katalog/get-schema-harga/:sampah_id
// Mengambil semua skema harga untuk satu jenis sampah.
func (kc *KatalogController) GetSchemaHarga(c *gin.Context) {
	sampahID := c.Param("sampah_id")

	var schemas []models.SchemaHargaSampah
	if err := kc.DB.Where("sampah_id = ?", sampahID).Find(&schemas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil skema harga: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Skema harga berhasil diambil",
		"data":    schemas,
	})
}

// ─── GetHistoryKatalogSampah ─────────────────────────────────────────────────
// GET /katalog/get-history/:sampah_id
// Mengambil riwayat perubahan harga, termasuk level_user dan nama admin.
func (kc *KatalogController) GetHistoryKatalogSampah(c *gin.Context) {
	sampahID := c.Param("sampah_id")

	type HistoryResponse struct {
		models.KatalogSampahHistory
		AdminNama string `json:"admin_nama" gorm:"column:admin_nama"`
	}

	var history []HistoryResponse

	if err := kc.DB.Model(&models.KatalogSampahHistory{}).
		Select("katalog_sampah_history.*, users.nama as admin_nama").
		Joins("JOIN admin ON admin.admin_id = katalog_sampah_history.changed_by").
		Joins("JOIN users ON users.user_id = admin.user_id").
		Where("katalog_sampah_history.sampah_id = ?", sampahID).
		Order("katalog_sampah_history.changed_at DESC").
		Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil history: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "History harga berhasil diambil",
		"data":    history,
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

	// Hapus history harga
	if err := tx.Where("sampah_id = ?", sampahID).Delete(&models.KatalogSampahHistory{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus history harga: " + err.Error()})
		return
	}

	// Hapus skema harga
	if err := tx.Where("sampah_id = ?", sampahID).Delete(&models.SchemaHargaSampah{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus skema harga: " + err.Error()})
		return
	}

	// Hapus stok sampah
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

	c.JSON(http.StatusOK, gin.H{"message": "Katalog sampah beserta data terkait berhasil dihapus"})
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