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

type BSIController struct {
	DB        *gorm.DB
	CFStorage *storage.CloudflareStorage
	Mailer    *utils.Mailer
}

func NewBSIController(db *gorm.DB, cfStorage *storage.CloudflareStorage, mailer *utils.Mailer) *BSIController {
	return &BSIController{
		DB:        db,
		CFStorage: cfStorage,
		Mailer:    mailer,
	}
}

type AddBSIRequest struct {
	NamaBSI       string   `form:"nama_bsi" binding:"required"`
	Deskripsi     string   `form:"deskripsi" binding:"required"`
	Provinsi      string   `form:"provinsi" binding:"required"`
	KabupatenKota string   `form:"kabupaten_kota" binding:"required"`
	IDKecamatan   int      `form:"id_kecamatan" binding:"required"`
	IDKelurahan   int      `form:"id_kelurahan" binding:"required"`
	AlamatLengkap string   `form:"alamat_lengkap" binding:"required"`
	Latitude      float64  `form:"latitude"`
	Longitude     float64  `form:"longitude"`
	UserIDs       []string `form:"user_id[]"`
	AdminID       string   `form:"admin_id" binding:"required"`
}

func (bc *BSIController) AddNewBSI(c *gin.Context) {
	var req AddBSIRequest
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

		fotoURL, err = bc.CFStorage.UploadFile(file, fileHeader, "bsi_photos")
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

	bankID, err := utils.GenerateBankID(tx, models.BSI, req.IDKecamatan, req.IDKelurahan)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal generate Bank ID: " + err.Error()})
		return
	}

	newBSI := models.BankSampah{
		BankID:        bankID,
		NamaBank:      req.NamaBSI,
		Deskripsi:     req.Deskripsi,
		PhotoURL:      fotoURL,
		Provinsi:      req.Provinsi,
		KabupatenKota: req.KabupatenKota,
		IDKecamatan:   &req.IDKecamatan,
		IDKelurahan:   &req.IDKelurahan,
		Alamat:        req.AlamatLengkap,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		JenisBank:     models.BSI,
		IsActive:      true,
	}

	if err := tx.Create(&newBSI).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create BSI: " + err.Error()})
		return
	}

	// Pre-create saldo rekening bank nominal 0
	var rewardUangBSI, rewardSembakoBSI models.Reward
	if err := tx.Where("nama_reward = ?", models.RewardEnumUang).First(&rewardUangBSI).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Reward Uang tidak ditemukan: " + err.Error()})
		return
	}
	if err := tx.Where("nama_reward = ?", models.RewardEnumSembako).First(&rewardSembakoBSI).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Reward Sembako tidak ditemukan: " + err.Error()})
		return
	}
	for _, rw := range []models.Reward{rewardUangBSI, rewardSembakoBSI} {
		rwCopy := rw
		satuan := models.SatuanRewardEnumRp
		if models.SatuanRewardEnum(rw.Satuan) == models.SatuanRewardEnumPoin {
			satuan = models.SatuanRewardEnumPoin
		}
		rekening := models.SaldoRekening{
			RekeningID:         utils.GenerateRekeningID(rwCopy.RewardID, bankID),
			BankID:             &bankID,
			RewardID:           &rwCopy.RewardID,
			Entitas:            models.EntitasBankSampah,
			NominalSaldo:       0,
			SatuanNominalSaldo: satuan,
		}
		if err := tx.Create(&rekening).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat rekening bank: " + err.Error()})
			return
		}
	}

	// Create admins for this BSI
	for _, userID := range req.UserIDs {
		// Pengecekan status (tetap dipertahankan kalau dia nasabah atau admin di tempat yang sama, sesuai role)
		var countNasabah int64
		if err := tx.Model(&models.Nasabah{}).Where("user_id = ? AND bank_id = ?", userID, newBSI.BankID).Count(&countNasabah).Error; err == nil && countNasabah > 0 {
			tx.Rollback()
			c.JSON(http.StatusConflict, gin.H{"error": "User tidak boleh terdaftar sebagai admin di bank sampah tempat ia terdaftar sebagai nasabah"})
			return
		}

		adminID, err := utils.GenerateAdminID(tx, models.AdminBSI, newBSI.BankID)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate admin ID: " + err.Error()})
			return
		}

		newAdmin := models.Admin{
			AdminID:     adminID,
			UserID:      userID,
			BankID:      &newBSI.BankID,
			Role:        models.AdminBSI,
			StatusAdmin: models.Pending,
		}

		if err := tx.Create(&newAdmin).Error; err != nil {
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
			Tujuan:      models.TujuanAktivasi,
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
					Anda telah ditunjuk sebagai <b>Administrator</b> di Bank Sampah <b>%s</b>.
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
		`, existingUser.Nama, newBSI.NamaBank, otp, newAktivasiAkun.ExpiredAt.Format("02 Jan 2006 15:04 WIB"))

		sendErr := bc.Mailer.SendEmail(utils.EmailParams{
			To:      existingUser.Email,
			Subject: "Aktivasi Akun Administrator Enviroo",
			Body:    emailBody,
		})

		if sendErr != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email: " + sendErr.Error()})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "BSI and admins created successfully",
		"data":    newBSI,
	})
}

func (bc *BSIController) GetBSI(c *gin.Context) {
	type BSIResponse struct {
		models.BankSampah
		JumlahBSU     int64 `json:"jumlah_bsu" gorm:"column:jumlah_bsu"`
		JumlahNasabah int64 `json:"jumlah_nasabah" gorm:"column:jumlah_nasabah"`
	}

	var results []BSIResponse

	// Query utama dengan subquery SELECT COUNT untuk menghitung anak cabang (BSU) dan jumlah nasabah
	query := bc.DB.Model(&models.BankSampah{}).
		Select("bank_sampah.*, (SELECT COUNT(bank_id) FROM bank_sampah bsu WHERE bsu.parent_bank_id = bank_sampah.bank_id AND bsu.jenis_bank = 'bsu') AS jumlah_bsu, " +
			"(SELECT COUNT(nasabah_id) FROM nasabah WHERE nasabah.bank_id = bank_sampah.bank_id) AS jumlah_nasabah").
		Where("bank_sampah.jenis_bank = ?", models.BSI)

	if err := query.Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get BSI: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "BSI fetched successfully",
		"data":    results,
	})
}

func (bc *BSIController) GetUnitBSI(c *gin.Context) {
	bankID := c.Param("bank_id")
	if bankID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank ID is required"})
		return
	}

	type BSUResponse struct {
		models.BankSampah
		JumlahNasabah int64 `json:"jumlah_nasabah" gorm:"column:jumlah_nasabah"`
	}

	var results []BSUResponse

	// Query utama dengan subquery SELECT COUNT untuk menghitung jumlah nasabah
	query := bc.DB.Model(&models.BankSampah{}).
		Select("bank_sampah.*, (SELECT COUNT(nasabah_id) FROM nasabah WHERE nasabah.bank_id = bank_sampah.bank_id) AS jumlah_nasabah").
		Where("bank_sampah.parent_bank_id = ? AND bank_sampah.jenis_bank = ?", bankID, models.BSU)

	if err := query.Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get BSU: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "BSU fetched successfully",
		"data":    results,
	})
}

type AddUnitRequest struct {
	NamaUnit      string   `form:"nama_unit" binding:"required"`
	Deskripsi     string   `form:"deskripsi" binding:"required"`
	Provinsi      string   `form:"provinsi" binding:"required"`
	KabupatenKota string   `form:"kabupaten_kota" binding:"required"`
	IDKecamatan   int      `form:"id_kecamatan" binding:"required"`
	IDKelurahan   int      `form:"id_kelurahan" binding:"required"`
	AlamatLengkap string   `form:"alamat_lengkap" binding:"required"`
	Latitude      float64  `form:"latitude"`
	Longitude     float64  `form:"longitude"`
	UserIDs       []string `form:"user_id[]"`
	AdminID       string   `form:"admin_id" binding:"required"`
}

func (bc *BSIController) AddNewUnit(c *gin.Context) {
	bankID := c.Param("bank_id")
	if bankID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank ID is required"})
		return
	}

	var req AddUnitRequest
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

		fotoURL, err = bc.CFStorage.UploadFile(file, fileHeader, "bsu_photos")
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

	newUnitBankID, err := utils.GenerateBankID(tx, models.BSU, req.IDKecamatan, req.IDKelurahan)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal generate Bank ID: " + err.Error()})
		return
	}

	newBSU := models.BankSampah{
		BankID:        newUnitBankID,
		NamaBank:      req.NamaUnit,
		Deskripsi:     req.Deskripsi,
		ParentBankID:  &bankID,
		PhotoURL:      fotoURL,
		Provinsi:      req.Provinsi,
		KabupatenKota: req.KabupatenKota,
		IDKecamatan:   &req.IDKecamatan,
		IDKelurahan:   &req.IDKelurahan,
		Alamat:        req.AlamatLengkap,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		JenisBank:     models.BSU,
		IsActive:      true,
	}

	if err := tx.Create(&newBSU).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create BSU: " + err.Error()})
		return
	}

	// Pre-create saldo rekening Uang nominal 0 untuk BSU
	var rewardUangBSU models.Reward
	if err := tx.Where("nama_reward = ?", models.RewardEnumUang).First(&rewardUangBSU).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Reward Uang tidak ditemukan: " + err.Error()})
		return
	}
	rekeningBSU := models.SaldoRekening{
		RekeningID:         utils.GenerateRekeningID(rewardUangBSU.RewardID, newUnitBankID),
		BankID:             &newUnitBankID,
		RewardID:           &rewardUangBSU.RewardID,
		Entitas:            models.EntitasBankSampah,
		NominalSaldo:       0,
		SatuanNominalSaldo: models.SatuanRewardEnumRp,
	}
	if err := tx.Create(&rekeningBSU).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat rekening BSU: " + err.Error()})
		return
	}

	// Create admins for this BSU
	for _, userID := range req.UserIDs {
		adminID, err := utils.GenerateAdminID(tx, models.AdminBSU, newBSU.BankID)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate admin ID: " + err.Error()})
			return
		}

		newAdmin := models.Admin{
			AdminID:     adminID,
			UserID:      userID,
			BankID:      &newBSU.BankID,
			Role:        models.AdminBSU,
			StatusAdmin: models.Pending,
		}

		if err := tx.Create(&newAdmin).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add admin: " + err.Error()})
			return
		}

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
			Tujuan:      models.TujuanAktivasi,
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
					Anda telah ditunjuk sebagai <b>Administrator</b> di Bank Sampah <b>%s</b> (BSU).
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
		`, existingUser.Nama, newBSU.NamaBank, otp, newAktivasiAkun.ExpiredAt.Format("02 Jan 2006 15:04 WIB"))

		if sendErr := bc.Mailer.SendEmail(utils.EmailParams{
			To:      existingUser.Email,
			Subject: "Aktivasi Akun Administrator (BSU) Enviroo",
			Body:    emailBody,
		}); sendErr != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email: " + sendErr.Error()})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "BSU and admins created successfully",
		"data":    newBSU,
	})
}
