package controllers

import (
	"enviroo-be/internal/models"
	"enviroo-be/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserController struct {
	DB     *gorm.DB
	Mailer *utils.Mailer
}

func NewUserController(db *gorm.DB, mailer *utils.Mailer) *UserController {
	return &UserController{
		DB:     db,
		Mailer: mailer,
	}
}

type AddUserRequest struct {
	UserID     string `json:"user_id" binding:"required"`
	Nama       string `json:"nama" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	NoWhatsapp string `json:"no_whatsapp" binding:"required"`
}

func (uc *UserController) AddUser(c *gin.Context) {
	var req AddUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Cek apakah NIK sudah dipakai
	var existingUser models.User
	if err := uc.DB.Where("user_id = ?", req.UserID).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User dengan NIK tersebut sudah terdaftar"})
		return
	}

	newUser := models.User{
		UserID:     req.UserID,
		Nama:       req.Nama,
		Email:      req.Email,
		NoWhatsapp: req.NoWhatsapp,
	}

	if err := uc.DB.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
		"data":    newUser,
	})
}

func (uc *UserController) GetNonAdminUser(c *gin.Context) {
	var users []models.User
	
	// Cari semua user di mana user_id mereka tidak ada dalam tabel admin sama sekali
	subQuery := uc.DB.Model(&models.Admin{}).Select("user_id")
	
	if err := uc.DB.Where("user_id NOT IN (?)", subQuery).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Users fetched successfully",
		"data":    users,
	})
}

func (uc *UserController) GetNonNasabahUser(c *gin.Context) {
	var users []models.User
	
	// Cari semua user di mana user_id mereka tidak ada dalam tabel nasabah sama sekali
	subQuery := uc.DB.Model(&models.Nasabah{}).Select("user_id")
	
	if err := uc.DB.Where("user_id NOT IN (?)", subQuery).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Users fetched successfully",
		"data":    users,
	})
}

func (uc *UserController) GetNonAdminBankSampah(c *gin.Context) {
	var users []models.User
	bankID := c.Param("bank_id")

	subQueryAdmin := uc.DB.Model(&models.Admin{}).Select("user_id")
	subQueryNasabah := uc.DB.Model(&models.Nasabah{}).Select("user_id").Where("bank_id = ?", bankID)
	
	err := uc.DB.
		Where("user_id NOT IN (?)", subQueryAdmin).
		Where("user_id NOT IN (?)", subQueryNasabah).
		Find(&users).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Users fetched successfully",
		"data":    users,
	})
}

func (uc *UserController) ActiveAdmin(c *gin.Context) {
	adminID := c.Param("admin_id")
	
	type ActiveAdminResponse struct{
		BankID string `json:"bank_id"`
		NamaBank string `json:"nama_bank"`
		JenisBank models.JenisBank `json:"jenis_bank"`
		PhotoURL string `json:"photo_url"`
	}

	var result ActiveAdminResponse
	if err := uc.DB.Model(&models.Admin{}).
		Select("admin.bank_id", "bank_sampah.nama_bank", "bank_sampah.jenis_bank", "bank_sampah.photo_url").
		Where("admin.admin_id = ?", adminID).
		Joins("JOIN bank_sampah ON admin.bank_id = bank_sampah.bank_id").
		Scan(&result).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get admin: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Current active Admin fetch successfully",
		"data":    result,
	})
}

func (uc *UserController) TesKirimEmail(c *gin.Context) {
	err := uc.Mailer.SendEmail(utils.EmailParams{
		To:      "annincarista@gmail.com",
		Subject: "Test Email Enviroo",
		Body:    "<h1>Halo!</h1><p>Ini adalah email percobaan dari sistem Enviroo.</p>",
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengirim email: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Email percobaan berhasil dikirim ke annincarista@gmail.com",
	})
}

func (uc *UserController) GetNonNasabahNonAdminBSI(c *gin.Context) {
	bankID := c.Param("bank_id")
	var users []models.User

	// Cek Admin lokal di bank ini (mencegah double role admin & nasabah di bank yang sama)
	subQueryAdmin := uc.DB.Model(&models.Admin{}).Select("user_id").Where("bank_id = ?", bankID)
	
	// Cek Nasabah secara global (user belum pernah jadi nasabah di manapun)
	subQueryNasabah := uc.DB.Model(&models.Nasabah{}).Select("user_id")

	err := uc.DB.
		Where("user_id NOT IN (?)", subQueryAdmin).
		Where("user_id NOT IN (?)", subQueryNasabah).
		Find(&users).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Users fetched successfully",
		"data":    users,
	})
}

func (uc *UserController) ActivePetugas(c *gin.Context) {
	adminID := c.Param("admin_id")

	// Model response khusus agar mudah dibaca di Flutter (Flat JSON)
	type PetugasResponse struct {
		AdminID     string `json:"admin_id"`
		Role        string `json:"role"`
		StatusAdmin string `json:"status_admin"`
		// User fields
		UserID   string `json:"user_id"`
		Nama     string `json:"nama"`
		Email    string `json:"email"`
		PhotoURL string `json:"photo_url"`
		// Bank fields
		BankID    string `json:"bank_id"`
		NamaBank  string `json:"nama_bank"`
		JenisBank string `json:"jenis_bank"`
	}

	var result PetugasResponse

	// Join 3 tabel: admin, users, dan bank_sampah
	err := uc.DB.Table("admin").
		Select("admin.admin_id, admin.role, admin.status_admin, users.user_id, users.nama, users.email, users.photo_url, bank_sampah.bank_id, bank_sampah.nama_bank, bank_sampah.jenis_bank").
		Joins("JOIN users ON users.user_id = admin.user_id").
		Joins("LEFT JOIN bank_sampah ON bank_sampah.bank_id = admin.bank_id").
		Where("admin.admin_id = ?", adminID).
		Scan(&result).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data petugas: " + err.Error()})
		return
	}

	// Cek jika data tidak ditemukan (AdminID tidak valid)
	if result.AdminID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Petugas tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data petugas aktif berhasil diambil",
		"data":    result,
	})
}
