package controllers

import (
	"encoding/json"
	"enviroo-be/internal/models"
	"enviroo-be/pkg/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminController struct {
	DB     *gorm.DB
	Mailer *utils.Mailer
}

func NewAdminController(db *gorm.DB, mailer *utils.Mailer) *AdminController {
	return &AdminController{
		DB:     db,
		Mailer: mailer,
	}
}

func (ac *AdminController) GetAdminBankSampah(c *gin.Context) {
	bankID := c.Param("bank_id")

	type AdminBankSampahResponse struct {
		AdminID     string `json:"admin_id" gorm:"column:admin_id"`
		UserID      string `json:"user_id" gorm:"column:user_id"`
		Nama        string `json:"nama" gorm:"column:nama"`
		Foto        string `json:"foto" gorm:"column:photo_url"`
		Email       string `json:"email" gorm:"column:email"`
		Role        string `json:"role" gorm:"column:role"`
		StatusAdmin string `json:"status_admin" gorm:"column:status_admin"`
	}

	var results []AdminBankSampahResponse

	query := ac.DB.Table("admin").
		Select("admin.admin_id, admin.user_id, u.nama, u.photo_url, u.email, admin.role, admin.status_admin").
		Joins("LEFT JOIN users u ON admin.user_id = u.user_id").
		Where("admin.bank_id = ?", bankID)

	if err := query.Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get admin: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Admin retrieved successfully",
		"data":    results,
	})
}

