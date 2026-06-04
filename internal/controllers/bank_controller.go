package controllers

import (
	"encoding/json"
	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"
	"fmt"
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
	if err := bc.DB.Preload("User").Where("bank_id = ?", bankID).Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get nasabah: " + err.Error()})
		return
	}

	type NasabahItem struct {
		NasabahID string            `json:"nasabah_id"`
		Nama      string            `json:"nama"`
		Email     string            `json:"email"`
		Foto      string            `json:"foto"`
		Status    models.StatusAkun `json:"status"`
		NomorRekening string 		`json:"nomor_rekening"`
	}

	items := make([]NasabahItem, 0, len(results))
	for _, n := range results {
		items = append(items, NasabahItem{
			NasabahID: n.NasabahID,
			Nama:      n.User.Nama,
			Email:     n.User.Email,
			Foto:      n.User.PhotoURL,
			Status:    n.StatusNasabah,
			NomorRekening: n.NomorRekening,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Nasabah fetched successfully",
		"data":    items,
	})
}

func (bc *BankController) GetAllBankSampah(c *gin.Context) {
	type JadwalPenimbangan struct{
		Hari string `json:"hari"`
		MingguKe int `json:"minggu_ke"`
		JamMulai string `json:"jam_mulai"`
		JamSelesai string `json:"jam_selesai"`
	}

	type BankSampahResponse struct{
		BankID string `json:"bank_id"`
		NamaBank string `json:"nama_bank"`
		AlamatBank string `json:"alamat_bank"`
		Kecamatan string `json:"kecamatan"`
		JenisBank string `json:"jenis_bank"`
		Kelurahan string `json:"kelurahan"`
		KabupatenKota string `json:"kabupaten_kota"`
		Provinsi string `json:"provinsi"`
		PhotoURL string `json:"photo_url"`
		Latitude float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Jadwal []JadwalPenimbangan `json:"jadwal_penimbangan"`
	}

	var banks []models.BankSampah
	// Mengambil bank sampah yang aktif dan me-preload jadwal penimbangan rutin yang aktif
	if err := bc.DB.
		Preload("Kecamatan").
		Preload("Kelurahan").
		Preload("Jadwals", "is_active = ? AND is_rutin = ? AND jenis_jadwal = ?", true, true, models.JadwalPenimbangan).
		Where("is_active = ?", true).
		Find(&banks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get all banks: " + err.Error()})
		return
	}

	var responseData []BankSampahResponse
	for _, bank := range banks {
		var jadwalPenimbangan []JadwalPenimbangan
		for _, j := range bank.Jadwals {
			jadwalPenimbangan = append(jadwalPenimbangan, JadwalPenimbangan{
				Hari:       string(j.Hari),
				MingguKe:   j.MingguKe,
				JamMulai:   j.JamMulai,
				JamSelesai: j.JamSelesai,
			})
		}

		if jadwalPenimbangan == nil {
			jadwalPenimbangan = []JadwalPenimbangan{}
		}

		kecamatanStr := ""
		if bank.Kecamatan != nil {
			kecamatanStr = bank.Kecamatan.Kecamatan
		}
		kelurahanStr := ""
		if bank.Kelurahan != nil {
			kelurahanStr = bank.Kelurahan.Kelurahan
		}

		responseData = append(responseData, BankSampahResponse{
			BankID:        bank.BankID,
			NamaBank:      bank.NamaBank,
			AlamatBank:    bank.Alamat,
			JenisBank: string(bank.JenisBank),
			Kecamatan:     kecamatanStr,
			Kelurahan:     kelurahanStr,
			KabupatenKota: bank.KabupatenKota,
			Provinsi:      bank.Provinsi,
			PhotoURL:      bank.PhotoURL,
			Latitude:      bank.Latitude,
			Longitude:     bank.Longitude,
			Jadwal:        jadwalPenimbangan,
		})
	}

	if responseData == nil {
		responseData = []BankSampahResponse{}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "All active banks fetched successfully",
		"data":    responseData,
	})
}

func (bc *BankController) EditProfilBankSampah(c *gin.Context) {
	bankID := c.Param("bank_id")

	adminID := c.PostForm("admin_id")
	if adminID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Admin ID wajib diisi"})
		return
	}

	var bank models.BankSampah
	if err := bc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	oldValue := map[string]interface{}{
		"nama_bank": bank.NamaBank,
		"photo_url": bank.PhotoURL,
		"alamat":    bank.Alamat,
		"latitude":  bank.Latitude,
		"longitude": bank.Longitude,
		"deskripsi": bank.Deskripsi,
	}

	if v := c.PostForm("nama_bank"); v != "" {
		bank.NamaBank = v
	}
	if v := c.PostForm("alamat"); v != "" {
		bank.Alamat = v
	}
	if v := c.PostForm("deskripsi"); v != "" {
		bank.Deskripsi = v
	}

	var latF, lngF float64
	latStr := c.PostForm("latitude")
	lngStr := c.PostForm("longitude")
	if latStr != "" {
		if _, err := fmt.Sscanf(latStr, "%f", &latF); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format latitude tidak valid"})
			return
		}
		bank.Latitude = latF
	}
	if lngStr != "" {
		if _, err := fmt.Sscanf(lngStr, "%f", &lngF); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format longitude tidak valid"})
			return
		}
		bank.Longitude = lngF
	}

	fileHeader, err := c.FormFile("foto_profil")
	if err == nil {
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuka file foto"})
			return
		}
		defer file.Close()

		photoURL, err := bc.CFStorage.UploadFile(file, fileHeader, "bank")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal upload foto profil: " + err.Error()})
			return
		}
		bank.PhotoURL = photoURL
	}

	newValue := map[string]interface{}{
		"nama_bank": bank.NamaBank,
		"photo_url": bank.PhotoURL,
		"alamat":    bank.Alamat,
		"latitude":  bank.Latitude,
		"longitude": bank.Longitude,
		"deskripsi": bank.Deskripsi,
	}

	oldVal, _ := json.Marshal(oldValue)
	newVal, _ := json.Marshal(newValue)

	txErr := bc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&bank).Error; err != nil {
			return err
		}

		history := models.HistoryAkunBank{
			BankID:     bankID,
			Action:     "UPDATE",
			OldValue:   datatypes.JSON(oldVal),
			NewValue:   datatypes.JSON(newVal),
			Informasi:  "Update profil bank sampah",
			Keterangan: "Admin memperbarui informasi profil bank sampah",
			CreatedBy:  adminID,
		}
		return tx.Create(&history).Error
	})

	if txErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui profil bank"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profil bank berhasil diperbarui",
		"data":    bank,
	})
}