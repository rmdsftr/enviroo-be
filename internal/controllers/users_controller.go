package controllers

import (
	"enviroo-be/internal/middleware"
	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserController struct {
	DB        *gorm.DB
	Mailer    *utils.Mailer
	CFStorage *storage.CloudflareStorage
}

func NewUserController(db *gorm.DB, mailer *utils.Mailer, cfStorage *storage.CloudflareStorage) *UserController {
	return &UserController{
		DB:        db,
		Mailer:    mailer,
		CFStorage: cfStorage,
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

	// Cek apakah email sudah dipakai
	if err := uc.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email sudah digunakan"})
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

func (uc *UserController) UpdateProfilUser(c *gin.Context) {
	userID := c.Param("user_id")

	nama := c.PostForm("nama")
	noWhatsapp := c.PostForm("no_whatsapp")
	fileHeader, _ := c.FormFile("photo_profile")

	if nama == "" && noWhatsapp == "" && fileHeader == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Minimal satu field harus diisi (nama, no_whatsapp, atau photo_profile)"})
		return
	}

	var user models.User
	if err := uc.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data user: " + err.Error()})
		}
		return
	}

	updates := map[string]interface{}{}

	if nama != "" {
		updates["nama"] = nama
	}
	if noWhatsapp != "" {
		updates["no_whatsapp"] = noWhatsapp
	}

	if fileHeader != nil {
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuka file foto"})
			return
		}
		defer file.Close()

		photoURL, err := uc.CFStorage.UploadFile(file, fileHeader, "profile")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal upload foto profil: " + err.Error()})
			return
		}
		updates["photo_url"] = photoURL
	}

	if err := uc.DB.Model(&user).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update profil: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profil berhasil diperbarui",
		"data":    user,
	})
}

func (uc *UserController) GetNonAdminUser(c *gin.Context) {
	var users []models.User

	subQuery := uc.DB.Model(&models.Admin{}).Select("user_id").
		Where("status_admin IN ?", []models.StatusAkun{models.Aktif, models.Pending})

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

	subQuery := uc.DB.Model(&models.Nasabah{}).Select("user_id").
		Where("status_nasabah IN ?", []models.StatusAkun{models.Aktif, models.Pending})

	if err := uc.DB.Where("user_id NOT IN (?)", subQuery).Find(&users).Error; err != nil {
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

	type ActiveAdminResponse struct {
		BankID    string           `json:"bank_id"`
		NamaBank  string           `json:"nama_bank"`
		JenisBank models.JenisBank `json:"jenis_bank"`
		PhotoURL  string           `json:"photo_url"`
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

	if result.BankID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Current active Admin fetch successfully",
		"data":    result,
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

func (uc *UserController) ActiveUser(c *gin.Context) {
	userID := c.Param("user_id")

	type UserActiveResponse struct {
		UserID   string `json:"user_id" gorm:"column:user_id"`
		Nama     string `json:"nama" gorm:"column:nama"`
		PhotoURL string `json:"photo_url" gorm:"column:photo_url"`
	}

	var result UserActiveResponse

	err := uc.DB.Table("users").
		Select("user_id, nama, photo_url").
		Where("user_id = ?", userID).
		First(&result).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data user"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data user berhasil diambil",
		"data":    result,
	})
}

type AddSuperadminRequest struct {
	UserID     string `json:"user_id" binding:"required"`
	Nama       string `json:"nama" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	NoWhatsapp string `json:"no_whatsapp" binding:"required"`
}

func (uc *UserController) AddSuperadmin(c *gin.Context) {
	var req AddSuperadminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existingUser models.User
	if err := uc.DB.Where("user_id = ?", req.UserID).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User dengan NIK tersebut sudah terdaftar"})
		return
	}
	if err := uc.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email sudah digunakan"})
		return
	}

	tx := uc.DB.Begin()

	user := models.User{
		UserID:     req.UserID,
		Nama:       utils.ToTitleCase(req.Nama),
		Email:      req.Email,
		NoWhatsapp: req.NoWhatsapp,
	}
	if err := tx.Omit(clause.Associations).Create(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat user: " + err.Error()})
		return
	}

	adminID, err := utils.GenerateAdminID(tx, models.SuperAdmin, "")
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal generate admin ID: " + err.Error()})
		return
	}

	admin := models.Admin{
		AdminID:     adminID,
		UserID:      req.UserID,
		BankID:      nil,
		Role:        models.SuperAdmin,
		StatusAdmin: models.Pending,
	}
	if err := tx.Omit(clause.Associations).Create(&admin).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat admin: " + err.Error()})
		return
	}

	otp, err := utils.GenerateOTP()
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal generate OTP: " + err.Error()})
		return
	}
	hashedOTP, err := utils.HashOTP(otp)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hash OTP: " + err.Error()})
		return
	}

	now := time.Now()
	aktivasiID := utils.GenerateAktivasiID(req.UserID)
	newAktivasiAkun := models.AktivasiAkun{
		AktivasiID:  aktivasiID,
		UserID:      req.UserID,
		AsRole:      models.RoleUserAdmin,
		Token:       hashedOTP,
		ExpiredAt:   now.Add(24 * time.Hour),
		GeneratedBy: "system",
		Tujuan:      models.TujuanAktivasi,
	}
	if err := tx.Create(&newAktivasiAkun).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat aktivasi akun: " + err.Error()})
		return
	}

	emailBody := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; background-color: #f4fdf4; padding: 30px; border-radius: 10px;">
			<h2 style="color: #4ea771; margin-top: 0;">Halo, %s!</h2>
			<p style="font-size: 14px; color: #333; line-height: 1.5;">
				Anda telah ditunjuk sebagai <b>Superadmin</b> dalam aplikasi Enviroo.
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
	`, user.Nama, otp, newAktivasiAkun.ExpiredAt.Format("02 Jan 2006 15:04 WIB"))

	if sendErr := uc.Mailer.SendEmail(utils.EmailParams{
		To:      user.Email,
		Subject: "Aktivasi Akun Superadmin Enviroo",
		Body:    emailBody,
	}); sendErr != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengirim email: " + sendErr.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Superadmin berhasil ditambahkan",
		"data": gin.H{
			"user_id":  user.UserID,
			"admin_id": admin.AdminID,
			"nama":     user.Nama,
			"email":    user.Email,
			"role":     admin.Role,
		},
	})
}