func (ac *AdminController) AddAdminBankSampah(c *gin.Context) {
	type AddAdminBankSampahRequest struct {
		UserID string `json:"user_id" binding:"required"`
		BankID string `json:"bank_id" binding:"required"`
		Role   string `json:"role" binding:"required"`
		AdminID string `json:"admin_id" binding:"required"`
	}

	var req AddAdminBankSampahRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validasi role yang diperbolehkan
	validRoles := map[models.RoleAdmin]bool{
		models.AdminBSI:   true,
		models.AdminBSM:   true,
		models.AdminBSU:   true,
		models.PetugasBSI: true,
		models.PetugasBSM: true,
		models.PetugasBSU: true,
	}

	role := models.RoleAdmin(req.Role)
	if !validRoles[role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role tidak valid"})
		return
	}

	tx := ac.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Cek apakah user sudah menjadi admin di bank ini
	var existingAdmin models.Admin
	if err := tx.Where("user_id = ? AND bank_id = ?", req.UserID, req.BankID).First(&existingAdmin).Error; err == nil {
		tx.Rollback()
		c.JSON(http.StatusConflict, gin.H{"error": "User sudah terdaftar sebagai admin di bank sampah ini"})
		return
	}

	// Cek apakah user sudah menjadi nasabah di bank ini
	var countNasabah int64
	if err := tx.Model(&models.Nasabah{}).Where("user_id = ? AND bank_id = ?", req.UserID, req.BankID).Count(&countNasabah).Error; err == nil && countNasabah > 0 {
		tx.Rollback()
		c.JSON(http.StatusConflict, gin.H{"error": "User tidak boleh terdaftar sebagai admin di bank sampah tempat ia terdaftar sebagai nasabah"})
		return
	}

	newAdmin := models.Admin{
		AdminID:     utils.GenerateAdminID(),
		UserID:      req.UserID,
		BankID:      &req.BankID,
		Role:        role,
		StatusAdmin: models.Pending,
	}

	if err := tx.Create(&newAdmin).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create admin: " + err.Error()})
		return
	}

	// LOGIK AKTIVASI AKUN ADMIN
	var existingUser models.User
	if err := tx.Where("user_id = ?", req.UserID).First(&existingUser).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user data for activation"})
		return
	}

	var bank models.BankSampah
	if err := tx.Where("bank_id = ?", req.BankID).First(&bank).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get bank data for activation"})
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

	var roleTitle string
	switch role {
	case models.AdminBSI, models.AdminBSM, models.AdminBSU:
		roleTitle = "Administrator"
	case models.PetugasBSI, models.PetugasBSM, models.PetugasBSU:
		roleTitle = "Petugas"
	default:
		roleTitle = string(role)
	}

	emailBody := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; background-color: #f4fdf4; padding: 30px; border-radius: 10px;">
			<h2 style="color: #4ea771; margin-top: 0;">Halo, %s!</h2>
			<p style="font-size: 14px; color: #333; line-height: 1.5;">
				Anda telah ditunjuk sebagai <b>%s</b> di Bank Sampah <b>%s</b>.
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
	`, existingUser.Nama, roleTitle, bank.NamaBank, otp, newAktivasiAkun.ExpiredAt.Format("02 Jan 2006 15:04 WIB"))

	sendErr := ac.Mailer.SendEmail(utils.EmailParams{
		To:      existingUser.Email,
		Subject: fmt.Sprintf("Aktivasi Akun %s Enviroo", roleTitle),
		Body:    emailBody,
	})

	if sendErr != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email: " + sendErr.Error()})
		return
	}

	// LOGIK HISTORY AKUN BANK
	actorName := req.AdminID
	ac.DB.Table("admin").
		Joins("JOIN users ON admin.user_id = users.user_id").
		Where("admin.admin_id = ?", req.AdminID).
		Select("users.nama").
		Scan(&actorName)

	newAdminData, _ := json.Marshal(newAdmin)
	newBankAkunHistory := models.HistoryAkunBank{
		BankID:    req.BankID,
		Action:    "CREATE",
		NewValue:  newAdminData,
		Informasi: fmt.Sprintf("Staff %s (%s) ditambahkan ke %s", existingUser.Nama, roleTitle, bank.NamaBank),
		Keterangan: fmt.Sprintf("%s menambahkan %s (%s) ke %s",
			actorName, existingUser.Nama, roleTitle, bank.NamaBank),
		CreatedBy: req.AdminID,
	}

	if err := tx.Create(&newBankAkunHistory).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create audit history: " + err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Admin created successfully and activation sent",
		"data":    newAdmin,
	})
}

func (ac *AdminController) DeleteStaffBankSampah(c *gin.Context) {
	adminID := c.Param("admin_id")

	type Request struct {
		DeletedBy string `json:"deleted_by" binding:"required"` // AdminID of the person deleting
	}

	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Fetch admin with associations (Preload might be flaky with custom PKs)
	var admin models.Admin
	if err := ac.DB.Preload("User").Preload("Bank").Where("admin_id = ?", adminID).First(&admin).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Staff not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch staff data: " + err.Error()})
		}
		return
	}

	// 2. Resolve Names with explicit fallbacks to ensure audit log is readable
	staffName := admin.User.Nama
	if staffName == "" {
		// Manual fetch if preload failed
		var user models.User
		if err := ac.DB.Select("nama").Where("user_id = ?", admin.UserID).First(&user).Error; err == nil && user.Nama != "" {
			staffName = user.Nama
		} else {
			staffName = admin.UserID // Use ID as fallback name
		}
	}

	bankName := "Unknown Bank"
	if admin.Bank.NamaBank != "" {
		bankName = admin.Bank.NamaBank
	} else if admin.BankID != nil {
		var bank models.BankSampah
		if err := ac.DB.Select("nama_bank").Where("bank_id = ?", *admin.BankID).First(&bank).Error; err == nil && bank.NamaBank != "" {
			bankName = bank.NamaBank
		}
	}

	// 3. Resolve Actor Name with robust Join fetch
	actorName := ""
	ac.DB.Table("admin").
		Joins("JOIN users ON admin.user_id = users.user_id").
		Where("admin.admin_id = ?", req.DeletedBy).
		Select("users.nama").
		Scan(&actorName)
	if actorName == "" {
		actorName = req.DeletedBy // Fallback to actor ID
	}

	// 4. Serialize admin's current state for audit trail
	type AdminSnapshot struct {
		AdminID     string            `json:"admin_id"`
		BankID      *string           `json:"bank_id"`
		UserID      string            `json:"user_id"`
		Nama        string            `json:"nama"`
		Role        models.RoleAdmin  `json:"role"`
		StatusAdmin models.StatusAkun `json:"status_admin"`
	}
	snapshot := AdminSnapshot{
		AdminID:     admin.AdminID,
		BankID:      admin.BankID,
		UserID:      admin.UserID,
		Nama:        staffName,
		Role:        admin.Role,
		StatusAdmin: admin.StatusAdmin,
	}
	adminData, _ := json.Marshal(snapshot)

	// 5. Begin transaction
	tx := ac.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 6. Cleanup Activation tokens
	if admin.StatusAdmin == models.Pending {
		tx.Where("user_id = ? AND as_role = ? AND is_used = false", admin.UserID, models.RoleUserAdmin).
			Delete(&models.AktivasiAkun{})
	}

	// 7. Create audit history
	// Note: We avoid setting AdminID property in the struct to prevent FK constraint issues
	// if the DB doesn't support SET NULL correctly on the existing record.
	// We store the ID in the OldValue and Information strings instead.
	newBankAkunHistory := models.HistoryAkunBank{
		BankID:     *admin.BankID,
		Action:     "DELETE",
		OldValue:   adminData,
		Informasi:  fmt.Sprintf("Staff %s (%s) dihapus dari %s", staffName, admin.Role, bankName),
		Keterangan: fmt.Sprintf("%s menghapus %s (%s) dari %s", actorName, staffName, admin.Role, bankName),
		CreatedBy:  req.DeletedBy,
	}

	if err := tx.Create(&newBankAkunHistory).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create audit history: " + err.Error()})
		return
	}

	// 8. Delete the admin record
	if err := tx.Where("admin_id = ?", adminID).Delete(&models.Admin{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete staff: " + err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Staff %s berhasil dihapus dari %s", staffName, bankName),
	})
}


