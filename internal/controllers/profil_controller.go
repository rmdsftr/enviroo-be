package controllers

import (
	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ProfilController struct {
	DB        *gorm.DB
	CFStorage *storage.CloudflareStorage
}

func NewProfilController(db *gorm.DB, cfStorage *storage.CloudflareStorage) *ProfilController {
	return &ProfilController{
		DB:        db,
		CFStorage: cfStorage,
	}
}

func (pc *ProfilController) GetProfilBankSampah(c *gin.Context) {
	bankID := c.Param("bank_id")

	type ProfilBankSampahResponse struct {
		BankID        string  `json:"bank_id" gorm:"column:bank_id"`
		NamaBank      string  `json:"nama_bank" gorm:"column:nama_bank"`
		JenisBank     string  `json:"jenis_bank" gorm:"column:jenis_bank"`
		Foto          string  `json:"foto" gorm:"column:photo_url"`
		Deskripsi     string  `json:"deskripsi" gorm:"column:deskripsi"`
		IsActive      bool    `json:"is_active" gorm:"column:is_active"`
		Provinsi      string  `json:"provinsi" gorm:"column:provinsi"`
		KabupatenKota string  `json:"kabupaten_kota" gorm:"column:kabupaten_kota"`
		Kecamatan     string  `json:"kecamatan" gorm:"column:kecamatan"`
		Alamat        string  `json:"alamat_lengkap" gorm:"column:alamat"`
		ParentID      *string `json:"parent_id" gorm:"column:parent_bank_id"`
		BankInduk     *string `json:"bank_induk" gorm:"column:bank_induk"`
	}

	var result ProfilBankSampahResponse

	query := pc.DB.Model(&models.BankSampah{}).
		Select("bank_sampah.bank_id, bank_sampah.nama_bank, bank_sampah.jenis_bank, bank_sampah.photo_url, bank_sampah.deskripsi, bank_sampah.is_active, bank_sampah.provinsi, bank_sampah.kabupaten_kota, bank_sampah.kecamatan, bank_sampah.alamat, bank_sampah.parent_bank_id, parent.nama_bank as bank_induk").
		Joins("LEFT JOIN bank_sampah as parent ON bank_sampah.parent_bank_id = parent.bank_id").
		Where("bank_sampah.bank_id = ?", bankID)

	if err := query.First(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get bank sampah profile: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bank sampah profile fetched successfully",
		"data":    result,
	})
}

func (pc *ProfilController) GetProfilNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	type ProfilNasabahResponse struct {
		NasabahID     string    `json:"nasabah_id" gorm:"column:nasabah_id"`
		BankID        string    `json:"bank_id" gorm:"column:bank_id"`
		UserID        string    `json:"user_id" gorm:"column:user_id"`
		JoinedAt      time.Time `json:"joined_at" gorm:"column:joined_at"`
		StatusNasabah string    `json:"status_nasabah" gorm:"column:status_nasabah"`
		NomorRekening string    `json:"nomor_rekening" gorm:"column:nomor_rekening"`

		Nama          string    `json:"nama" gorm:"column:nama"`
		Email         string    `json:"email" gorm:"column:email"`
		NoWhatsapp    string    `json:"no_whatsapp" gorm:"column:no_whatsapp"`
		PhotoURL      string    `json:"foto" gorm:"column:photo_url"`
		CreatedAt     time.Time `json:"created_at" gorm:"column:created_at"`
		UpdatedAt     time.Time `json:"updated_at" gorm:"column:updated_at"`

		BsiID   *string `json:"bsi_id" gorm:"column:bsi_id"`
		NamaBsi *string `json:"nama_bsi" gorm:"column:nama_bsi"`
		BsuID   *string `json:"bsu_id" gorm:"column:bsu_id"`
		NamaBsu *string `json:"nama_bsu" gorm:"column:nama_bsu"`

		SaldoPoin int     `json:"saldo_poin" gorm:"column:saldo_poin"`
	}

	var result ProfilNasabahResponse

	query := pc.DB.Model(&models.Nasabah{}).
		Select(`
			nasabah.nasabah_id, nasabah.bank_id, nasabah.user_id, nasabah.joined_at, nasabah.status_nasabah, nasabah.nomor_rekening,
			users.nama, users.email, users.no_whatsapp, users.photo_url, users.created_at, users.updated_at,
			CASE WHEN b.jenis_bank = 'bsu' THEN parent.bank_id ELSE b.bank_id END as bsi_id,
			CASE WHEN b.jenis_bank = 'bsu' THEN parent.nama_bank ELSE b.nama_bank END as nama_bsi,
			CASE WHEN b.jenis_bank = 'bsu' THEN b.bank_id ELSE NULL END as bsu_id,
			CASE WHEN b.jenis_bank = 'bsu' THEN b.nama_bank ELSE NULL END as nama_bsu,
			COALESCE(sn.total_poin, 0) as saldo_poin
		`).
		Joins("JOIN users ON nasabah.user_id = users.user_id").
		Joins("LEFT JOIN bank_sampah b ON nasabah.bank_id = b.bank_id").
		Joins("LEFT JOIN bank_sampah parent ON b.parent_bank_id = parent.bank_id").
		Joins("LEFT JOIN saldo_nasabah sn ON nasabah.nasabah_id = sn.nasabah_id").
		Where("nasabah.nasabah_id = ?", nasabahID)

	if err := query.First(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nasabah tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get nasabah profile: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Nasabah profile fetched successfully",
		"data":    result,
	})
}

