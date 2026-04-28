package controllers

import (
	"enviroo-be/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type JadwalController struct {
	db *gorm.DB
}

func NewJadwalController(db *gorm.DB) *JadwalController {
	return &JadwalController{db: db}
}

func getWeekOfMonth(date time.Time) int {
	firstOfMonth := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
	firstWeekday := int(firstOfMonth.Weekday())
	adjustedDay := date.Day() + firstWeekday - 1
	return (adjustedDay / 7) + 1
}

func (jc *JadwalController) AddNewJadwal(c *gin.Context) {
	bankID := c.Param("bank_id")
	if bankID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank ID is required"})
		return
	}

	var bank models.BankSampah
	if err := jc.db.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank not found"})
		return
	}

	type AddNewJadwalRequest struct {
		Hari              models.HariEnum   `json:"hari"`
		MingguKe          int               `json:"minggu_ke"`
		JamMulai          string            `json:"jam_mulai"`
		JamSelesai        string            `json:"jam_selesai"`
		JenisJadwal       models.JadwalEnum `json:"jenis_jadwal"`
		TargetBankID      string            `json:"target_bank_id"`
		IsActive          bool              `json:"is_active"`
		IsRutin           bool              `json:"is_rutin"`
		Tanggal           string            `json:"tanggal"`
		NamaJadwalSpesial string            `json:"nama_jadwal_spesial"`
		AdminID      string            `json:"admin_id"`
	}

	var req AddNewJadwalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var tanggal time.Time
	if !req.IsRutin && req.Tanggal != "" {
		t, err := time.Parse("2006-01-02", req.Tanggal)
		if err != nil {
			t, err = time.Parse(time.RFC3339, req.Tanggal)
		}
		if err == nil {
			tanggal = t
			if req.Hari == "" {
				weekdays := []models.HariEnum{models.Minggu, models.Senin, models.Selasa, models.Rabu, models.Kamis, models.Jumat, models.Sabtu}
				req.Hari = weekdays[t.Weekday()]
			}
			if req.MingguKe == 0 {
				req.MingguKe = getWeekOfMonth(t)
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Tanggal format. Use YYYY-MM-DD"})
			return
		}
	}

	if req.IsRutin {
		tanggal = time.Time{}
		req.NamaJadwalSpesial = ""
	}

	if req.Hari == "" {
		req.Hari = models.Senin // Fallback to avoid Postgres enum error empty string
	}

	if bank.JenisBank == models.BSU {
		req.TargetBankID = ""
	}

	newJadwal := models.Jadwal{
		BankID:            bankID,
		Hari:              req.Hari,
		MingguKe:          req.MingguKe,
		JamMulai:          req.JamMulai,
		JamSelesai:        req.JamSelesai,
		JenisJadwal:       req.JenisJadwal,
		TargetBankID:      req.TargetBankID,
		IsActive:          &req.IsActive,
		IsRutin:           &req.IsRutin,
		Tanggal:           tanggal,
		NamaJadwalSpesial: req.NamaJadwalSpesial,
		CreatedBy:         req.AdminID,
	}

	if err := jc.db.Create(&newJadwal).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jadwal berhasil ditambahkan", "data": newJadwal})
}

