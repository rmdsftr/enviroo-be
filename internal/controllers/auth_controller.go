package controllers

import (
	"enviroo-be/internal/models"
	"enviroo-be/internal/middleware"
	"enviroo-be/pkg/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AuthController struct {
	DB     *gorm.DB
	Mailer *utils.Mailer
}

func NewAuthController(db *gorm.DB, mailer *utils.Mailer) *AuthController {
	return &AuthController{
		DB:     db,
		Mailer: mailer,
	}
}

type AddSuperadminRequest struct {
	UserID     string `json:"user_id" binding:"required"`
	Nama       string `json:"nama" binding:"required"`
	Email      string `json:"email" binding:"required,email"`
	NoWhatsapp string `json:"no_whatsapp" binding:"required"`
	Password   string `json:"password" binding:"required,min=8"`
}



func (ac *AuthController) AddSuperadmin(c *gin.Context) {
	var req AddSuperadminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	var existingUser models.User
	if err := ac.DB.Where("user_id = ?", req.UserID).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User dengan NIK tersebut sudah terdaftar"})
		return
	}

	// Check if email already used
	if err := ac.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email sudah digunakan"})
		return
	}

	// Hash password with bcrypt cost 12
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password"})
		return
	}

	// Begin transaction
	tx := ac.DB.Begin()

	// Create user
	user := models.User{
		UserID:     req.UserID,
		Nama:       utils.ToTitleCase(req.Nama),
		Email:      req.Email,
		NoWhatsapp: req.NoWhatsapp,
		Password:   string(hashedPassword),
	}

	if err := tx.Omit(clause.Associations).Create(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat user: " + err.Error()})
		return
	}

	// Create admin with superadmin role
	admin := models.Admin{
		AdminID:     utils.GenerateAdminID(),
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

	tx.Commit()

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

// LoginWeb memvalidasi kredensial lalu menyimpan JWT sebagai HttpOnly cookie.
func (ac *AuthController) Login(c *gin.Context) {
	var req struct {
		Email    string              `json:"email" binding:"required,email"`
		Password string              `json:"password" binding:"required,min=8"`
		Platform string              `json:"platform" binding:"required"` // "web" or "mobile"
		Role     models.RoleUserEnum `json:"role" binding:"required"`     // "nasabah" or "admin"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Cari User berdasarkan Email
	var user models.User
	if err := ac.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email atau password salah"})
		return
	}

	// 2. Verifikasi Password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email atau password salah"})
		return
	}

	var finalRole models.RoleAdmin
	var identityID string
	var bankID *string

	// 3. Alur Validasi berdasarkan Role & Platform
	switch req.Role {
	case models.RoleUserNasabah:
		// --- NASABAH BLOCK ---
		if req.Platform == "web" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Nasabah tidak memiliki akses ke dashboard web"})
			return
		}

		var nasabah models.Nasabah
		if err := ac.DB.Where("user_id = ?", user.UserID).First(&nasabah).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Akun Anda tidak terdaftar sebagai nasabah"})
			return
		}

		if nasabah.StatusNasabah != models.Aktif {
			var msg string = "Akun nasabah Anda belum aktif"
			if nasabah.StatusNasabah == models.Nonaktif {
				msg = "Akun nasabah Anda dinonaktifkan. Silakan hubungi admin bank sampah."
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": msg})
			return
		}

		finalRole = models.RoleNasabah
		identityID = nasabah.NasabahID
		bankID = &nasabah.BankID

	case models.RoleUserAdmin:
		// --- ADMIN/STAFF BLOCK ---
		var admin models.Admin
		if err := ac.DB.Where("user_id = ?", user.UserID).First(&admin).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Akun Anda tidak terdaftar sebagai pengurus atau admin"})
			return
		}

		if admin.StatusAdmin != models.Aktif {
			var msg string = "Akun pengurus Anda belum aktif"
			if admin.StatusAdmin == models.Nonaktif {
				msg = "Akun pengurus Anda dinonaktifkan"
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": msg})
			return
		}

		// Validasi Platform Dashboard Web
		if req.Platform == "web" {
			if admin.Role != models.SuperAdmin && admin.Role != models.AdminBSI && admin.Role != models.AdminBSU && admin.Role != models.AdminBSM {
				c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak. Platform ini hanya untuk Administrator"})
				return
			}
		}

		// Validasi Platform Mobile App (Hanya untuk Petugas)
		if req.Platform == "mobile" {
			if admin.Role != models.PetugasBSI && admin.Role != models.PetugasBSM && admin.Role != models.PetugasBSU {
				c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak. Platform mobile hanya untuk Petugas Lapangan"})
				return
			}
		}

		finalRole = admin.Role
		identityID = admin.AdminID
		bankID = admin.BankID

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tipe role tidak valid"})
		return
	}

	// 4. Generate Tokens
	accessToken, err := utils.GenerateJWT(user.UserID, finalRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat access token"})
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.UserID, finalRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat refresh token"})
		return
	}

	// 5. Handling Response berdasarkan Platform
	if req.Platform == "web" {
		utils.SetTokenCookies(c, accessToken, refreshToken)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login berhasil",
		"data": gin.H{
			"user_id":       user.UserID,
			"nama":          user.Nama,
			"email":         user.Email,
			"role":          finalRole,
			"bank_id":       bankID,
			"identity_id":   identityID,
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		},
	})
}

