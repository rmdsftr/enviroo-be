package controllers

import (
	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"net/http"

	"enviroo-be/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SembakoController struct {
	DB        *gorm.DB
	CFStorage *storage.CloudflareStorage
}

func NewSembakoController(db *gorm.DB, cfStorage *storage.CloudflareStorage) *SembakoController {
	return &SembakoController{
		DB:        db,
		CFStorage: cfStorage,
	}
}

func (sc *SembakoController) AddNewSembako(c *gin.Context) {
	bankID := c.Param("bank_id")

	// Cek apakah bank sampah ada
	var bank models.BankSampah
	if err := sc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get bank sampah: " + err.Error()})
		}
		return
	}

	var req struct {
		NamaSembako    string  `form:"nama_sembako" binding:"required"`
		HargaNasabah   float64 `form:"harga_nasabah" binding:"required"`
		HargaEksternal float64 `form:"harga_eksternal" binding:"required"`
		HargaBSU       float64 `form:"harga_bsu"` // Wajib jika bank adalah BSI
	}

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request form: " + err.Error()})
		return
	}

	if bank.JenisBank == models.BSI && req.HargaBSU == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Harga BSU wajib diisi untuk bank BSI"})
		return
	}

	// Upload foto (opsional)
	var fotoURL string
	fileHeader, err := c.FormFile("foto")
	if err == nil {
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca foto"})
			return
		}
		defer file.Close()

		fotoURL, err = sc.CFStorage.UploadFile(file, fileHeader, "katalog_sembako")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupload foto ke cloud: " + err.Error()})
			return
		}
	}

	// Cek apakah nama sembako sudah ada di bank ini
	var existing models.KatalogSembako
	if err := sc.DB.Where("bank_id = ? AND nama_sembako = ?", bankID, req.NamaSembako).Limit(1).Find(&existing).Error; err == nil && existing.SembakoID != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama sembako sudah ada di bank ini"})
		return
	}

	// Generate ID
	sembakoID := utils.GenerateBankRelatedID(bankID)

	tx := sc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
		return
	}

	newSembako := models.KatalogSembako{
		SembakoID:   sembakoID,
		BankID:      bankID,
		NamaSembako: req.NamaSembako,
		PhotoURL:    fotoURL,
	}

	if err := tx.Create(&newSembako).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sembako: " + err.Error()})
		return
	}

	// Buat schema harga
	schemaHargas := []models.SchemaHargaSembako{
		{SembakoID: sembakoID, LevelUser: models.LevelNasabah, PoinHarga: req.HargaNasabah},
		{SembakoID: sembakoID, LevelUser: models.LevelEksternal, PoinHarga: req.HargaEksternal},
	}
	if bank.JenisBank == models.BSI {
		schemaHargas = append(schemaHargas, models.SchemaHargaSembako{SembakoID: sembakoID, LevelUser: models.LevelBSU, PoinHarga: req.HargaBSU})
	}

	if err := tx.Create(&schemaHargas).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat skema harga sembako: " + err.Error()})
		return
	}

	// Buat stok awal
	stokAwal := models.StokSembakoBank{BankID: bankID, SembakoID: sembakoID, Stok: 0}
	if err := tx.Create(&stokAwal).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat stok awal sembako: " + err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Sembako berhasil ditambahkan",
		"data": gin.H{
			"sembako":      newSembako,
			"schema_harga": schemaHargas,
			"stok":         stokAwal,
		},
	})
}

func (sc *SembakoController) GetSembakoBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := sc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	if bank.JenisBank == models.BSU {
		bankID = *bank.ParentBankID
	}

	type HargaSembakoInfo struct {
		LevelUser models.LevelUser `json:"level_user"`
		PoinHarga float64          `json:"poin_harga"`
	}

	type SembakoResponseItem struct {
		SembakoID   string             `json:"sembako_id"`
		NamaSembako string             `json:"nama_sembako"`
		PhotoURL    string             `json:"photo_url"`
		Stok        float64            `json:"stok"`
		SchemaHarga []HargaSembakoInfo `json:"schema_harga"`
	}

	var sembakoRows []models.KatalogSembako
	if err := sc.DB.Where("bank_id = ?", bankID).Find(&sembakoRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get sembako: " + err.Error()})
		return
	}

	var result []SembakoResponseItem
	for _, s := range sembakoRows {
		var schemaRows []models.SchemaHargaSembako
		sc.DB.Where("sembako_id = ?", s.SembakoID).Find(&schemaRows)

		var schemaInfo []HargaSembakoInfo
		for _, sr := range schemaRows {
			schemaInfo = append(schemaInfo, HargaSembakoInfo{
				LevelUser: sr.LevelUser,
				PoinHarga: sr.PoinHarga,
			})
		}

		var stokRow models.StokSembakoBank
		sc.DB.Where("bank_id = ? AND sembako_id = ?", bankID, s.SembakoID).First(&stokRow)

		result = append(result, SembakoResponseItem{
			SembakoID:   s.SembakoID,
			NamaSembako: s.NamaSembako,
			PhotoURL:    s.PhotoURL,
			Stok:        stokRow.Stok,
			SchemaHarga: schemaInfo,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Sembako fetched successfully",
		"data":    result,
	})
}