func (jc *JadwalController) GetJadwalBank(c *gin.Context) {
	bankID := c.Param("bank_id")
	if bankID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank ID is required"})
		return
	}

	var bank models.BankSampah
	if err := jc.db.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank not found"})
		return
	}

	type PenimbanganResponse struct {
		models.Jadwal
		BankName string `json:"bank_name"`
	}

	type PengangkutanResponse struct {
		models.Jadwal
		BankName       string `json:"bank_name"`
		TargetBankName string `json:"target_bank_name"`
	}

	var listPenimbangan []PenimbanganResponse
	var listPengangkutan []PengangkutanResponse

	// Get Penimbangan where bank_id = bankID and jenis_jadwal = "penimbangan"
	if err := jc.db.Table("jadwal").
		Select("jadwal.*, bank_sampah.nama_bank as bank_name").
		Joins("left join bank_sampah on bank_sampah.bank_id = jadwal.bank_id").
		Where("jadwal.bank_id = ? AND jadwal.jenis_jadwal = ?", bankID, models.JadwalPenimbangan).
		Find(&listPenimbangan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get Pengangkutan based on JenisBank
	if bank.JenisBank == models.BSU {
		// BSU fetches pengangkutan where they are the target
		if err := jc.db.Table("jadwal").
			Select("jadwal.*, b1.nama_bank as bank_name, b2.nama_bank as target_bank_name").
			Joins("left join bank_sampah b1 on b1.bank_id = jadwal.bank_id").
			Joins("left join bank_sampah b2 on b2.bank_id = jadwal.target_bank_id").
			Where("jadwal.target_bank_id = ? AND jadwal.jenis_jadwal = ?", bankID, models.JadwalPengangkutan).
			Find(&listPengangkutan).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// Other banks fetch pengangkutan where they are the initiator (bank_id)
		if err := jc.db.Table("jadwal").
			Select("jadwal.*, b1.nama_bank as bank_name, b2.nama_bank as target_bank_name").
			Joins("left join bank_sampah b1 on b1.bank_id = jadwal.bank_id").
			Joins("left join bank_sampah b2 on b2.bank_id = jadwal.target_bank_id").
			Where("jadwal.bank_id = ? AND jadwal.jenis_jadwal = ?", bankID, models.JadwalPengangkutan).
			Find(&listPengangkutan).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"penimbangan": listPenimbangan,
			"pengangkutan": listPengangkutan,
		},
	})
}

func (jc *JadwalController) DeleteJadwal(c *gin.Context) {
	jadwalID := c.Param("jadwal_id")
	if jadwalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jadwal ID is required"})
		return
	}

	var jadwal models.Jadwal
	if err := jc.db.Where("jadwal_id = ?", jadwalID).First(&jadwal).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jadwal not found"})
		return
	}

	if err := jc.db.Delete(&jadwal).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jadwal berhasil dihapus"})
}

func (jc *JadwalController) UpdateJadwal(c *gin.Context) {
	jadwalID := c.Param("jadwal_id")
	if jadwalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jadwal ID is required"})
		return
	}

	var jadwal models.Jadwal
	if err := jc.db.Where("jadwal_id = ?", jadwalID).First(&jadwal).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jadwal not found"})
		return
	}

	type UpdateJadwalRequest struct {
		Hari         models.HariEnum   `json:"hari"`
		MingguKe     int               `json:"minggu_ke"`
		JamMulai     string            `json:"jam_mulai"`
		JamSelesai   string            `json:"jam_selesai"`
		TargetBankID string            `json:"target_bank_id"`
		IsActive     *bool             `json:"is_active"`
		Tanggal      string            `json:"tanggal"`
		NamaJadwalSpesial string         `json:"nama_jadwal_spesial"`
		AdminID      string            `json:"admin_id"`
	}

	var req UpdateJadwalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var tanggal time.Time
	var err error
	
	// Handle Tanggal parsing only if it is provided and the schedule is not rutin
	// If it's rutin, we don't allow modifying Tanggal
	if req.Tanggal != "" && jadwal.IsRutin != nil && !*jadwal.IsRutin {
		tanggal, err = time.Parse("2006-01-02", req.Tanggal)
		if err != nil {
			tanggal, err = time.Parse(time.RFC3339, req.Tanggal)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Tanggal format. Use YYYY-MM-DD"})
				return
			}
		}
	}

	if req.TargetBankID != "" {
		jadwal.TargetBankID = req.TargetBankID
	}

	if req.Hari != "" {
		jadwal.Hari = req.Hari
	}

	if req.MingguKe != 0 {
		jadwal.MingguKe = req.MingguKe
	}

	if req.JamMulai != "" {
		jadwal.JamMulai = req.JamMulai
	}

	if req.JamSelesai != "" {
		jadwal.JamSelesai = req.JamSelesai
	}

	if req.IsActive != nil {
		jadwal.IsActive = req.IsActive
	}

	// Update tanggal only if it's explicitly provided and the schedule is not rutin
	if req.Tanggal != "" && jadwal.IsRutin != nil && !*jadwal.IsRutin {
		jadwal.Tanggal = tanggal
	}
	
	if req.NamaJadwalSpesial != "" && jadwal.IsRutin != nil && !*jadwal.IsRutin {
		jadwal.NamaJadwalSpesial = req.NamaJadwalSpesial
	}

	if req.AdminID != "" {
		jadwal.CreatedBy = req.AdminID
	}

	if err := jc.db.Save(&jadwal).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jadwal berhasil diupdate", "data": jadwal})
}
