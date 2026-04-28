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

type NasabahController struct {
	DB        *gorm.DB
	CFStorage *storage.CloudflareStorage
	Mailer    *utils.Mailer
}

func NewNasabahController(db *gorm.DB, cfStorage *storage.CloudflareStorage, mailer *utils.Mailer) *NasabahController {
	return &NasabahController{
		DB:        db,
		CFStorage: cfStorage,
		Mailer:    mailer,
	}
}

type AddNasabahRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Nama string `json:"nama" binding:"required"`
	Email string `json:"email" binding:"required,email"`
	NoWhatsapp string `json:"no_whatsapp" binding:"required"`
	BankID string `json:"bank_id" binding:"required"`
	AdminID string `json:"admin_id" binding:"required"`
}

func (nc *NasabahController) AddNewNasabah(c *gin.Context) {
	var req AddNasabahRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var adminCount int64
	nc.DB.Model(&models.Admin{}).Where("user_id = ? AND bank_id = ?", req.UserID, req.BankID).Count(&adminCount)
	if adminCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Admin bank sampah tidak boleh menjadi nasabah di bank sampah yang sama"})
		return
	}

	var userCount int64
	nc.DB.Model(&models.User{}).Where("user_id = ?", req.UserID).Count(&userCount)
	if userCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "User dengan NIK/ID ini sudah terdaftar. Gunakan menu pilih akun lama."})
		return
	}

	// Ambil data BankSampah tempat Nasabah mendaftar
	var bank models.BankSampah
	if err := nc.DB.Where("bank_id = ?", req.BankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	// Untuk NasabahID, tentukan reference bank (jika BSU ambil ParentBankID dari BSI, jika bukan tetap ID sendirinya)
	referenceBankID := bank.BankID
	if bank.JenisBank == models.BSU && bank.ParentBankID != nil {
		referenceBankID = *bank.ParentBankID
	}

	tx := nc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database"})
		return
	}

	newUser := models.User{
		UserID:     req.UserID,
		Nama:       req.Nama,
		Email:      req.Email,
		NoWhatsapp: req.NoWhatsapp,
	}

	if err := tx.Create(&newUser).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user: " + err.Error()})
		return
	}

	newNasabah := models.Nasabah{
		NasabahID:     utils.GenerateNasabahID(bank.JenisBank, referenceBankID), // Generate ID/No Rekening
		UserID:        req.UserID,
		BankID:        req.BankID,
		StatusNasabah: models.Pending,
	}

	if err := tx.Create(&newNasabah).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create nasabah: " + err.Error()})
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
	aktivasiID := utils.GenerateAktivasiID(req.UserID)

	newAktivasiAkun := models.AktivasiAkun{
		AktivasiID:  aktivasiID,
		UserID:      req.UserID,
		AsRole:      models.RoleUserNasabah,
		Token:       hashedOTP,
		ExpiredAt:   now.Add(24 * time.Hour),
		GeneratedBy: req.AdminID,
	}

	if err := tx.Create(&newAktivasiAkun).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create aktivasi akun: " + err.Error()})
		return
	}

	// Kirim email persis sebelum Commit
	emailBody := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; background-color: #f4fdf4; padding: 30px; border-radius: 10px;">
			<h2 style="color: #4ea771; margin-top: 0;">Halo, %s!</h2>
			<p style="font-size: 14px; color: #333; line-height: 1.5;">
				Terima kasih telah mendaftar sebagai Nasabah di Bank Sampah <b>%s</b>.
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
	`, req.Nama, bank.NamaBank, otp, newAktivasiAkun.ExpiredAt.Format("02 Jan 2006 15:04 WIB"))

	sendErr := nc.Mailer.SendEmail(utils.EmailParams{
		To:      req.Email,
		Subject: "Aktivasi Akun Nasabah Enviroo",
		Body:    emailBody,
	})

	if sendErr != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email: " + sendErr.Error()})
		return
	}

	// Semua beres, simpan permanen
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Nasabah created successfully. Activation email sent.",
		"data":    newNasabah,
	})
}

func (nc *NasabahController) GetAfiliasi(c *gin.Context) {
	var afiliasi []models.BankSampah
	if err := nc.DB.Find(&afiliasi).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get afiliasi: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Afiliasi retrieved successfully",
		"data":    afiliasi,
	})
}

func (nc *NasabahController) GetNasabah(c *gin.Context) {
	type NasabahResponse struct {
		NasabahID       string `json:"nasabah_id" gorm:"column:nasabah_id"`
		NamaNasabah     string `json:"nama_nasabah" gorm:"column:nama_nasabah"`
		Foto            string `json:"foto" gorm:"column:foto"`
		BankSampahPusat string `json:"bank_sampah_pusat" gorm:"column:bank_sampah_pusat"`
		BankSampahUnit  string `json:"bank_sampah_unit" gorm:"column:bank_sampah_unit"`
		StatusNasabah   string `json:"status_nasabah" gorm:"column:status_nasabah"`
	}

	var results []NasabahResponse

	query := nc.DB.Table("nasabah").
		Select("nasabah.nasabah_id, u.nama AS nama_nasabah, u.photo_url AS foto, nasabah.status_nasabah, " +
			"CASE WHEN b.jenis_bank = 'bsu' THEN p.nama_bank ELSE b.nama_bank END AS bank_sampah_pusat, " +
			"CASE WHEN b.jenis_bank = 'bsu' THEN b.nama_bank ELSE '' END AS bank_sampah_unit").
		Joins("LEFT JOIN users u ON nasabah.user_id = u.user_id").
		Joins("LEFT JOIN bank_sampah b ON nasabah.bank_id = b.bank_id").
		Joins("LEFT JOIN bank_sampah p ON b.parent_bank_id = p.bank_id")

	if err := query.Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get nasabah: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Nasabah retrieved successfully",
		"data":    results,
	})
}

type AddNasabahOldUserRequest struct {
	UserID  string `json:"user_id" binding:"required"`
	BankID  string `json:"bank_id" binding:"required"`
	AdminID string `json:"admin_id" binding:"required"`
}

func (nc *NasabahController) AddNewNasabahOldUser(c *gin.Context) {
	var req AddNasabahOldUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var adminCount int64
	nc.DB.Model(&models.Admin{}).Where("user_id = ? AND bank_id = ?", req.UserID, req.BankID).Count(&adminCount)
	if adminCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Admin bank sampah tidak boleh menjadi nasabah di bank sampah yang sama"})
		return
	}

	var existingUser models.User
	if err := nc.DB.Where("user_id = ?", req.UserID).First(&existingUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User dengan ID ini tidak ditemukan"})
		return
	}

	var nasabahCount int64
	nc.DB.Model(&models.Nasabah{}).Where("user_id = ? AND bank_id = ?", req.UserID, req.BankID).Count(&nasabahCount)
	if nasabahCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "User sudah terdaftar sebagai nasabah di bank sampah ini"})
		return
	}

	// Ambil data BankSampah tempat Nasabah mendaftar
	var bank models.BankSampah
	if err := nc.DB.Where("bank_id = ?", req.BankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	// Untuk NasabahID, tentukan reference bank (jika BSU ambil ParentBankID dari BSI, jika bukan tetap ID sendirinya)
	referenceBankID := bank.BankID
	if bank.JenisBank == models.BSU && bank.ParentBankID != nil {
		referenceBankID = *bank.ParentBankID
	}

	tx := nc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database"})
		return
	}

	newNasabah := models.Nasabah{
		NasabahID:     utils.GenerateNasabahID(bank.JenisBank, referenceBankID), // Generate ID/No Rekening
		UserID:        req.UserID,
		BankID:        req.BankID,
		StatusNasabah: models.Pending,
	}

	if err := tx.Create(&newNasabah).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create nasabah: " + err.Error()})
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
	aktivasiID := utils.GenerateAktivasiID(req.UserID)

	newAktivasiAkun := models.AktivasiAkun{
		AktivasiID:  aktivasiID,
		UserID:      req.UserID,
		AsRole:      models.RoleUserNasabah,
		Token:       hashedOTP,
		ExpiredAt:   now.Add(24 * time.Hour),
		GeneratedBy: req.AdminID,
	}

	if err := tx.Create(&newAktivasiAkun).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create aktivasi akun: " + err.Error()})
		return
	}

	// Kirim email
	emailBody := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; background-color: #f4fdf4; padding: 30px; border-radius: 10px;">
			<h2 style="color: #4ea771; margin-top: 0;">Halo, %s!</h2>
			<p style="font-size: 14px; color: #333; line-height: 1.5;">
				Terima kasih telah mendaftar sebagai Nasabah di Bank Sampah <b>%s</b>.
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
	`, existingUser.Nama, bank.NamaBank, otp, newAktivasiAkun.ExpiredAt.Format("02 Jan 2006 15:04 WIB"))

	sendErr := nc.Mailer.SendEmail(utils.EmailParams{
		To:      existingUser.Email,
		Subject: "Aktivasi Akun Nasabah Enviroo",
		Body:    emailBody,
	})

	if sendErr != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email: " + sendErr.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Nasabah created successfully. Activation email sent.",
		"data":    newNasabah,
	})
}

