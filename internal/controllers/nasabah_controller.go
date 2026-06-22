package controllers

import (
	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"
	"errors"
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
	UserID      string  `json:"user_id" binding:"required"`
	Nama        string  `json:"nama" binding:"required"`
	Email       string  `json:"email" binding:"required,email"`
	NoWhatsapp  string  `json:"no_whatsapp" binding:"required"`
	BankID      string  `json:"bank_id" binding:"required"`
	AdminID     string  `json:"admin_id" binding:"required"`
	NoRekening  string  `json:"no_rekening"`
	SaldoRupiah float64 `json:"saldo_rupiah"`
	SaldoPoin   float64 `json:"saldo_poin"`
}

func (nc *NasabahController) AddNewNasabah(c *gin.Context) {
	var req AddNasabahRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userCount int64
	if err := nc.DB.Model(&models.User{}).Where("user_id = ?", req.UserID).Count(&userCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi data user"})
		return
	}
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

	nasabahID, err := utils.GenerateNasabahID(tx, req.BankID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate nasabah ID: " + err.Error()})
		return
	}

	nomorRekening := req.NoRekening
	if nomorRekening == "" {
		nomorRekening = nasabahID
	}

	newNasabah := models.Nasabah{
		NasabahID:     nasabahID,
		UserID:        req.UserID,
		BankID:        req.BankID,
		NomorRekening: nomorRekening,
		StatusNasabah: models.Pending,
	}

	if err := tx.Create(&newNasabah).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create nasabah: " + err.Error()})
		return
	}

	// Ambil reward_id secara dinamis
	var rewardUang, rewardSembako models.Reward
	if err := tx.Where("nama_reward = ?", models.RewardEnumUang).First(&rewardUang).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Reward Uang tidak ditemukan: " + err.Error()})
		return
	}
	if err := tx.Where("nama_reward = ?", models.RewardEnumSembako).First(&rewardSembako).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Reward Sembako tidak ditemukan: " + err.Error()})
		return
	}

	// Insert saldo awal Rupiah
	saldoRupiah := models.SaldoRekening{
		RekeningID:         utils.GenerateRekeningID(rewardUang.RewardID, newNasabah.NasabahID),
		NasabahID:          &newNasabah.NasabahID,
		RewardID:           &rewardUang.RewardID,
		Entitas:            models.EntitasNasabah,
		NominalSaldo:       req.SaldoRupiah,
		SatuanNominalSaldo: models.SatuanRewardEnumRp,
		LastUpdatedBy:      &req.AdminID,
	}
	if err := tx.Create(&saldoRupiah).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create saldo rupiah: " + err.Error()})
		return
	}

	riwayatRupiah := models.RiwayatArusSaldo{
		RiwayatSaldoID: utils.GenerateID("RWY"),
		RekeningID:     &saldoRupiah.RekeningID,
		NominalSebelum: 0,
		NominalSesudah: req.SaldoRupiah,
		CreatedBy:      &req.AdminID,
		Keterangan:     "Saldo awal nasabah",
	}
	if err := tx.Create(&riwayatRupiah).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create riwayat saldo rupiah: " + err.Error()})
		return
	}

	// Insert saldo awal Poin (sembako)
	saldoPoin := models.SaldoRekening{
		RekeningID:         utils.GenerateRekeningID(rewardSembako.RewardID, newNasabah.NasabahID),
		NasabahID:          &newNasabah.NasabahID,
		RewardID:           &rewardSembako.RewardID,
		Entitas:            models.EntitasNasabah,
		NominalSaldo:       req.SaldoPoin,
		SatuanNominalSaldo: models.SatuanRewardEnumPoin,
		LastUpdatedBy:      &req.AdminID,
	}
	if err := tx.Create(&saldoPoin).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create saldo poin: " + err.Error()})
		return
	}

	riwayatPoin := models.RiwayatArusSaldo{
		RiwayatSaldoID: utils.GenerateID("RWY"),
		RekeningID:     &saldoPoin.RekeningID,
		NominalSebelum: 0,
		NominalSesudah: req.SaldoPoin,
		CreatedBy:      &req.AdminID,
		Keterangan:     "Saldo awal nasabah",
	}
	if err := tx.Create(&riwayatPoin).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create riwayat saldo poin: " + err.Error()})
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
		Tujuan:      models.TujuanAktivasi,
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
				Terima kasih telah mendaftar sebagai Nasabah di <b>%s</b>.
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

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	go nc.Mailer.SendEmail(utils.EmailParams{
		To:      req.Email,
		Subject: "Aktivasi Akun Nasabah Enviroo",
		Body:    emailBody,
	})

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
	UserID      string  `json:"user_id" binding:"required"`
	BankID      string  `json:"bank_id" binding:"required"`
	AdminID     string  `json:"admin_id" binding:"required"`
	NoRekening  string  `json:"no_rekening"`
	SaldoRupiah float64 `json:"saldo_rupiah"`
	SaldoPoin   float64 `json:"saldo_poin"`
}