// GET /superadmin/list
func (uc *UserController) GetListSuperadmin(c *gin.Context) {
	type SuperadminItem struct {
		AdminID     string    `gorm:"column:admin_id"     json:"admin_id"`
		UserID      string    `gorm:"column:user_id"      json:"user_id"`
		Nama        string    `gorm:"column:nama"         json:"nama"`
		Email       string    `gorm:"column:email"        json:"email"`
		NoWhatsapp  string    `gorm:"column:no_whatsapp"  json:"no_whatsapp"`
		PhotoURL    string    `gorm:"column:photo_url"    json:"photo_url"`
		StatusAdmin string    `gorm:"column:status_admin" json:"status_admin"`
		JoinedAt    time.Time `gorm:"column:joined_at"    json:"joined_at"`
	}

	var list []SuperadminItem
	if err := uc.DB.Table("admin").
		Select("admin.admin_id, admin.status_admin, admin.joined_at, users.user_id, users.nama, users.email, users.no_whatsapp, users.photo_url").
		Joins("JOIN users ON users.user_id = admin.user_id").
		Where("admin.role = ?", models.SuperAdmin).
		Order("admin.joined_at ASC").
		Scan(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar superadmin: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Daftar superadmin berhasil diambil",
		"data":    list,
	})
}

// PATCH /superadmin/nonaktif/:admin_id
func (uc *UserController) NonaktifkanSuperadmin(c *gin.Context) {
	targetAdminID := c.Param("admin_id")

	// Ambil user yang sedang login untuk cegah self-deactivation
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return
	}

	var target models.Admin
	if err := uc.DB.Where("admin_id = ? AND role = ?", targetAdminID, models.SuperAdmin).First(&target).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Superadmin tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data: " + err.Error()})
		}
		return
	}

	if target.UserID == claims.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tidak dapat menonaktifkan akun sendiri"})
		return
	}

	if target.StatusAdmin == models.Nonaktif {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Akun superadmin ini sudah nonaktif"})
		return
	}

	if err := uc.DB.Model(&target).Update("status_admin", models.Nonaktif).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menonaktifkan superadmin: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Superadmin berhasil dinonaktifkan",
		"admin_id": targetAdminID,
	})
}

