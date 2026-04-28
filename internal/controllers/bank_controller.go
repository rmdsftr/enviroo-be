package controllers

import (
	"encoding/json"
	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BankController struct {
	DB        *gorm.DB
	CFStorage *storage.CloudflareStorage
	Mailer    *utils.Mailer
}

func NewBankController(db *gorm.DB, cfStorage *storage.CloudflareStorage, mailer *utils.Mailer) *BankController {
	return &BankController{
		DB:        db,
		CFStorage: cfStorage,
		Mailer:    mailer,
	}
}

func (bc *BankController) AktivasiBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	type RequestDeactivate struct {
		AdminID   string `json:"admin_id"`
		Informasi string `json:"informasi"`
		Keterangan string `json:"keterangan"`
	}

	var req RequestDeactivate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format request tidak valid"})
		return
	}

	var bank models.BankSampah
	if err := bc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	oldStatus := "Non Aktif"
	if bank.IsActive {
		oldStatus = "Aktif"
	}

	// Toggle status
	bank.IsActive = !bank.IsActive

	newStatus := "Non Aktif"
	if bank.IsActive {
		newStatus = "Aktif"
	}

	// Gunakan transaksi untuk memastikan konsistensi data
	err := bc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&bank).Error; err != nil {
			return err
		}

		oldVal, _ := json.Marshal(map[string]string{"status_bank": oldStatus})
		newVal, _ := json.Marshal(map[string]string{"status_bank": newStatus})

		newHistory := models.HistoryAkunBank{
			BankID:    bankID,
			Action:    "UPDATE",
			OldValue:  datatypes.JSON(oldVal),
			NewValue:  datatypes.JSON(newVal),
			Informasi: req.Informasi,
			Keterangan: req.Keterangan,
			CreatedBy: req.AdminID,
		}

		if err := tx.Create(&newHistory).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status bank"})
		return
	}

	statusMsg := "diaktifkan"
	if !bank.IsActive {
		statusMsg = "dinonaktifkan"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bank berhasil " + statusMsg,
		"data":    bank,
	})
}

func (bc *BankController) GetNasabahByBankID(c *gin.Context) {
	bankID := c.Param("bank_id")
	if bankID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank ID is required"})
		return
	}

	var results []models.Nasabah

	// Menggunakan Preload untuk mengambil data User yang berelasi dengan Nasabah
	if err := bc.DB.Preload("User").Where("bank_id = ?", bankID).Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get nasabah: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Nasabah fetched successfully",
		"data":    results,
	})
}
