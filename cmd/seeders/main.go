package main

import (
	"enviroo-be/internal/config"
	"enviroo-be/internal/database"
	"enviroo-be/internal/models"
	"enviroo-be/pkg/utils"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	superadminUserID   = "1234567890"
	superadminNama     = "Annin Carista"
	superadminEmail    = "annincarista@gmail.com"
	superadminWhatsapp = "081299001122"
)

func main() {
	cfg := config.LoadConfig()

	db, err := database.ConnectDB(cfg)
	if err != nil {
		log.Fatalf("Gagal konek ke database: %v", err)
	}

	mailer := utils.NewMailer(cfg)

	log.Println("Memulai seeder superadmin...")

	if err := seedSuperadmin(db, mailer); err != nil {
		log.Fatalf("Seeder gagal: %v", err)
	}
}

func seedSuperadmin(db *gorm.DB, mailer *utils.Mailer) error {
	var userCount int64
	db.Model(&models.User{}).Where("user_id = ?", superadminUserID).Count(&userCount)
	if userCount > 0 {
		log.Printf("User dengan ID %s sudah ada, seeder dilewati.", superadminUserID)
		return nil
	}

	var emailCount int64
	db.Model(&models.User{}).Where("email = ?", superadminEmail).Count(&emailCount)
	if emailCount > 0 {
		return fmt.Errorf("email %s sudah digunakan oleh user lain", superadminEmail)
	}

	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("gagal memulai transaksi: %v", tx.Error)
	}

	user := models.User{
		UserID:     superadminUserID,
		Nama:       utils.ToTitleCase(superadminNama),
		Email:      superadminEmail,
		NoWhatsapp: superadminWhatsapp,
	}
	if err := tx.Omit(clause.Associations).Create(&user).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("gagal membuat user: %v", err)
	}

	adminID, err := utils.GenerateAdminID(tx, models.SuperAdmin, "")
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("gagal generate admin ID: %v", err)
	}

	admin := models.Admin{
		AdminID:     adminID,
		UserID:      superadminUserID,
		BankID:      nil,
		Role:        models.SuperAdmin,
		StatusAdmin: models.Pending,
	}
	if err := tx.Omit(clause.Associations).Create(&admin).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("gagal membuat admin: %v", err)
	}

	otp, err := utils.GenerateOTP()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("gagal generate OTP: %v", err)
	}
	hashedOTP, err := utils.HashOTP(otp)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("gagal hash OTP: %v", err)
	}

	now := time.Now()
	aktivasiID := utils.GenerateAktivasiID(superadminUserID)
	aktivasi := models.AktivasiAkun{
		AktivasiID:  aktivasiID,
		UserID:      superadminUserID,
		AsRole:      models.RoleUserAdmin,
		Token:       hashedOTP,
		ExpiredAt:   now.Add(24 * time.Hour),
		GeneratedBy: "system",
		Tujuan:      models.TujuanAktivasi,
	}
	if err := tx.Create(&aktivasi).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("gagal membuat aktivasi akun: %v", err)
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
	`, user.Nama, otp, aktivasi.ExpiredAt.Format("02 Jan 2006 15:04 WIB"))

	if err := mailer.SendEmail(utils.EmailParams{
		To:      superadminEmail,
		Subject: "Aktivasi Akun Superadmin Enviroo",
		Body:    emailBody,
	}); err != nil {
		tx.Rollback()
		return fmt.Errorf("gagal mengirim email aktivasi: %v", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("gagal commit transaksi: %v", err)
	}

	log.Println("========================================")
	log.Printf("Superadmin berhasil dibuat!")
	log.Printf("User ID  : %s", superadminUserID)
	log.Printf("Admin ID : %s", adminID)
	log.Printf("Email    : %s", superadminEmail)
	log.Printf("OTP aktivasi dikirim ke email di atas.")
	log.Println("========================================")

	return nil
}