// RefreshToken menerima refresh_token dari cookie, validasi, lalu terbitkan access_token baru.
func (ac *AuthController) RefreshToken(c *gin.Context) {
	refreshStr, err := c.Cookie(utils.RefreshTokenCookieName)
	if err != nil || refreshStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token tidak ditemukan"})
		return
	}

	claims, err := utils.ParseJWT(refreshStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token tidak valid atau sudah kadaluarsa"})
		return
	}

	// Terbitkan access token baru
	newAccessToken, err := utils.GenerateJWT(claims.UserID, claims.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token baru"})
		return
	}

	// Terbitkan juga refresh token baru (rolling refresh)
	newRefreshToken, err := utils.GenerateRefreshToken(claims.UserID, claims.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat refresh token baru"})
		return
	}

	utils.SetTokenCookies(c, newAccessToken, newRefreshToken)

	c.JSON(http.StatusOK, gin.H{"message": "Token berhasil diperbarui"})
}

// Logout menghapus cookie JWT dari browser.
func (ac *AuthController) Logout(c *gin.Context) {
	utils.ClearTokenCookies(c)
	c.JSON(http.StatusOK, gin.H{"message": "Logout berhasil"})
}

// Me mengembalikan info user yang sedang login berdasarkan JWT di cookie.
func (ac *AuthController) Me(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return
	}

	var user models.User
	if err := ac.DB.Where("user_id = ?", claims.UserID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	var admin models.Admin
	if err := ac.DB.Where("user_id = ?", claims.UserID).First(&admin).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"user_id":  user.UserID,
			"nama":     user.Nama,
			"email":    user.Email,
			"photo":    user.PhotoURL,
			"role":     admin.Role,
			"bank_id":  admin.BankID,
			"admin_id": admin.AdminID,
			"status":   admin.StatusAdmin,
		},
	})
}

