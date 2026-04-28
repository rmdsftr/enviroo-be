package controllers

import (
	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BSMController struct {
	DB        *gorm.DB
	CFStorage *storage.CloudflareStorage
	Mailer    *utils.Mailer
}

func NewBSMController(db *gorm.DB, cfStorage *storage.CloudflareStorage, mailer *utils.Mailer) *BSMController {
	return &BSMController{
		DB:        db,
		CFStorage: cfStorage,
		Mailer:    mailer,
	}
}

type AddBSMRequest struct {
	NamaBSM       string   `form:"nama_bsm" binding:"required"`
	Deskripsi     string   `form:"deskripsi" binding:"required"`
	Provinsi      string   `form:"provinsi" binding:"required"`
	KabupatenKota string   `form:"kabupaten_kota" binding:"required"`
	Kecamatan     string   `form:"kecamatan" binding:"required"`
	AlamatLengkap string   `form:"alamat_lengkap" binding:"required"`
	Latitude      float64  `form:"latitude"`
	Longitude     float64  `form:"longitude"`
	UserIDs       []string `form:"user_id[]"`
	AdminID       string   `form:"admin_id" binding:"required"`
}

func (bc *BSMController) AddNewBSM(c *gin.Context) {
	var req AddBSMRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var fotoURL string
	fileHeader, err := c.FormFile("foto")
	if err == nil {
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca foto"})
			return
		}
		defer file.Close()

		fotoURL, err = bc.CFStorage.UploadFile(file, fileHeader, "bsm_photos")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupload foto ke cloud: " + err.Error()})
			return
		}
	}

	tx := bc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	newBSM := models.BankSampah{
		NamaBank:      req.NamaBSM,
		Deskripsi:     req.Deskripsi,
		PhotoURL:      fotoURL,
		Provinsi:      req.Provinsi,
		KabupatenKota: req.KabupatenKota,
		Kecamatan:     req.Kecamatan,
		Alamat:        req.AlamatLengkap,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		JenisBank:     models.BSM,
		IsActive:      true,
	}

	if err := tx.Create(&newBSM).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create BSM: " + err.Error()})
		return
	}

	// Create admins for this BSM
	for _, userID := range req.UserIDs {
		// Pengecekan nasabah (meskipun bank baru terbuat, ini ditambahkan sesuai instruksi)
		var countNasabah int64
		if err := tx.Model(&models.Nasabah{}).Where("user_id = ? AND bank_id = ?", userID, newBSM.BankID).Count(&countNasabah).Error; err == nil && countNasabah > 0 {
			tx.Rollback()
			c.JSON(http.StatusConflict, gin.H{"error": "User tidak boleh terdaftar sebagai admin di bank sampah tempat ia terdaftar sebagai nasabah"})
			return
		}

		admin := models.Admin{
			AdminID:     utils.GenerateAdminID(),
			BankID:      &newBSM.BankID,
			UserID:      userID,
			Role:        models.AdminBSM,
			StatusAdmin: models.Pending,
		}

		if err := tx.Create(&admin).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add admin: " + err.Error()})
			return
		}

		// LOGIK AKTIVASI AKUN ADMIN
		var existingUser models.User
		if err := tx.Where("user_id = ?", userID).First(&existingUser).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user data for activation"})
			return
		}

		otp, err := utils.GenerateOTP()
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP: " + err.Error()})
			return
		}

		hashedOTP, err := utils.HashOTP(otp)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash OTP: " + err.Error()})
			return
		}

		now := time.Now()
		aktivasiID := utils.GenerateAktivasiID(userID)

		newAktivasiAkun := models.AktivasiAkun{
			AktivasiID:  aktivasiID,
			UserID:      userID,
			AsRole:      models.RoleUserAdmin,
			Token:       hashedOTP,
			ExpiredAt:   now.Add(24 * time.Hour),
			GeneratedBy: req.AdminID,
		}

		if err := tx.Create(&newAktivasiAkun).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create aktivasi akun: " + err.Error()})
			return
		}

		emailBody := fmt.Sprintf(`
			<div style="font-family: Arial, sans-serif; background-color: #f4fdf4; padding: 30px; border-radius: 10px;">
				<h2 style="color: #4ea771; margin-top: 0;">Halo, %s!</h2>
				<p style="font-size: 14px; color: #333; line-height: 1.5;">
					Anda telah ditunjuk sebagai <b>Administrator</b> di Bank Sampah <b>%s</b> (BSM).
					Untuk menyelesaikan proses aktivasi akun, gunakan kode OTP berikut:
				</p>
				<div style="background-color: #fff; border: 2px dashed #4ea771; padding: 15px; text-align: center; margin: 20px 0;">
					<h1 style="color: #4ea771; font-size: 32px; font-weight: bold; margin: 0; letter-spacing: 5px;">%s</h1>
				</div>
				<p style="font-size: 13px; color: #666;">
					Kode ini berlaku selama 24 jam hingga <b>%s</b>.
					Mohon untuk tidak membagikan kode OTP ini ke siapapun.
				</p>
				<hr style="border: 0; height: 1px; background: #ddd; margin: 25px 0;">
				<p style="font-size: 12px; color: #999; text-align: center; margin: 0;">&copy; Enviroo APP</p>
			</div>
		`, existingUser.Nama, newBSM.NamaBank, otp, newAktivasiAkun.ExpiredAt.Format("02 Jan 2006 15:04 WIB"))

		sendErr := bc.Mailer.SendEmail(utils.EmailParams{
			To:      existingUser.Email,
			Subject: "Aktivasi Akun Administrator (BSM) Enviroo",
			Body:    emailBody,
		})

		if sendErr != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email: " + sendErr.Error()})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "BSM and admins created successfully",
		"data":    newBSM,
	})
}

func (bc *BSMController) GetBSM(c *gin.Context) {
	type BSMResponse struct {
		models.BankSampah
		JumlahNasabah int64 `json:"jumlah_nasabah" gorm:"column:jumlah_nasabah"`
	}

	var results []BSMResponse

	// Query utama dengan subquery SELECT COUNT untuk menghitung jumlah nasabah di BSM ini
	query := bc.DB.Model(&models.BankSampah{}).
		Select("bank_sampah.*, (SELECT COUNT(user_id) FROM nasabah WHERE nasabah.bank_id = bank_sampah.bank_id) AS jumlah_nasabah").
		Where("bank_sampah.jenis_bank = ?", models.BSM)

	if err := query.Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get BSM: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "BSM fetched successfully",
		"data":    results,
	})
}