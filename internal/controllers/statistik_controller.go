package controllers

import (
	"enviroo-be/internal/models"
	"net/http"

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