func (ac *AuthController) AktivasiAkun(c *gin.Context) {
	var req struct {
		UserID   string `json:"user_id" binding:"required"`
		OTP      string `json:"otp" binding:"required"`
		Password string `json:"password" binding:"required,min=8"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Cari record aktivasi akun yang terbaru, belum kadaluarsa, dan belum digunakan
	var aktivasi models.AktivasiAkun
	if err := ac.DB.Where("user_id = ? AND is_used = ? AND expired_at > ?", req.UserID, false, time.Now()).
		Order("created_at desc").
		First(&aktivasi).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kode aktivasi tidak valid, kadaluarsa, atau sudah digunakan"})
		return
	}

	// 2. Verifikasi OTP
	decodeOTP := utils.VerifyOTP(req.OTP, aktivasi.Token)
	if !decodeOTP {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OTP tidak valid atau salah"})
		return
	}

	// 3. Pastikan user ada
	var user models.User
	if err := ac.DB.Where("user_id = ?", req.UserID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	// 4. Periksa record berdasarkan role aktivasi (Nasabah atau Admin)
	var isNasabah bool
	var isRoleAdmin bool
	var nasabah models.Nasabah
	var admin models.Admin

	switch aktivasi.AsRole {
	case models.RoleUserNasabah:
		isNasabah = true
		if err := ac.DB.Where("user_id = ?", req.UserID).First(&nasabah).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Identitas Nasabah tidak ditemukan, aktivasi gagal"})
			return
		}
		if nasabah.StatusNasabah != models.Pending {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Akun Nasabah sudah aktif atau tidak berlaku"})
			return
		}
	case models.RoleUserAdmin:
		isRoleAdmin = true
		if err := ac.DB.Where("user_id = ?", req.UserID).First(&admin).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Identitas Admin/Petugas tidak ditemukan, aktivasi gagal"})
			return
		}
		if admin.StatusAdmin != models.Pending {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Akun Admin/Petugas sudah aktif atau tidak berlaku"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tipe role aktivasi tidak diketahui"})
		return
	}

	// 5. Generate Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password"})
		return
	}

	tx := ac.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 6. Update password user
	if err := tx.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate password user"})
		return
	}

	// 7. Update status Nasabah atau Admin
	if isNasabah {
		if err := tx.Model(&nasabah).Update("status_nasabah", models.Aktif).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate status nasabah"})
			return
		}
	} else if isRoleAdmin {
		if err := tx.Model(&admin).Update("status_admin", models.Aktif).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate status admin"})
			return
		}
	}

	// 8. Tandai aktivasi sebagai telah digunakan
	if err := tx.Model(&aktivasi).Where("aktivasi_id = ?", aktivasi.AktivasiID).Updates(map[string]interface{}{
		"is_used": true,
		"used_at": time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate status aktivasi"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan ke database"})
		return
	}

	var successMessage string
	if isNasabah {
		successMessage = "Aktivasi nasabah berhasil. Anda sekarang dapat login menggunakan password."
	} else {
		successMessage = "Aktivasi staff berhasil. Anda sekarang dapat login menggunakan password."
	}

	c.JSON(http.StatusOK, gin.H{"message": successMessage})
}



func (ac *AuthController) GenerateReactivateAkun(c *gin.Context) {
	var req struct {
		UserID  string `json:"user_id" binding:"required"`
		Role models.RoleUserEnum `json:"role" binding:"required"`
		AdminID string `json:"admin_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := ac.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Pastikan user ada
	var user models.User
	if err := tx.Where("user_id = ?", req.UserID).First(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	// 2. Cari Identitas & Update Status berdasarkan Role dari Request
	var bankID string
	var successRoleName string

	var nasabah models.Nasabah
	var admin models.Admin

	switch req.Role {
	case models.RoleUserNasabah:
		// Cari di Nasabah
		if err := tx.Where("user_id = ?", req.UserID).First(&nasabah).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{"error": "Identitas nasabah tidak ditemukan"})
			return
		}
		bankID = nasabah.BankID
		successRoleName = "nasabah"
		if err := tx.Model(&nasabah).Update("status_nasabah", models.Pending).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate status nasabah"})
			return
		}
	case models.RoleUserAdmin:
		// Cari di Admin
		if err := tx.Where("user_id = ?", req.UserID).First(&admin).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{"error": "Identitas staff tidak ditemukan"})
			return
		}
		if admin.BankID != nil {
			bankID = *admin.BankID
		}
		successRoleName = "staff"
		if err := tx.Model(&admin).Update("status_admin", models.Pending).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate status admin"})
			return
		}
	default:
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role tidak valid"})
		return
	}

	// 3. Ambil Nama Bank Sampah
	var bankSampah models.BankSampah
	var namaBank string = "Enviroo System"
	if bankID != "" {
		if err := tx.Where("bank_id = ?", bankID).First(&bankSampah).Error; err == nil {
			namaBank = bankSampah.NamaBank
		}
	}

	// 4. Persiapkan OTP
	OTP, err := utils.GenerateOTP()
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal generate OTP"})
		return
	}

	hashedOTP, err := utils.HashOTP(OTP)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal hash OTP"})
		return
	}

	// 5. Create/Update record aktivasi
	var aktivasi models.AktivasiAkun
	errAktivasi := tx.Where("user_id = ? AND is_used = ? AND expired_at > ?", req.UserID, false, time.Now()).
		Order("created_at desc").
		Limit(1).Find(&aktivasi).Error

	if errAktivasi == nil && aktivasi.AktivasiID != "" {
		// Update yang sudah ada
		if err := tx.Model(&aktivasi).Where("aktivasi_id = ?", aktivasi.AktivasiID).Updates(map[string]interface{}{
			"token":        hashedOTP,
			"expired_at":   time.Now().Add(24 * time.Hour),
			"generated_by": req.AdminID,
			"as_role":      req.Role,
		}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate record aktivasi"})
			return
		}
	} else {
		// Buat baru
		now := time.Now()
		aktivasiID := utils.GenerateAktivasiID(req.UserID)

		aktivasi = models.AktivasiAkun{
			AktivasiID:  aktivasiID,
			UserID:      req.UserID,
			AsRole:      req.Role,
			IsUsed:      false,
			Token:       hashedOTP,
			ExpiredAt:   now.Add(24 * time.Hour),
			GeneratedBy: req.AdminID,
		}

		if err := tx.Create(&aktivasi).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat record aktivasi baru"})
			return
		}
	}

	// 6. Kirim Email (Asynchronous)
	emailBody := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; background-color: #f4fdf4; padding: 30px; border-radius: 10px;">
			<h2 style="color: #4ea771; margin-top: 0;">Halo, %s!</h2>
			<p style="font-size: 14px; color: #333; line-height: 1.5;">
				Untuk mengaktifkan kembali akun %s Anda di Bank Sampah <b>%s</b>, gunakan kode OTP berikut:
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
	`, user.Nama, successRoleName, namaBank, OTP, aktivasi.ExpiredAt.Format("02 Jan 2006 15:04 WIB"))

	go func() {
		_ = ac.Mailer.SendEmail(utils.EmailParams{
			To:      user.Email,
			Subject: "Aktivasi Kembali Akun Enviroo",
			Body:    emailBody,
		})
	}()

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan ke database"})
		return
	}

	var response = map[string]interface{}{
		"aktivasi_id": aktivasi.AktivasiID,
		"otp":         OTP,
		"expired_at":  aktivasi.ExpiredAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Permintaan aktivasi ulang berhasil diproses. Informasi telah dikirimkan ke email user terkait.",
		"data":    response,
	})
}

func (ac *AuthController) ReactivateAkun(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		OTP    string `json:"otp" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := ac.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Validasi User
	var user models.User
	if err := tx.Where("user_id = ?", req.UserID).First(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	// 2. Validasi OTP & Aktivasi record
	var aktivasi models.AktivasiAkun
	if err := tx.Where("user_id = ? AND is_used = ? AND expired_at > ?", req.UserID, false, time.Now()).
		Order("created_at desc").
		First(&aktivasi).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Kode aktivasi tidak valid, kadaluarsa, atau sudah digunakan"})
		return
	}

	if !utils.VerifyOTP(req.OTP, aktivasi.Token) {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "OTP tidak valid atau salah"})
		return
	}

	// 3. Update Status berdasarkan Role Aktivasi
	switch aktivasi.AsRole {
	case models.RoleUserNasabah:
		var nasabah models.Nasabah
		if err := tx.Where("user_id = ?", req.UserID).First(&nasabah).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{"error": "Data nasabah tidak ditemukan"})
			return
		}

		if nasabah.StatusNasabah != models.Pending {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nasabah sudah aktif atau tidak dalam status pending"})
			return
		}

		if err := tx.Model(&nasabah).Update("status_nasabah", models.Aktif).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengaktifkan nasabah"})
			return
		}
	case models.RoleUserAdmin:
		var admin models.Admin
		if err := tx.Where("user_id = ?", req.UserID).First(&admin).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{"error": "Data admin tidak ditemukan"})
			return
		}

		if admin.StatusAdmin != models.Pending {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Admin sudah aktif atau tidak dalam status pending"})
			return
		}

		if err := tx.Model(&admin).Update("status_admin", models.Aktif).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengaktifkan admin"})
			return
		}
	default:
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tipe aktivasi tidak didukung"})
		return
	}

	// 4. Tandai aktivasi sudah digunakan
	if err := tx.Model(&aktivasi).Where("aktivasi_id = ?", aktivasi.AktivasiID).Updates(map[string]interface{}{
		"is_used": true,
		"used_at": time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan proses aktivasi"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses transaksi aktivasi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Akun berhasil diaktifkan kembali. Anda sekarang dapat login menggunakan password lama Anda."})
}

