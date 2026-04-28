package controllers

import (
	"enviroo-be/internal/models"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PenimbanganController struct {
	db *gorm.DB
}

func NewPenimbanganController(db *gorm.DB) *PenimbanganController {
	return &PenimbanganController{db: db}
}

func generatePenimbanganID() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%s-%04d", time.Now().Format("20060102150405"), r.Intn(10000))
}

func (pc *PenimbanganController) CheckJadwalHariIni(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := pc.db.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank sampah tidak ditemukan"})
		return
	}

	var activeCount int64
	if err := pc.db.Model(&models.Penimbangan{}).Where("bank_id = ? AND status_penimbangan = ?", bankID, models.StatusAktif).Count(&activeCount).Error; err == nil && activeCount > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"status": "active_session",
			"error": "Masih ada penimbangan yang aktif untuk BSU ini",
		})
		return
	}

	now := time.Now()
	todayDate := now.Format("2006-01-02")
	todayHari := []models.HariEnum{
		models.Minggu, models.Senin, models.Selasa, models.Rabu, models.Kamis, models.Jumat, models.Sabtu,
	}[now.Weekday()]
	mingguKe := getWeekOfMonth(now)

	var jadwal models.Jadwal
	err := pc.db.Where("bank_id = ? AND jenis_jadwal = ? AND ("+
		"(is_rutin = true AND hari = ? AND (minggu_ke = ? OR minggu_ke = 0)) OR "+
		"(is_rutin = false AND DATE(tanggal) = ?))",
		bankID, models.JadwalPenimbangan, todayHari, mingguKe, todayDate).First(&jadwal).Error

	jadwalTersedia := false
	if err == nil {
		var penimbanganCount int64
		pc.db.Model(&models.Penimbangan{}).Where("jadwal_id = ? AND DATE(started_at) = ?", jadwal.JadwalID, todayDate).Count(&penimbanganCount)
		if penimbanganCount == 0 {
			jadwalTersedia = true
		}
	}

	if !jadwalTersedia {
		c.JSON(http.StatusOK, gin.H{
			"status": "unscheduled",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "scheduled",
	})
}

func (pc *PenimbanganController) AddNewPenimbangan(c *gin.Context) {
	bankID := c.Param("bank_id")
	adminID := c.Param("admin_id")

	var req struct {
		ForceDadakan bool `json:"force_dadakan"`
	}
	// Opsional: jika request body tidak ada, ForceDadakan otomatis false
	c.ShouldBindJSON(&req)

	// 1. Cek bank sampah
	var bank models.BankSampah
	if err := pc.db.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank sampah tidak ditemukan"})
		return
	}

	// 2. Cek session aktif (mencegah penimbangan redundan di satu bank secara bersamaan)
	var activeCount int64
	if err := pc.db.Model(&models.Penimbangan{}).Where("bank_id = ? AND status_penimbangan = ?", bankID, models.StatusAktif).Count(&activeCount).Error; err == nil && activeCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Masih ada penimbangan yang aktif untuk BSU ini"})
		return
	}

	// 3. Match jadwal penimbangan hari ini
	now := time.Now()
	todayDate := now.Format("2006-01-02")
	todayHari := []models.HariEnum{
		models.Minggu, models.Senin, models.Selasa, models.Rabu, models.Kamis, models.Jumat, models.Sabtu,
	}[now.Weekday()]
	mingguKe := getWeekOfMonth(now)

	var jadwal models.Jadwal
	
	// Query cek apakah hari ini tercatat ada jadwal penimbangan secara formal
	err := pc.db.Where("bank_id = ? AND jenis_jadwal = ? AND ("+
		"(is_rutin = true AND hari = ? AND (minggu_ke = ? OR minggu_ke = 0)) OR "+
		"(is_rutin = false AND DATE(tanggal) = ?))",
		bankID, models.JadwalPenimbangan, todayHari, mingguKe, todayDate).First(&jadwal).Error

	jadwalTersedia := false
	if err == nil {
		var penimbanganCount int64
		pc.db.Model(&models.Penimbangan{}).Where("jadwal_id = ? AND DATE(started_at) = ?", jadwal.JadwalID, todayDate).Count(&penimbanganCount)
		if penimbanganCount == 0 {
			jadwalTersedia = true
		}
	}

	tx := pc.db.Begin()

	// Jika jadwal tidak ada atau sudah terpakai, cek status ForceDadakan. Jika false, tolak!
	if !jadwalTersedia {
		if !req.ForceDadakan {
			tx.Rollback()
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak ada jadwal penimbangan hari ini. Konfirmasi sesi dadakan diperlukan."})
			return
		}

		trueVal := true
		falseVal := false
		
		jamMulai := now.Format("15:04")
		endTime := now.Add(2 * time.Hour)
		jamSelesai := endTime.Format("15:04")

		// Jika melewati tengah malam, paksa ke 23:59 agar tidak melanggar constraint (jam_selesai > jam_mulai)
		if endTime.Day() != now.Day() {
			jamSelesai = "23:59"
		}

		jadwal = models.Jadwal{
			JadwalID:          uuid.New(),
			BankID:            bankID,
			Hari:              todayHari,
			MingguKe:          mingguKe,
			JenisJadwal:       models.JadwalPenimbangan,
			JamMulai:          jamMulai,
			JamSelesai:        jamSelesai,
			IsActive:          &trueVal,
			IsRutin:           &falseVal,
			Tanggal:           now,
			NamaJadwalSpesial: "Penimbangan Dadakan",
			CreatedBy:         adminID,
		}
		if err := tx.Create(&jadwal).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat jadwal dadakan"})
			return
		}
	}

	// 4. Create penimbangan record
	nowTime := time.Now()
	newPenimbangan := models.Penimbangan{
		PenimbanganID:     generatePenimbanganID(),
		JadwalID:          &jadwal.JadwalID,
		BankID:            &bankID,
		StartedBy:         &adminID,
		StartedAt:         &nowTime,
		StatusPenimbangan: models.StatusAktif,
	}

	if err := tx.Create(&newPenimbangan).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai sesi penimbangan"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusCreated, gin.H{
		"message": "Sesi penimbangan berhasil dimulai",
		"data":    newPenimbangan,
	})
}

