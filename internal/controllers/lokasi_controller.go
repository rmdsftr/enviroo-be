package controllers

import (
	"enviroo-be/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LokasiController struct {
	DB *gorm.DB
}

func NewLokasiController(db *gorm.DB) *LokasiController {
	return &LokasiController{DB: db}
}

func (lc *LokasiController) GetLokasiBankSampah(c *gin.Context) {
	type LokasiBankSampahResponse struct {
		BankID        string  `json:"bank_id" gorm:"column:bank_id"`
		NamaBank      string  `json:"nama_bank" gorm:"column:nama_bank"`
		JenisBank     string  `json:"jenis_bank" gorm:"column:jenis_bank"`
		Latitude      float64 `json:"latitude" gorm:"column:latitude"`
		Longitude     float64 `json:"longitude" gorm:"column:longitude"`
	}

	var results []LokasiBankSampahResponse

	query := lc.DB.Model(&models.BankSampah{}).
		Select("bank_sampah.bank_id, bank_sampah.nama_bank, bank_sampah.jenis_bank, bank_sampah.latitude, bank_sampah.longitude")

	if err := query.Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get bank sampah locations: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bank sampah locations fetched successfully",
		"data":    results,
	})
}