func (uc *UserController) UpdateFCMToken(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return
	}

	var req struct {
		FCMToken string `json:"fcm_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := uc.DB.Model(&models.User{}).Where("user_id = ?", claims.UserID).Update("fcm_token", req.FCMToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update FCM token: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "FCM Token berhasil diperbarui",
	})
}

func (uc *UserController) LogAkun(c *gin.Context) {
	userID := c.Param("user_id")

	var user models.User
	if err := uc.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	roleLabel := func(role models.RoleAdmin) string {
		switch role {
		case models.SuperAdmin:
			return "Super Admin"
		case models.AdminBSI:
			return "Admin BSI"
		case models.AdminBSM:
			return "Admin BSM"
		case models.AdminBSU:
			return "Admin BSU"
		case models.PetugasBSI:
			return "Petugas BSI"
		case models.PetugasBSM:
			return "Petugas BSM"
		case models.PetugasBSU:
			return "Petugas BSU"
		default:
			return string(role)
		}
	}

	type AkunNasabah struct {
		NasabahID string            `json:"nasabah_id"`
		JoinedAt  time.Time         `json:"joined_at"`
		Status    models.StatusAkun `json:"status"`
		Afiliasi  string            `json:"afiliasi"`
	}

	type AkunAdmin struct {
		AdminID  string            `json:"admin_id"`
		Role     string            `json:"role"`
		JoinedAt time.Time         `json:"joined_at"`
		Status   models.StatusAkun `json:"status"`
		Afiliasi string            `json:"afiliasi"`
	}

	var nasabahList []models.Nasabah
	uc.DB.Preload("Bank").Where("user_id = ?", userID).Order("joined_at DESC").Find(&nasabahList)

	var adminList []models.Admin
	uc.DB.Preload("Bank").Where("user_id = ?", userID).Order("joined_at DESC").Find(&adminList)

	akunNasabah := make([]AkunNasabah, 0, len(nasabahList))
	for _, n := range nasabahList {
		akunNasabah = append(akunNasabah, AkunNasabah{
			NasabahID: n.NasabahID,
			JoinedAt:  n.JoinedAt,
			Status:    n.StatusNasabah,
			Afiliasi:  n.Bank.NamaBank,
		})
	}

	akunAdmin := make([]AkunAdmin, 0, len(adminList))
	for _, a := range adminList {
		afiliasi := ""
		if a.Role == models.SuperAdmin {
			afiliasi = "Dinas Lingkungan Hidup Kota Padang"
		} else if a.BankID != nil {
			afiliasi = a.Bank.NamaBank
		}
		akunAdmin = append(akunAdmin, AkunAdmin{
			AdminID:  a.AdminID,
			Role:     roleLabel(a.Role),
			JoinedAt: a.JoinedAt,
			Status:   a.StatusAdmin,
			Afiliasi: afiliasi,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Log akun berhasil diambil",
		"data": gin.H{
			"user_id":      userID,
			"nama":         user.Nama,
			"akun_nasabah": akunNasabah,
			"akun_admin":   akunAdmin,
		},
	})
}

func (uc *UserController) GetDetailUser(c *gin.Context) {
	userID := c.Param("user_id")

	var user models.User
	if err := uc.DB.
		Preload("Nasabahs.Bank").
		Preload("Admins.Bank").
		Where("user_id = ?", userID).
		First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data user: " + err.Error()})
		}
		return
	}

	type AkunNasabah struct {
		NasabahID     string            `json:"nasabah_id"`
		NomorRekening string            `json:"nomor_rekening"`
		Status        models.StatusAkun `json:"status"`
		Afiliasi      string            `json:"afiliasi"`
		JoinedAt      time.Time         `json:"joined_at"`
	}

	type AkunAdmin struct {
		AdminID  string            `json:"admin_id"`
		Role     string            `json:"role"`
		Status   models.StatusAkun `json:"status"`
		Afiliasi string            `json:"afiliasi"`
		JoinedAt time.Time         `json:"joined_at"`
	}

	roleLabel := func(role models.RoleAdmin) string {
		switch role {
		case models.SuperAdmin:
			return "Super Admin"
		case models.AdminBSI:
			return "Admin BSI"
		case models.AdminBSM:
			return "Admin BSM"
		case models.AdminBSU:
			return "Admin BSU"
		case models.PetugasBSI:
			return "Petugas BSI"
		case models.PetugasBSM:
			return "Petugas BSM"
		case models.PetugasBSU:
			return "Petugas BSU"
		default:
			return string(role)
		}
	}

	akunNasabah := make([]AkunNasabah, 0)
	for _, n := range user.Nasabahs {
		akunNasabah = append(akunNasabah, AkunNasabah{
			NasabahID:     n.NasabahID,
			NomorRekening: n.NomorRekening,
			Status:        n.StatusNasabah,
			Afiliasi:      n.Bank.NamaBank,
			JoinedAt:      n.JoinedAt,
		})
	}

	akunAdmin := make([]AkunAdmin, 0)
	for _, a := range user.Admins {
		afiliasi := ""
		if a.Role == models.SuperAdmin {
			afiliasi = "Dinas Lingkungan Hidup Kota Padang"
		} else if a.BankID != nil {
			afiliasi = a.Bank.NamaBank
		}
		akunAdmin = append(akunAdmin, AkunAdmin{
			AdminID:  a.AdminID,
			Role:     roleLabel(a.Role),
			Status:   a.StatusAdmin,
			Afiliasi: afiliasi,
			JoinedAt: a.JoinedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail user berhasil diambil",
		"data": gin.H{
			"user_id":      user.UserID,
			"nama":         user.Nama,
			"email":        user.Email,
			"no_whatsapp":  user.NoWhatsapp,
			"photo_url":    user.PhotoURL,
			"created_at":   user.CreatedAt,
			"akun_nasabah": akunNasabah,
			"akun_admin":   akunAdmin,
		},
	})
}

func (uc *UserController) DeleteUser(c *gin.Context) {
	userID := c.Param("user_id")

	var user models.User
	if err := uc.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data user: " + err.Error()})
		}
		return
	}

	var nasabahCount int64
	if err := uc.DB.Model(&models.Nasabah{}).Where("user_id = ?", userID).Count(&nasabahCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa data nasabah: " + err.Error()})
		return
	}
	if nasabahCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "User tidak dapat dihapus karena sudah pernah terdaftar sebagai nasabah"})
		return
	}

	var adminCount int64
	if err := uc.DB.Model(&models.Admin{}).Where("user_id = ?", userID).Count(&adminCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa data admin: " + err.Error()})
		return
	}
	if adminCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "User tidak dapat dihapus karena sudah pernah terdaftar sebagai admin"})
		return
	}

	if err := uc.DB.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus user: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User berhasil dihapus"})
}

func (uc *UserController) GetAllUsers(c *gin.Context) {
	type UserResponse struct {
		UserID   string   `json:"user_id"`
		NamaUser string   `json:"nama_user"`
		Email    string   `json:"email"`
		Foto     string   `json:"foto"`
		Roles    []string `json:"roles"`
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "25"))
	if err != nil || limit < 1 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	var total int64
	if err := uc.DB.Model(&models.User{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung data pengguna: " + err.Error()})
		return
	}

	var users []models.User
	if err := uc.DB.Preload("Nasabahs").Preload("Admins").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data pengguna: " + err.Error()})
		return
	}

	response := make([]UserResponse, 0)

	for _, user := range users {
		var roles []string

		for _, nasabah := range user.Nasabahs {
			if nasabah.StatusNasabah == models.Aktif {
				roles = append(roles, "Nasabah")
				break
			}
		}

		for _, admin := range user.Admins {
			if admin.StatusAdmin != models.Aktif {
				continue
			}
			switch admin.Role {
			case models.SuperAdmin:
				roles = append(roles, "Super Admin")
			case models.AdminBSI:
				roles = append(roles, "Admin BSI")
			case models.AdminBSM:
				roles = append(roles, "Admin BSM")
			case models.AdminBSU:
				roles = append(roles, "Admin BSU")
			case models.PetugasBSI:
				roles = append(roles, "Petugas BSI")
			case models.PetugasBSM:
				roles = append(roles, "Petugas BSM")
			case models.PetugasBSU:
				roles = append(roles, "Petugas BSU")
			}
			break
		}

		if len(roles) == 0 {
			roles = append(roles, "Belum Terdaftar")
		}

		response = append(response, UserResponse{
			UserID:   user.UserID,
			NamaUser: user.Nama,
			Email:    user.Email,
			Foto:     user.PhotoURL,
			Roles:    roles,
		})
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	c.JSON(http.StatusOK, gin.H{
		"data": response,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total_items": total,
			"total_pages": totalPages,
		},
	})
}