func (nc *NasabahController) AddNewNasabahOldUser(c *gin.Context) {
	var req AddNasabahOldUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existingUser models.User
	if err := nc.DB.Where("user_id = ?", req.UserID).First(&existingUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User dengan ID ini tidak ditemukan"})
		return
	}

	var activeNasabahCount int64
	if err := nc.DB.Model(&models.Nasabah{}).Where("user_id = ? AND status_nasabah = ?", req.UserID, models.Aktif).Count(&activeNasabahCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi data nasabah"})
		return
	}
	if activeNasabahCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "User sudah memiliki akun nasabah aktif. Nonaktifkan akun nasabah yang ada terlebih dahulu."})
		return
	}

	var pendingNasabahElsewhere int64
	if err := nc.DB.Model(&models.Nasabah{}).Where("user_id = ? AND status_nasabah = ? AND bank_id != ?", req.UserID, models.Pending, req.BankID).Count(&pendingNasabahElsewhere).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi data nasabah"})
		return
	}
	if pendingNasabahElsewhere > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "User masih dalam proses pendaftaran di bank sampah lain. Selesaikan atau batalkan pendaftaran tersebut terlebih dahulu."})
		return
	}

	var existingNasabah models.Nasabah
	if err := nc.DB.Where("user_id = ? AND bank_id = ?", req.UserID, req.BankID).First(&existingNasabah).Error; err == nil {
		if existingNasabah.StatusNasabah == models.Nonaktif {
			c.JSON(http.StatusConflict, gin.H{"error": "User pernah terdaftar di bank sampah ini. Gunakan fitur aktivasi untuk mengaktifkan kembali."})
		} else {
			c.JSON(http.StatusConflict, gin.H{"error": "User sudah terdaftar sebagai nasabah di bank sampah ini."})
		}
		return
	}

	// Ambil data BankSampah tempat Nasabah mendaftar
	var bank models.BankSampah
	if err := nc.DB.Where("bank_id = ?", req.BankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	tx := nc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database"})
		return
	}

	nasabahID, err := utils.GenerateNasabahID(tx, req.BankID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate nasabah ID: " + err.Error()})
		return
	}

	nomorRekening2 := req.NoRekening
	if nomorRekening2 == "" {
		nomorRekening2 = nasabahID
	}

	newNasabah := models.Nasabah{
		NasabahID:     nasabahID,
		UserID:        req.UserID,
		BankID:        req.BankID,
		NomorRekening: nomorRekening2,
		StatusNasabah: models.Pending,
	}

	if err := tx.Create(&newNasabah).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create nasabah: " + err.Error()})
		return
	}

	// Ambil reward_id secara dinamis
	var rewardUang2, rewardSembako2 models.Reward
	if err := tx.Where("nama_reward = ?", models.RewardEnumUang).First(&rewardUang2).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Reward Uang tidak ditemukan: " + err.Error()})
		return
	}
	if err := tx.Where("nama_reward = ?", models.RewardEnumSembako).First(&rewardSembako2).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Reward Sembako tidak ditemukan: " + err.Error()})
		return
	}

	// Insert saldo awal Rupiah
	saldoRupiah := models.SaldoRekening{
		RekeningID:         utils.GenerateRekeningID(rewardUang2.RewardID, newNasabah.NasabahID),
		NasabahID:          &newNasabah.NasabahID,
		RewardID:           &rewardUang2.RewardID,
		Entitas:            models.EntitasNasabah,
		NominalSaldo:       req.SaldoRupiah,
		SatuanNominalSaldo: models.SatuanRewardEnumRp,
		LastUpdatedBy:      &req.AdminID,
	}
	if err := tx.Create(&saldoRupiah).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create saldo rupiah: " + err.Error()})
		return
	}

	riwayatRupiah := models.RiwayatArusSaldo{
		RiwayatSaldoID: utils.GenerateID("RWY"),
		RekeningID:     &saldoRupiah.RekeningID,
		NominalSebelum: 0,
		NominalSesudah: req.SaldoRupiah,
		CreatedBy:      &req.AdminID,
		Keterangan:     "Saldo awal nasabah",
	}
	if err := tx.Create(&riwayatRupiah).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create riwayat saldo rupiah: " + err.Error()})
		return
	}

	// Insert saldo awal Poin (sembako)
	saldoPoin := models.SaldoRekening{
		RekeningID:         utils.GenerateRekeningID(rewardSembako2.RewardID, newNasabah.NasabahID),
		NasabahID:          &newNasabah.NasabahID,
		RewardID:           &rewardSembako2.RewardID,
		Entitas:            models.EntitasNasabah,
		NominalSaldo:       req.SaldoPoin,
		SatuanNominalSaldo: models.SatuanRewardEnumPoin,
		LastUpdatedBy:      &req.AdminID,
	}
	if err := tx.Create(&saldoPoin).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create saldo poin: " + err.Error()})
		return
	}

	riwayatPoin := models.RiwayatArusSaldo{
		RiwayatSaldoID: utils.GenerateID("RWY"),
		RekeningID:     &saldoPoin.RekeningID,
		NominalSebelum: 0,
		NominalSesudah: req.SaldoPoin,
		CreatedBy:      &req.AdminID,
		Keterangan:     "Saldo awal nasabah",
	}
	if err := tx.Create(&riwayatPoin).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create riwayat saldo poin: " + err.Error()})
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

	if err := tx.Where("user_id = ? AND as_role = ? AND tujuan = ? AND is_used = ?", req.UserID, models.RoleUserNasabah, models.TujuanAktivasi, false).
		Delete(&models.AktivasiAkun{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clean up old activation: " + err.Error()})
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
		Tujuan:      models.TujuanAktivasi,
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
				Terima kasih telah mendaftar sebagai Nasabah di <b>%s</b>.
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

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	go nc.Mailer.SendEmail(utils.EmailParams{
		To:      existingUser.Email,
		Subject: "Aktivasi Akun Nasabah Enviroo",
		Body:    emailBody,
	})

	c.JSON(http.StatusCreated, gin.H{
		"message": "Nasabah created successfully. Activation email sent.",
		"data":    newNasabah,
	})
}

func (nc *NasabahController) DeleteNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	var nasabah models.Nasabah
	if err := nc.DB.Where("nasabah_id = ?", nasabahID).First(&nasabah).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nasabah tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data nasabah"})
		}
		return
	}

	// Cek apakah nasabah punya riwayat transaksi nyata
	type txCheck struct {
		table string
		count int64
	}
	checks := []txCheck{
		{table: "setoran_nasabah"},
		{table: "penarikan"},
		{table: "penerima_bagi_hasil"},
		{table: "tabungan_sampah"},
	}
	for _, chk := range checks {
		if err := nc.DB.Table(chk.table).Where("nasabah_id = ?", nasabahID).Count(&chk.count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi data nasabah"})
			return
		}
		if chk.count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Nasabah tidak dapat dihapus karena masih memiliki riwayat transaksi."})
			return
		}
	}

	if err := nc.DB.Transaction(func(tx *gorm.DB) error {
		// Ambil semua rekening_id milik nasabah ini
		var rekeningIDs []string
		if err := tx.Model(&models.SaldoRekening{}).
			Where("nasabah_id = ?", nasabahID).
			Pluck("rekening_id", &rekeningIDs).Error; err != nil {
			return err
		}
		if len(rekeningIDs) > 0 {
			if err := tx.Where("rekening_id IN ?", rekeningIDs).Delete(&models.RiwayatArusSaldo{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("nasabah_id = ?", nasabahID).Delete(&models.SaldoRekening{}).Error; err != nil {
			return err
		}
		return tx.Delete(&nasabah).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus nasabah"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Nasabah deleted successfully"})
}

func (nc *NasabahController) NasabahBankSampah(c *gin.Context) {
	bankID := c.Param("bank_id")

	type NasabahBankSampahResponse struct {
		NasabahID     string `json:"nasabah_id" gorm:"column:nasabah_id"`
		NamaNasabah   string `json:"nama_nasabah" gorm:"column:nama_nasabah"`
		Foto          string `json:"foto" gorm:"column:foto"`
		NIK           string `json:"nik" gorm:"column:nik"`
		Email         string `json:"email" gorm:"column:email"`
		StatusNasabah string `json:"status_nasabah" gorm:"column:status_nasabah"`
	}

	var results []NasabahBankSampahResponse

	query := nc.DB.Table("nasabah").
		Select("nasabah.nasabah_id, u.nama AS nama_nasabah, u.photo_url AS foto, nasabah.status_nasabah, u.user_id AS nik, u.email").
		Joins("LEFT JOIN users u ON nasabah.user_id = u.user_id").
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