func (sc *SembakoController) DeleteSembako(c *gin.Context) {
	sembakoID := c.Param("sembako_id")

	var sembako models.KatalogSembako
	if err := sc.DB.Where("sembako_id = ?", sembakoID).First(&sembako).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sembako tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get sembako: " + err.Error()})
		}
		return
	}

	tx := sc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Delete History
	if err := tx.Where("sembako_id = ?", sembakoID).Delete(&models.HistoryPoinSembako{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus history sembako: " + err.Error()})
		return
	}

	// Delete Schema
	if err := tx.Where("sembako_id = ?", sembakoID).Delete(&models.SchemaHargaSembako{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus skema harga sembako: " + err.Error()})
		return
	}

	// Delete Stok
	if err := tx.Where("sembako_id = ?", sembakoID).Delete(&models.StokSembakoBank{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus stok sembako: " + err.Error()})
		return
	}

	// Delete Sembako
	if err := tx.Delete(&sembako).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete sembako: " + err.Error()})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "Sembako berhasil dihapus",
	})
}

func (sc *SembakoController) UpdateHargaSembako(c *gin.Context) {
	sembakoID := c.Param("sembako_id")

	var req struct {
		LevelUser models.LevelUser `form:"level_user" binding:"required"`
		PoinHarga float64          `form:"poin_harga" binding:"required"`
		ChangedBy string           `form:"changed_by" binding:"required"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request form: " + err.Error()})
		return
	}

	var schemaHarga models.SchemaHargaSembako
	if err := sc.DB.Where("sembako_id = ? AND level_user = ?", sembakoID, req.LevelUser).First(&schemaHarga).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Skema harga sembako tidak ditemukan"})
		return
	}

	tx := sc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	poinLama := schemaHarga.PoinHarga

	// Simpan history perubahan poin
	history := models.HistoryPoinSembako{
		SembakoID: sembakoID,
		LevelUser: req.LevelUser,
		PoinLama:  poinLama,
		PoinBaru:  req.PoinHarga,
		ChangedBy: req.ChangedBy,
	}

	// Update poin di schema
	schemaHarga.PoinHarga = req.PoinHarga
	if err := tx.Save(&schemaHarga).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update harga sembako: " + err.Error()})
		return
	}

	// Simpan rekam jejak harga
	if err := tx.Create(&history).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sembako history: " + err.Error()})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "Harga sembako berhasil diupdate",
	})
}

func (sc *SembakoController) EditSembako(c *gin.Context) {
	sembakoID := c.Param("sembako_id")

	var req struct {
		NamaSembako string `form:"nama_sembako"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request form: " + err.Error()})
		return
	}

	var sembako models.KatalogSembako
	if err := sc.DB.Where("sembako_id = ?", sembakoID).First(&sembako).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sembako tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get sembako: " + err.Error()})
		}
		return
	}

	// Cek apakah nama sembako baru sudah ada di bank ini (kecuali dirinya sendiri)
	if req.NamaSembako != "" && req.NamaSembako != sembako.NamaSembako {
		var existing models.KatalogSembako
		if err := sc.DB.Where("bank_id = ? AND nama_sembako = ? AND sembako_id != ?", sembako.BankID, req.NamaSembako, sembakoID).Limit(1).Find(&existing).Error; err == nil && existing.SembakoID != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nama sembako sudah ada di bank ini"})
			return
		}
	}

	// Build updates
	updates := make(map[string]interface{})
	if req.NamaSembako != "" {
		updates["nama_sembako"] = req.NamaSembako
	}

	// Upload foto baru (opsional)
	fileHeader, err := c.FormFile("foto")
	if err == nil {
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca foto"})
			return
		}
		defer file.Close()

		fotoURL, err := sc.CFStorage.UploadFile(file, fileHeader, "katalog_sembako")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupload foto ke cloud: " + err.Error()})
			return
		}
		updates["photo_url"] = fotoURL
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada data yang diupdate"})
		return
	}

	if err := sc.DB.Model(&sembako).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sembako: " + err.Error()})
		return
	}

	// Reload data terbaru
	sc.DB.Where("sembako_id = ?", sembakoID).First(&sembako)

	c.JSON(http.StatusOK, gin.H{
		"message": "Sembako berhasil diupdate",
		"data":    sembako,
	})
}

func (sc *SembakoController) GetHistorySembako(c *gin.Context) {
	sembakoID := c.Param("sembako_id")

	var sembako models.KatalogSembako
	if err := sc.DB.Where("sembako_id = ?", sembakoID).First(&sembako).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sembako tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get sembako: " + err.Error()})
		}
		return
	}

	var history []models.HistoryPoinSembako
	if err := sc.DB.Where("sembako_id = ?", sembakoID).Order("changed_at DESC").Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get sembako history: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "History poin sembako fetched successfully",
		"data":    history,
	})
}