func (pc *PenimbanganController) UpdatePenimbangan(c *gin.Context) {
	penimbanganID := c.Param("penimbangan_id")
	adminID := c.Param("admin_id")

	var req struct {
		StatusPenimbangan models.StatusPenimbangan `json:"status_penimbangan" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status penimbangan harus disertakan (selesai/dibatalkan)"})
		return
	}

	// Validasi status string
	if req.StatusPenimbangan != models.StatusSelesai && req.StatusPenimbangan != models.StatusDibatalkan {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status penimbangan tidak valid. Gunakan 'selesai' atau 'dibatalkan'"})
		return
	}

	var penimbangan models.Penimbangan
	if err := pc.db.Where("penimbangan_id = ?", penimbanganID).First(&penimbangan).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data penimbangan tidak ditemukan"})
		return
	}

	// Cek apakah penimbangan sudah pernah di-end (tidak aktif lagi)
	if penimbangan.StatusPenimbangan != models.StatusAktif {
		c.JSON(http.StatusConflict, gin.H{"error": "Sesi penimbangan ini sudah diakhiri (selesai/dibatalkan) sebelumnya"})
		return
	}

	nowTime := time.Now()
	penimbangan.EndedBy = &adminID
	penimbangan.EndedAt = &nowTime
	penimbangan.StatusPenimbangan = req.StatusPenimbangan

	if err := pc.db.Save(&penimbangan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan status penimbangan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Status sesi penimbangan berhasil diperbarui",
		"data":    penimbangan,
	})
}

func (pc *PenimbanganController) GetPenimbangan(c *gin.Context) {
	bankID := c.Param("bank_id")

	type PenimbanganResponse struct {
		models.Penimbangan
		NamaAdmin string `json:"nama_admin" gorm:"column:nama"`
	}

	var penimbanganList []PenimbanganResponse

	if err := pc.db.Table("penimbangan").
		Select("penimbangan.*, users.nama").
		Joins("LEFT JOIN users ON users.user_id = penimbangan.started_by").
		Where("penimbangan.bank_id = ?", bankID).
		Order("penimbangan.started_at desc").
		Find(&penimbanganList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data penimbangan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil data penimbangan",
		"data":    penimbanganList,
	})
}

func (pc *PenimbanganController) ListSetoranPenimbangan(c *gin.Context) {
	penimbanganID := c.Param("penimbangan_id")

	type ListSetoranResponse struct {
		SetoranID          string               `json:"setoran_id" gorm:"column:setoran_id"`
		NamaPetugas        string               `json:"nama_petugas" gorm:"column:nama_petugas"`
		NamaNasabah        string               `json:"nama_nasabah" gorm:"column:nama_nasabah"`
		TransaksiTimestamp time.Time            `json:"transaksi_timestamp" gorm:"column:transaksi_timestamp"`
		TotalItem          int                  `json:"total_item" gorm:"column:total_item"`
		TotalPoin          int                  `json:"total_poin" gorm:"column:total_poin"`
		StatusSetoran      models.StatusSetoran `json:"status_setoran" gorm:"column:status_setoran"`
	}

	var listSetoran []ListSetoranResponse
	if err := pc.db.Table("setoran_nasabah").
		Select("setoran_nasabah.setoran_id, u_petugas.nama as nama_petugas, u_nasabah.nama as nama_nasabah, setoran_nasabah.created_at as transaksi_timestamp, setoran_nasabah.total_item, setoran_nasabah.total_poin, setoran_nasabah.status_setoran").
		Joins("LEFT JOIN admin ON admin.admin_id = setoran_nasabah.admin_id").
		Joins("LEFT JOIN users u_petugas ON u_petugas.user_id = admin.user_id").
		Joins("LEFT JOIN nasabah ON nasabah.nasabah_id = setoran_nasabah.nasabah_id").
		Joins("LEFT JOIN users u_nasabah ON u_nasabah.user_id = nasabah.user_id").
		Where("setoran_nasabah.penimbangan_id = ?", penimbanganID).
		Order("setoran_nasabah.created_at desc").
		Find(&listSetoran).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data setoran"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil data setoran",
		"data":    listSetoran,
	})
}