func (nc *NasabahController) NasabahBankSampah(c *gin.Context) {
	bankID := c.Param("bank_id")

	type NasabahBankSampahResponse struct {
		NasabahID       string `json:"nasabah_id" gorm:"column:nasabah_id"`
		NamaNasabah     string `json:"nama_nasabah" gorm:"column:nama_nasabah"`
		Foto            string `json:"foto" gorm:"column:foto"`
		BankSampahPusat string `json:"bank_sampah_pusat" gorm:"column:bank_sampah_pusat"`
		BankSampahUnit  string `json:"bank_sampah_unit" gorm:"column:bank_sampah_unit"`
		StatusNasabah   string `json:"status_nasabah" gorm:"column:status_nasabah"`
	}

	var results []NasabahBankSampahResponse

	query := nc.DB.Table("nasabah").
		Select("nasabah.nasabah_id, u.nama AS nama_nasabah, u.photo_url AS foto, nasabah.status_nasabah, " +
			"CASE WHEN b.jenis_bank = 'bsu' THEN p.nama_bank ELSE b.nama_bank END AS bank_sampah_pusat, " +
			"CASE WHEN b.jenis_bank = 'bsu' THEN b.nama_bank ELSE '' END AS bank_sampah_unit").
		Joins("LEFT JOIN users u ON nasabah.user_id = u.user_id").
		Joins("LEFT JOIN bank_sampah b ON nasabah.bank_id = b.bank_id").
		Joins("LEFT JOIN bank_sampah p ON b.parent_bank_id = p.bank_id").
		Where("nasabah.bank_id = ?", bankID)

	if err := query.Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get nasabah: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Nasabah retrieved successfully",
		"data":    results,
	})
}