func (pc *ProfilController) AktivasiBankSampah(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := pc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get bank sampah: " + err.Error()})
		}
		return
	}

	// Toggle nilai IsActive berdasarkan status saat ini di database
	targetActive := !bank.IsActive

	if err := pc.DB.Model(&bank).Update("is_active", targetActive).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bank sampah status: " + err.Error()})
		return
	}
	
	// Update instance secara lokal agar response merefleksikan nilai yang baru
	bank.IsActive = targetActive

	c.JSON(http.StatusOK, gin.H{
		"message": "Bank sampah status updated successfully",
		"data":    bank,
	})
}

func (pc *ProfilController) AktivasiNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	var nasabah models.Nasabah
	if err := pc.DB.Where("nasabah_id = ?", nasabahID).First(&nasabah).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nasabah tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get nasabah: " + err.Error()})
		}
		return
	}

	// Toggle nilai status berdasarkan status saat ini
	currentStatus := nasabah.StatusNasabah
	var targetStatus models.StatusAkun

	// Jika statusnya nonaktif atau pending, maka diaktifkan. Selain itu (jika aktif), maka dinonaktifkan.
	if currentStatus == models.Nonaktif || currentStatus == models.Pending {
		targetStatus = models.Aktif
	} else {
		targetStatus = models.Nonaktif
	}

	if err := pc.DB.Model(&nasabah).Update("status_nasabah", targetStatus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update nasabah status: " + err.Error()})
		return
	}

	// Update secara lokal instance nasabah
	nasabah.StatusNasabah = targetStatus

	c.JSON(http.StatusOK, gin.H{
		"message": "Nasabah status updated successfully",
		"data":    nasabah,
	})
}

func (pc *ProfilController) DeleteBankSampah(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := pc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get bank sampah: " + err.Error()})
		}
		return
	}

	if err := pc.DB.Delete(&bank).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete bank sampah: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bank sampah deleted successfully",
	})
}

func (pc *ProfilController) DeleteNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	var nasabah models.Nasabah
	if err := pc.DB.Where("nasabah_id = ?", nasabahID).First(&nasabah).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nasabah tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get nasabah: " + err.Error()})
		}
		return
	}

	if err := pc.DB.Delete(&nasabah).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete nasabah: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Nasabah deleted successfully",
	})
}

func (pc *ProfilController) GetProfilUser(c *gin.Context) {
	userID := c.Param("user_id")

	type ProfilUserResponse struct {
		models.User
		models.Admin
	}

	var result ProfilUserResponse
	// Gunakan Model(&models.User{}) agar GORM tahu tabel utamanya adalah 'users', 
	// lalu gunakan LEFT JOIN agar user tetap ketemu meskipun dia bukan admin.
	if err := pc.DB.Model(&models.User{}).
		Select("users.*, admin.*").
		Joins("LEFT JOIN admin ON admin.user_id = users.user_id").
		Where("users.user_id = ?", userID).
		First(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User profile fetched successfully",
		"data":    result,
	})
}

func (pc *ProfilController) GetHistoryAkunBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	type HistoryResponse struct {
		HistoryBankID uuid.UUID      `json:"history_bank_id" gorm:"column:history_bank_id"`
		BankID        string         `json:"bank_id" gorm:"column:bank_id"`
		Action        string         `json:"action" gorm:"column:action"`
		OldValue      datatypes.JSON `json:"old_value" gorm:"column:old_value"`
		NewValue      datatypes.JSON `json:"new_value" gorm:"column:new_value"`
		Informasi     string         `json:"informasi" gorm:"column:informasi"`
		Keterangan    string         `json:"keterangan" gorm:"column:keterangan"`
		CreatedAt     time.Time      `json:"created_at" gorm:"column:created_at"`
		CreatedByName string         `json:"created_by_name" gorm:"column:created_by_name"`
	}

	var result []HistoryResponse

	query := pc.DB.Model(&models.HistoryAkunBank{}).
		Select(`
			history_akun_bank.history_bank_id, 
			history_akun_bank.bank_id, 
			history_akun_bank.action, 
			history_akun_bank.old_value, 
			history_akun_bank.new_value, 
			history_akun_bank.informasi, 
			history_akun_bank.keterangan, 
			history_akun_bank.created_at, 
			users.nama as created_by_name
		`).
		Joins("LEFT JOIN admin ON history_akun_bank.created_by = admin.admin_id").
		Joins("LEFT JOIN users ON admin.user_id = users.user_id").
		Where("history_akun_bank.bank_id = ?", bankID).
		Order("history_akun_bank.created_at desc")

	if err := query.Find(&result).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get history: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "History bank fetched successfully",
		"data":    result,
	})
}
