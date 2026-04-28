package controllers

import (
	"enviroo-be/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DashboardController struct {
	DB *gorm.DB
}

func NewDashboardController(db *gorm.DB) *DashboardController {
	return &DashboardController{
		DB: db,
	}
}

func (dc *DashboardController) GetDashboardPetugas(c *gin.Context) {
	bankID := c.Param("bank_id")
	if bankID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank ID is required"})
		return
	}

	type ResponDashboardPetugasBSU struct {
		NamaBank string `json:"nama_bank"`
		PhotoBank string `json:"photo_bank"`
		AlamatBank string `json:"alamat_bank"`
		NamaBankPusat string `json:"nama_bank_pusat"`
		JumlahNasabah int64 `json:"jumlah_nasabah"`
		JumlahStaff int64 `json:"jumlah_staff"`
	}

	type ResponDashboardPetugasBSI struct {
		NamaBank      string `json:"nama_bank"`
		PhotoBank     string `json:"photo_bank"`
		AlamatBank    string `json:"alamat_bank"`
		JumlahNasabah int64  `json:"jumlah_nasabah"`
		JumlahBSU     int64  `json:"jumlah_bsu"`
		JumlahStaff   int64  `json:"jumlah_staff"`
	}

	type ResponDashboardPetugasBSM struct {
		NamaBank string `json:"nama_bank"`
		PhotoBank string `json:"photo_bank"`
		AlamatBank string `json:"alamat_bank"`
		JumlahNasabah int64 `json:"jumlah_nasabah"`
		JumlahStaff int64 `json:"jumlah_staff"`
	}

	var bank models.BankSampah
	if err := dc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank not found"})
		return
	}

	var jumlahNasabah int64
	dc.DB.Model(&models.Nasabah{}).Where("bank_id = ?", bankID).Count(&jumlahNasabah)

	var jumlahStaff int64
	dc.DB.Model(&models.Admin{}).Where("bank_id = ?", bankID).Count(&jumlahStaff)

	switch bank.JenisBank {
	case models.BSU:
		var namaBankPusat string
		if bank.ParentBankID != nil {
			var parentBank models.BankSampah
			if err := dc.DB.Where("bank_id = ?", *bank.ParentBankID).First(&parentBank).Error; err == nil {
				namaBankPusat = parentBank.NamaBank
			}
		}

		response := ResponDashboardPetugasBSU{
			NamaBank:      bank.NamaBank,
			PhotoBank:     bank.PhotoURL,
			AlamatBank:    bank.Alamat,
			NamaBankPusat: namaBankPusat,
			JumlahNasabah: jumlahNasabah,
			JumlahStaff:   jumlahStaff,
		}
		c.JSON(http.StatusOK, gin.H{"data": response})

	case models.BSI:
		var jumlahBSU int64
		dc.DB.Model(&models.BankSampah{}).Where("parent_bank_id = ? AND jenis_bank = ?", bankID, models.BSU).Count(&jumlahBSU)

		response := ResponDashboardPetugasBSI{
			NamaBank:      bank.NamaBank,
			PhotoBank:     bank.PhotoURL,
			AlamatBank:    bank.Alamat,
			JumlahNasabah: jumlahNasabah,
			JumlahBSU:     jumlahBSU,
			JumlahStaff:   jumlahStaff,
		}
		c.JSON(http.StatusOK, gin.H{"data": response})

	case models.BSM:
		response := ResponDashboardPetugasBSM{
			NamaBank:      bank.NamaBank,
			PhotoBank:     bank.PhotoURL,
			AlamatBank:    bank.Alamat,
			JumlahNasabah: jumlahNasabah,
			JumlahStaff:   jumlahStaff,
		}
		c.JSON(http.StatusOK, gin.H{"data": response})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bank type"})
	}
}
	
// GetSaldoBank: Mengembalikan saldo poin dan kas fisik (per reward) milik sebuah bank.
// Response mencakup:
// - saldo_poin: total poin yang dimiliki bank (dari tabel saldo_bank)
// - kas: daftar kas fisik per reward, misal Rupiah atau Emas (dari tabel kas_bank)
func (dc *DashboardController) GetSaldoBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	// Validasi bank
	var bank models.BankSampah
	if err := dc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Bank tidak ditemukan"})
		return
	}

	// Ambil saldo poin dari tabel saldo_bank
	var saldoBank models.SaldoBank
	if err := dc.DB.Where("bank_id = ?", bankID).First(&saldoBank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Saldo bank tidak ditemukan"})
		return
	}

	// Ambil semua kas per reward dari tabel kas_bank (beserta info reward-nya)
	var kasList []models.KasBank
	if err := dc.DB.Preload("Reward").Where("bank_id = ?", bankID).Find(&kasList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal mengambil data kas bank"})
		return
	}

	// Format response kas agar lebih mudah dibaca frontend
	type kasResponse struct {
		RewardID   int     `json:"reward_id"`
		NamaReward string  `json:"nama_reward"`
		Satuan     string  `json:"satuan"`
		Nominal    float64 `json:"nominal"`
	}
	var kasFormatted []kasResponse
	for _, k := range kasList {
		kasFormatted = append(kasFormatted, kasResponse{
			RewardID:   k.RewardID,
			NamaReward: k.Reward.NamaReward,
			Satuan:     k.Reward.Satuan,
			Nominal:    k.Nominal,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data saldo bank berhasil diambil",
		"data": gin.H{
			"bank_id":        bank.BankID,
			"nama_bank":      bank.NamaBank,
			"saldo_poin":     saldoBank.TotalPoin,
			"last_updated":   saldoBank.LastUpdatedAt,
			"kas":            kasFormatted,
		},
	})
}
