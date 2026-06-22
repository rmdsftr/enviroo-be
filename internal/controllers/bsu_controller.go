package controllers

import (
	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BSUController struct {
	DB        *gorm.DB
	CFStorage *storage.CloudflareStorage
	Mailer    *utils.Mailer
}

func NewBSUController(db *gorm.DB, cfStorage *storage.CloudflareStorage, mailer *utils.Mailer) *BSUController {
	return &BSUController{
		DB:        db,
		CFStorage: cfStorage,
		Mailer:    mailer,
	}
}

type AddBSURequest struct {
	NamaBSU       string   `form:"nama_bsu" binding:"required"`
	ParentBankID  string   `form:"parent_bank_id" binding:"required"`
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

func (bc *BSUController) AddNewBSU(c *gin.Context) {
	var req AddBSURequest
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

	bankID, err := utils.GenerateBankID(tx, models.BSU, req.IDKecamatan, req.IDKelurahan)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal generate Bank ID: " + err.Error()})
		return
	}

	newBSU := models.BankSampah{
		BankID:        bankID,
		NamaBank:      req.NamaBSU,
		Deskripsi:     req.Deskripsi,
		ParentBankID:  &req.ParentBankID,
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

	// Pre-create saldo rekening Uang nominal 0 (BSU hanya menerima distribusi Uang)
	var rewardUangBSU models.Reward
	if err := tx.Where("nama_reward = ?", models.RewardEnumUang).First(&rewardUangBSU).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Reward Uang tidak ditemukan: " + err.Error()})
		return
	}
	rekeningBSU := models.SaldoRekening{
		RekeningID:         utils.GenerateRekeningID(rewardUangBSU.RewardID, bankID),
		BankID:             &bankID,
		RewardID:           &rewardUangBSU.RewardID,
		Entitas:            models.EntitasBankSampah,
		NominalSaldo:       0,
		SatuanNominalSaldo: models.SatuanRewardEnumRp,
	}
	if err := tx.Create(&rekeningBSU).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat rekening bank: " + err.Error()})
		return
	}

	// Create admins for this BSU
	for _, userID := range req.UserIDs {
		// Pengecekan nasabah (meskipun bank baru terbuat, ini ditambahkan sesuai instruksi)
		var countNasabah int64
		if err := tx.Model(&models.Nasabah{}).Where("user_id = ? AND bank_id = ?", userID, newBSU.BankID).Count(&countNasabah).Error; err == nil && countNasabah > 0 {
			tx.Rollback()
			c.JSON(http.StatusConflict, gin.H{"error": "User tidak boleh terdaftar sebagai admin di bank sampah tempat ia terdaftar sebagai nasabah"})
			return
		}

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
					Anda telah ditunjuk sebagai <b>Administrator</b> di <b>%s</b> (BSU).
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

		sendErr := bc.Mailer.SendEmail(utils.EmailParams{
			To:      existingUser.Email,
			Subject: "Aktivasi Akun Administrator (BSU) Enviroo",
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
		"message": "BSU and admins created successfully",
		"data":    newBSU,
	})
}

func (bc *BSUController) GetBSU(c *gin.Context) {
	type BSUResponse struct {
		BankID        string `json:"bank_id" gorm:"column:bank_id"`
		NamaBSU       string `json:"nama_bsu" gorm:"column:nama_bank"`
		PhotoURL      string `json:"photo_url" gorm:"column:photo_url"`
		IsActive      bool   `json:"is_active" gorm:"column:is_active"`
		NamaBankInduk string `json:"nama_bank_induk" gorm:"column:nama_bank_induk"`
		JumlahNasabah int64  `json:"jumlah_nasabah" gorm:"column:jumlah_nasabah"`
		JumlahStaff   int64  `json:"jumlah_staff" gorm:"column:jumlah_staff"`
		TotalCount    int    `json:"-" gorm:"column:total_count"`
	}

	pageStr := c.Query("page")

	const selectCols = "bank_sampah.bank_id, bank_sampah.nama_bank, bank_sampah.photo_url, bank_sampah.is_active, " +
		"(SELECT nama_bank FROM bank_sampah bsi WHERE bsi.bank_id = bank_sampah.parent_bank_id) AS nama_bank_induk, " +
		"(SELECT COUNT(user_id) FROM nasabah WHERE nasabah.bank_id = bank_sampah.bank_id) AS jumlah_nasabah, " +
		"(SELECT COUNT(user_id) FROM admin WHERE admin.bank_id = bank_sampah.bank_id) AS jumlah_staff"

	if pageStr == "" {
		var results []BSUResponse
		query := bc.DB.Model(&models.BankSampah{}).
			Select(selectCols).
			Where("bank_sampah.jenis_bank = ?", models.BSU)
		if err := query.Find(&results).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get BSU: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "BSU fetched successfully", "data": results})
		return
	}

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	const limit = 20
	offset := (page - 1) * limit

	var results []BSUResponse
	query := bc.DB.Model(&models.BankSampah{}).
		Select(selectCols+", COUNT(*) OVER() AS total_count").
		Where("bank_sampah.jenis_bank = ?", models.BSU).
		Limit(limit).Offset(offset)
	if err := query.Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get BSU: " + err.Error()})
		return
	}

	totalCount := 0
	if len(results) > 0 {
		totalCount = results[0].TotalCount
	}
	totalPages := (totalCount + limit - 1) / limit

	c.JSON(http.StatusOK, gin.H{
		"message": "BSU fetched successfully",
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       totalCount,
			"total_pages": totalPages,
		},
		"data": results,
	})
}