func (ac *AuthController) DeactivateAkun(c *gin.Context) {
	var req struct {
		UserID string              `json:"user_id" binding:"required"`
		Role   models.RoleUserEnum `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var result *gorm.DB
	switch req.Role {
	case models.RoleUserNasabah:
		result = ac.DB.Model(&models.Nasabah{}).Where("user_id = ?", req.UserID).Update("status_nasabah", models.Nonaktif)
	case models.RoleUserAdmin:
		result = ac.DB.Model(&models.Admin{}).Where("user_id = ?", req.UserID).Update("status_admin", models.Nonaktif)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role tidak valid"})
		return
	}

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menonaktifkan akun: " + result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data akun tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Akun berhasil dinonaktifkan."})
}

func (ac *AuthController) CekUserMobile(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Cari User berdasarkan Email
	var user models.User
	if err := ac.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	// 2. Verifikasi Password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Password yang Anda masukkan salah"})
		return
	}

	// 3. Deteksi Role yang Aktif & Valid untuk Mobile
	var validRoles []string

	// Check Nasabah
	var nasabah models.Nasabah
	if err := ac.DB.Where("user_id = ? AND status_nasabah = ?", user.UserID, models.Aktif).First(&nasabah).Error; err == nil {
		validRoles = append(validRoles, string(models.RoleUserNasabah))
	}

	// Check Admin (Hanya Petugas yang boleh di Mobile)
	var admin models.Admin
	if err := ac.DB.Where("user_id = ? AND status_admin = ?", user.UserID, models.Aktif).First(&admin).Error; err == nil {
		// Filter: hanya role petugas yang diizinkan di mobile
		if admin.Role == models.PetugasBSI || admin.Role == models.PetugasBSM || admin.Role == models.PetugasBSU {
			validRoles = append(validRoles, string(models.RoleUserAdmin))
		}
	}

	// 4. Response Berdasarkan Role yang Ditemukan
	if len(validRoles) == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akun Anda tidak aktif atau tidak memiliki akses untuk aplikasi mobile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User ditemukan",
		"data": gin.H{
			"user_id":        user.UserID,
			"multiple_roles": len(validRoles) > 1,
			"roles":          validRoles,
		},
	})
}