func (bc *BSUController) GetBSUbyBankID(c *gin.Context) {
	bankID := c.Param("bank_id")

	type BSUResponse struct {
		BankID        string `json:"bank_id" gorm:"column:bank_id"`
		NamaBSU       string `json:"nama_bsu" gorm:"column:nama_bank"`
		JumlahNasabah int64  `json:"jumlah_nasabah" gorm:"column:jumlah_nasabah"`
		JumlahStaff   int64  `json:"jumlah_staff" gorm:"column:jumlah_staff"`
		IsActive      bool   `json:"is_active" gorm:"column:is_active"`
	}

	var results []BSUResponse

	query := bc.DB.Model(&models.BankSampah{}).
		Select("bank_sampah.bank_id, bank_sampah.nama_bank, bank_sampah.is_active, " +
			"(SELECT COUNT(user_id) FROM nasabah WHERE nasabah.bank_id = bank_sampah.bank_id) AS jumlah_nasabah, " +
			"(SELECT COUNT(user_id) FROM admin WHERE admin.bank_id = bank_sampah.bank_id) AS jumlah_staff").
		Where("bank_sampah.jenis_bank = ? AND bank_sampah.parent_bank_id = ?", models.BSU, bankID)

	if err := query.Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get BSU: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "BSU fetched successfully",
		"data":    results,
	})
}

func (bc *BSUController) NonAktifkanBSU(c *gin.Context) {
	bsuID := c.Param("bsu_id")

	type RequestDeactivate struct {
		AdminID   string `json:"admin_id"`
		Informasi string `json:"informasi"`
	}

	var req RequestDeactivate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var bsu models.BankSampah
	if err := bc.DB.Where("bank_id = ?", bsuID).First(&bsu).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "BSU not found"})
		return
	}

	bsu.IsActive = false
	if err := bc.DB.Save(&bsu).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate BSU"})
		return
	}

	newHistory := models.HistoryAkunBank{
		BankID: bsuID,
		Action: "UPDATE",
		OldValue: datatypes.JSON([]byte(`{
			"status_bank" : "Aktif"
		}`)),
		NewValue: datatypes.JSON([]byte(`{
			"status_bank" : "Non Aktif"
		}`)),
		Informasi: req.Informasi,
		CreatedBy: req.AdminID,
	}

	if err := bc.DB.Create(&newHistory).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "BSU deactivated successfully",
		"data":    bsu,
	})
}

func (bc *BSUController) AktifkanBSU(c *gin.Context) {
	bsuID := c.Param("bsu_id")

	var bsu models.BankSampah
	if err := bc.DB.Where("bank_id = ?", bsuID).First(&bsu).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "BSU not found"})
		return
	}

	bsu.IsActive = true
	if err := bc.DB.Save(&bsu).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to activate BSU"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "BSU activated successfully",
		"data":    bsu,
	})
}

