package controllers

import (
	"context"
	"encoding/json"
	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PengangkutanController struct {
	db        *gorm.DB
	cfStorage *storage.CloudflareStorage
	notifSvc  services.NotifikasiService
}

func NewPengangkutanController(db *gorm.DB, cfStorage *storage.CloudflareStorage, notifSvc services.NotifikasiService) *PengangkutanController {
	return &PengangkutanController{db: db, cfStorage: cfStorage, notifSvc: notifSvc}
}

func (p *PengangkutanController) CheckJadwalPengangkutan(c *gin.Context) {
	bsiID := c.Param("bsi_id")
	bsuID := c.Param("bsu_id")

	var bsi, bsu models.BankSampah
	if err := p.db.Where("bank_id = ? AND is_active = ?", bsiID, true).First(&bsi).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "BSI tidak ditemukan atau tidak aktif"})
		return
	}
	if err := p.db.Where("bank_id = ? AND parent_bank_id = ? AND is_active = ? AND jenis_bank = ?", bsuID, bsiID, true, models.BSU).First(&bsu).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "BSU tidak ditemukan atau tidak aktif"})
		return
	}

	now := time.Now()
	todayDate := now.Format("2006-01-02")
	todayHari := []models.HariEnum{
		models.Minggu, models.Senin, models.Selasa, models.Rabu, models.Kamis, models.Jumat, models.Sabtu,
	}[now.Weekday()]
	mingguKe := getWeekOfMonth(now)

	var jadwal models.Jadwal
	err := p.db.Where("bank_id = ? AND target_bank_id = ? AND jenis_jadwal = ? AND ("+
		"(is_rutin = true AND hari = ? AND (minggu_ke = ? OR minggu_ke = 0)) OR "+
		"(is_rutin = false AND tanggal = ?))",
		bsiID, bsuID, models.JadwalPengangkutan, todayHari, mingguKe, todayDate).First(&jadwal).Error

	if err != nil {
		// Jadwal tidak ditemukan untuk hari ini (rutin maupun khusus), status dadakan
		c.JSON(http.StatusOK, gin.H{
			"status": "dadakan",
		})
		return
	}

	// Jadwal ditemukan untuk hari ini
	c.JSON(http.StatusOK, gin.H{
		"status": "scheduled",
	})
}

func (p *PengangkutanController) StartSesiPengangkutan(c *gin.Context) {
	type request struct {
		BSIID         string `json:"bsi_id" binding:"required"`
		BSUID         string `json:"bsu_id" binding:"required"`
		AdminBSIID    string `json:"admin_bsi_id" binding:"required"`
		StatusDadakan bool   `json:"status_dadakan"`
	}

	var req request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Cek bank sampah
	var bsi, bsu models.BankSampah
	if err := p.db.Where("bank_id = ? AND is_active = ?", req.BSIID, true).First(&bsi).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BSI tidak ditemukan atau tidak aktif"})
		return
	}
	if err := p.db.Where("bank_id = ? AND parent_bank_id = ? AND is_active = ? AND jenis_bank = ?", req.BSUID, req.BSIID, true, models.BSU).First(&bsu).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BSU tidak ditemukan atau tidak aktif"})
		return
	}

	// 2. Match jadwal pengangkutan hari ini
	now := time.Now()
	todayDate := now.Format("2006-01-02")
	todayHari := []models.HariEnum{
		models.Minggu, models.Senin, models.Selasa, models.Rabu, models.Kamis, models.Jumat, models.Sabtu,
	}[now.Weekday()]
	mingguKe := getWeekOfMonth(now)

	var jadwal models.Jadwal
	err := p.db.Where("bank_id = ? AND target_bank_id = ? AND jenis_jadwal = ? AND ("+
		"(is_rutin = true AND hari = ? AND (minggu_ke = ? OR minggu_ke = 0)) OR "+
		"(is_rutin = false AND tanggal = ?))",
		req.BSIID, req.BSUID, models.JadwalPengangkutan, todayHari, mingguKe, todayDate).First(&jadwal).Error

	jadwalTersedia := (err == nil)

	tx := p.db.Begin()

	if !jadwalTersedia {
		if !req.StatusDadakan {
			tx.Rollback()
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak ada jadwal pengangkutan hari ini. Konfirmasi sesi dadakan diperlukan."})
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
			BankID:            req.BSIID,
			TargetBankID:      req.BSUID,
			Hari:              todayHari,
			MingguKe:          mingguKe,
			JenisJadwal:       models.JadwalPengangkutan,
			JamMulai:          jamMulai,
			JamSelesai:        jamSelesai,
			IsActive:          &trueVal,
			IsRutin:           &falseVal,
			Tanggal:           now,
			NamaJadwalSpesial: "Pengangkutan Dadakan",
			CreatedBy:         req.AdminBSIID,
		}
		if err := tx.Create(&jadwal).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat jadwal dadakan"})
			return
		}
	}

	// 3. Create pengangkutan_sampah record
	pengangkutanID := utils.GenerateID("PGK")
	newPengangkutan := models.PengangkutanSampah{
		PengangkutanID: pengangkutanID,
		BSIID:          req.BSIID,
		BSUId:          req.BSUID,
		AdminBSIID:     &req.AdminBSIID,
		JadwalID:       jadwal.JadwalID,
	}

	if err := tx.Create(&newPengangkutan).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat sesi pengangkutan: " + err.Error()})
		return
	}

	// 4. Create riwayat_pengangkutan perdana (OTW)
	newRiwayat := models.RiwayatPengangkutan{
		PengangkutanID:     pengangkutanID,
		StatusPengangkutan: models.StatusOTW,
		ChangedAt:          now,
		ChangedBy:          req.AdminBSIID,
		Notes:              "Sesi pengangkutan dimulai",
	}

	if err := tx.Create(&newRiwayat).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencatat riwayat pengangkutan: " + err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Sesi pengangkutan berhasil dimulai",
		"data":    newPengangkutan,
	})
}

func (p *PengangkutanController) GetAllPengangkutan(c *gin.Context) {
	bankID := c.Param("bank_id")

	type response struct {
		PengangkutanID string  `json:"pengangkutan_id" gorm:"column:pengangkutan_id"`
		BSIID          string  `json:"bsi_id" gorm:"column:bsi_id"`
		BSUId          string  `json:"bsu_id" gorm:"column:bsu_id"`
		AdminBSIID     *string `json:"admin_bsi_id" gorm:"column:admin_bsi_id"`
		AdminBSUId     *string `json:"admin_bsu_id" gorm:"column:admin_bsu_id"`
		JadwalID       string  `json:"jadwal_id" gorm:"column:jadwal_id"`

		NamaBSI      string  `json:"nama_bsi" gorm:"column:nama_bsi"`
		NamaBSU      string  `json:"nama_bsu" gorm:"column:nama_bsu"`
		NamaAdminBSI *string `json:"nama_admin_bsi" gorm:"column:nama_admin_bsi"`
		NamaAdminBSU *string `json:"nama_admin_bsu" gorm:"column:nama_admin_bsu"`

		StatusPengangkutan string    `json:"status_pengangkutan" gorm:"column:status_pengangkutan"`
		ChangedAt          time.Time `json:"changed_at" gorm:"column:changed_at"`
	}

	var bank models.BankSampah
	if err := p.db.Where("bank_id = ? AND is_active = ?", bankID, true).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan"})
		return
	}

	var results []response

	query := p.db.Table("pengangkutan_sampah ps").
		Select(`
			ps.pengangkutan_id, ps.bsi_id, ps.bsu_id, ps.admin_bsi_id, ps.admin_bsu_id, ps.jadwal_id,
			bsi.nama_bank AS nama_bsi,
			bsu.nama_bank AS nama_bsu,
			u_bsi.nama AS nama_admin_bsi,
			u_bsu.nama AS nama_admin_bsu,
			rp.status_pengangkutan,
			rp.changed_at
		`).
		Joins("LEFT JOIN bank_sampah bsi ON bsi.bank_id = ps.bsi_id").
		Joins("LEFT JOIN bank_sampah bsu ON bsu.bank_id = ps.bsu_id").
		Joins("LEFT JOIN admin a_bsi ON a_bsi.admin_id = ps.admin_bsi_id").
		Joins("LEFT JOIN users u_bsi ON u_bsi.user_id = a_bsi.user_id").
		Joins("LEFT JOIN admin a_bsu ON a_bsu.admin_id = ps.admin_bsu_id").
		Joins("LEFT JOIN users u_bsu ON u_bsu.user_id = a_bsu.user_id").
		Joins(`LEFT JOIN riwayat_pengangkutan rp ON rp.riwayat_pengangkutan_id = (
			SELECT MAX(riwayat_pengangkutan_id) 
			FROM riwayat_pengangkutan 
			WHERE pengangkutan_id = ps.pengangkutan_id
		)`).
		Order("rp.changed_at DESC")

	switch bank.JenisBank {
	case models.BSU:
		query = query.Where("ps.bsu_id = ?", bankID)
	case models.BSI:
		query = query.Where("ps.bsi_id = ?", bankID)
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak: role bank tidak diizinkan"})
		return
	}

	if err := query.Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data pengangkutan: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data pengangkutan berhasil diambil",
		"data":    results,
	})
}

func (p *PengangkutanController) UpdatePengangkutanByBSI(c *gin.Context) {
	pengangkutanID := c.Param("pengangkutan_id")
	adminBSIID := c.Param("admin_bsi_id")

	type request struct {
		CurrentStatus models.StatusPengangkutan `json:"current_status" binding:"required"`
		NewStatus     models.StatusPengangkutan `json:"new_status" binding:"required"`
		Notes         string                    `json:"notes"`
	}

	var req request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Verifikasi Pengangkutan
	var pengangkutan models.PengangkutanSampah
	if err := p.db.Where("pengangkutan_id = ?", pengangkutanID).First(&pengangkutan).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data pengangkutan tidak ditemukan"})
		return
	}

	// 2. Verifikasi Kepemilikan Admin BSI
	var adminBSI models.Admin
	if err := p.db.Where("admin_id = ? AND bank_id = ?", adminBSIID, pengangkutan.BSIID).First(&adminBSI).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki akses ke pengangkutan ini"})
		return
	}

	// 3. Update AdminBSIID jika masih kosong (pertama kali di-handle oleh BSI)
	if pengangkutan.AdminBSIID == nil {
		p.db.Model(&pengangkutan).Update("admin_bsi_id", adminBSIID)
	}

	// 2. Ambil status terakhir dari DB untuk validasi konsistensi
	var lastStatus models.RiwayatPengangkutan
	if err := p.db.Where("pengangkutan_id = ?", pengangkutanID).Order("changed_at DESC").First(&lastStatus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil riwayat status"})
		return
	}

	// Pastikan status yang dikirim frontend sesuai dengan status di DB
	if lastStatus.StatusPengangkutan != req.CurrentStatus {
		c.JSON(http.StatusConflict, gin.H{
			"error":     "Status pengangkutan telah berubah. Silakan muat ulang data.",
			"db_status": lastStatus.StatusPengangkutan,
		})
		return
	}

	// 3. Validasi transisi status khusus Admin BSI
	valid := false
	switch req.CurrentStatus {
	case models.StatusRequested:
		if req.NewStatus == models.StatusApproved || req.NewStatus == models.StatusRejected {
			valid = true
		}
	case models.StatusApproved:
		if req.NewStatus == models.StatusOTW || req.NewStatus == models.StatusCanceled {
			valid = true
		}
	case models.StatusOTW:
		if req.NewStatus == models.StatusArrived || req.NewStatus == models.StatusCanceled {
			valid = true
		}
	case models.StatusArrived:
		if req.NewStatus == models.StatusCompleted || req.NewStatus == models.StatusCanceled {
			valid = true
		}
	}

	if !valid {
		c.JSON(http.StatusForbidden, gin.H{
			"error": fmt.Sprintf("Admin BSI tidak diizinkan mengubah status dari %s ke %s", req.CurrentStatus, req.NewStatus),
		})
		return
	}

	// 4. Validasi catatan (Wajib jika ditolak atau dibatalkan)
	if (req.NewStatus == models.StatusRejected || req.NewStatus == models.StatusCanceled) && req.Notes == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Catatan/Alasan diperlukan untuk pembatalan atau penolakan"})
		return
	}

	// 5. Simpan riwayat status baru
	newRiwayat := models.RiwayatPengangkutan{
		PengangkutanID:     pengangkutanID,
		StatusPengangkutan: req.NewStatus,
		ChangedAt:          time.Now(),
		ChangedBy:          adminBSIID,
		Notes:              req.Notes,
	}

	if err := p.db.Create(&newRiwayat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status pengangkutan: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Status pengangkutan berhasil diperbarui",
		"data":    newRiwayat,
	})
}

func (p *PengangkutanController) RequestPengangkutanByBSU(c *gin.Context) {
	bsuID := c.Param("bsu_id")
	adminBSUId := c.Param("admin_bsu_id")

	var req struct {
		Tanggal  string `json:"tanggal" binding:"required"`
		JamMulai string `json:"jam_mulai" binding:"required"`
		Notes    string `json:"notes" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reqDate, err := time.Parse("2006-01-02", req.Tanggal)
	if err != nil {
		reqDate, err = time.Parse(time.RFC3339, req.Tanggal)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format tanggal tidak valid. Gunakan YYYY-MM-DD atau RFC3339"})
			return
		}
	}

	// 1. Verifikasi BSU
	var bsu models.BankSampah
	if err := p.db.Where("bank_id = ? AND is_active = ? AND jenis_bank = ?", bsuID, true, models.BSU).First(&bsu).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "BSU tidak ditemukan atau tidak aktif"})
		return
	}

	if bsu.ParentBankID == nil || *bsu.ParentBankID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BSU ini tidak memiliki BSI induk"})
		return
	}
	bsiID := *bsu.ParentBankID

	// 2. Cek apakah admin_bsu_id valid? (Opsional, tapi baiknya dicek)
	var adminBSU models.Admin
	if err := p.db.Where("admin_id = ? AND bank_id = ?", adminBSUId, bsuID).First(&adminBSU).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin BSU tidak valid"})
		return
	}

	// 3. Kalkulasi Jam Selesai (tambah 2 jam, cegah error beda hari)
	startTime, errParse := time.Parse("15:04", req.JamMulai)
	if errParse != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format jam_mulai tidak valid (harap gunakan HH:mm)"})
		return
	}
	endTime := startTime.Add(2 * time.Hour)
	jamSelesai := endTime.Format("15:04")
	if endTime.Day() != startTime.Day() {
		jamSelesai = "23:59"
	}

	// 4. Cek apakah ada jadwal yang bertabrakan (tanggal sama + slot waktu overlap)
	reqDateStr := reqDate.Format("2006-01-02")
	reqHari := []models.HariEnum{
		models.Minggu, models.Senin, models.Selasa, models.Rabu, models.Kamis, models.Jumat, models.Sabtu,
	}[reqDate.Weekday()]
	mingguKe := getWeekOfMonth(reqDate)

	var jadwal models.Jadwal
	err = p.db.Where("bank_id = ? AND target_bank_id = ? AND jenis_jadwal = ? AND is_active = ? AND ("+
		"(is_rutin = true AND hari = ? AND (minggu_ke = ? OR minggu_ke = 0)) OR "+
		"(is_rutin = false AND tanggal = ?)"+
		") AND jam_mulai < ? AND jam_selesai > ?",
		bsiID, bsuID, models.JadwalPengangkutan, true, reqHari, mingguKe, reqDateStr, jamSelesai, req.JamMulai).First(&jadwal).Error

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Permintaan ditolak. Jadwal pengangkutan pada waktu tersebut bertabrakan dengan jadwal yang sudah ada.",
		})
		return
	}

	tx := p.db.Begin()

	trueVal := true
	falseVal := false

	// 5. Buat Jadwal Request (Dadakan/Khusus)
	newJadwal := models.Jadwal{
		JadwalID:          uuid.New(),
		BankID:            bsiID,
		TargetBankID:      bsuID,
		Hari:              reqHari,
		MingguKe:          mingguKe,
		JenisJadwal:       models.JadwalPengangkutan,
		JamMulai:          req.JamMulai,
		JamSelesai:        jamSelesai,
		IsActive:          &trueVal,
		IsRutin:           &falseVal,
		Tanggal:           reqDate,
		NamaJadwalSpesial: "Request Pengangkutan BSU",
		CreatedBy:         adminBSUId,
	}

	if err := tx.Create(&newJadwal).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat jadwal baru: " + err.Error()})
		return
	}

	// 6. Buat Sesi Pengangkutan
	pengangkutanID := utils.GenerateID("PGK")
	newPengangkutan := models.PengangkutanSampah{
		PengangkutanID: pengangkutanID,
		BSIID:          bsiID,
		BSUId:          bsuID,
		AdminBSIID:     nil,         // Kosong karena belum di-handle oleh BSI
		AdminBSUId:     &adminBSUId, // Set Admin BSU yang request
		JadwalID:       newJadwal.JadwalID,
	}

	if err := tx.Create(&newPengangkutan).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat sesi pengangkutan: " + err.Error()})
		return
	}

	// 7. Buat Riwayat Requested
	newRiwayat := models.RiwayatPengangkutan{
		PengangkutanID:     pengangkutanID,
		StatusPengangkutan: models.StatusRequested,
		ChangedAt:          time.Now(),
		ChangedBy:          adminBSUId,
		Notes:              req.Notes,
	}

	if err := tx.Create(&newRiwayat).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat riwayat status: " + err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
		return
	}

	go func() {
		namaBSU := bsu.NamaBank
		pgkID := pengangkutanID
		type AdminUser struct {
			UserID   string
			FCMToken string
		}
		var adminsBSI []AdminUser
		if err := p.db.Table("admin").
			Select("users.user_id, users.fcm_token").
			Joins("JOIN users ON users.user_id = admin.user_id").
			Where("admin.bank_id = ?", bsiID).
			Scan(&adminsBSI).Error; err != nil {
			return
		}
		for _, adm := range adminsBSI {
			p.notifSvc.NotifRequestPengangkutan(context.Background(), adm.UserID, adm.FCMToken, namaBSU, pgkID)
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
		"message": "Permintaan pengangkutan berhasil diajukan",
		"data":    newPengangkutan,
	})
}

func (p *PengangkutanController) PreviewPengangkutanSampah(c *gin.Context) {
	pengangkutanID := c.Param("pengangkutan_id")

	// ── 1. Parse form ─────────────────────────────────────────────────────────
	itemsStr := c.PostForm("items")
	if itemsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data items tidak boleh kosong"})
		return
	}

	type ItemRequest struct {
		SampahID string  `json:"sampah_id"`
		Qty      float64 `json:"qty"`
	}
	var items []ItemRequest
	if err := json.Unmarshal([]byte(itemsStr), &items); err != nil || len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data items tidak valid atau kosong"})
		return
	}

	// ── 2. Ambil data pengangkutan → BSU & BSI ────────────────────────────────
	var pengangkutan models.PengangkutanSampah
	if err := p.db.Where("pengangkutan_id = ?", pengangkutanID).First(&pengangkutan).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sesi pengangkutan tidak ditemukan"})
		return
	}
	bsiID := pengangkutan.BSIID
	bsuID := pengangkutan.BSUId

	// ── 3. Preview per item ───────────────────────────────────────────────────
	type ItemPreview struct {
		SampahID        string  `json:"sampah_id"`
		NamaSampah      string  `json:"nama_sampah"`
		NamaReward      string  `json:"nama_reward"`
		Qty             float64 `json:"qty"`
		StokBSUSebelum  float64 `json:"stok_bsu_sebelum"`
		StokBSUSetelah  float64 `json:"stok_bsu_setelah"`
		StokBSISebelum  float64 `json:"stok_bsi_sebelum"`
		StokBSISetelah  float64 `json:"stok_bsi_setelah"`
		CukupUntukKirim bool    `json:"cukup_untuk_kirim"`
	}

	var itemPreviews []ItemPreview
	adaStokKurang := false

	for _, item := range items {
		if item.Qty <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Qty harus lebih dari 0"})
			return
		}

		// Ambil nama sampah
		var sampah models.KatalogSampah
		if err := p.db.Preload("Reward").Where("sampah_id = ?", item.SampahID).First(&sampah).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Sampah tidak ditemukan: " + item.SampahID})
			return
		}

		// Ambil stok BSU saat ini
		var stokBSU models.StokSampah
		stokBSUSaatIni := 0.0
		if err := p.db.Where("bank_id = ? AND sampah_id = ?", bsuID, item.SampahID).
			First(&stokBSU).Error; err == nil {
			stokBSUSaatIni = stokBSU.Stok
		}

		// Ambil stok BSI saat ini
		var stokBSI models.StokSampah
		stokBSISaatIni := 0.0
		if err := p.db.Where("bank_id = ? AND sampah_id = ?", bsiID, item.SampahID).
			First(&stokBSI).Error; err == nil {
			stokBSISaatIni = stokBSI.Stok
		}

		cukup := stokBSUSaatIni >= item.Qty
		if !cukup {
			adaStokKurang = true
		}

		itemPreviews = append(itemPreviews, ItemPreview{
			SampahID:        item.SampahID,
			NamaSampah:      sampah.NamaSampah,
			NamaReward:      string(sampah.Reward.NamaReward),
			Qty:             item.Qty,
			StokBSUSebelum:  stokBSUSaatIni,
			StokBSUSetelah:  stokBSUSaatIni - item.Qty,
			StokBSISebelum:  stokBSISaatIni,
			StokBSISetelah:  stokBSISaatIni + item.Qty,
			CukupUntukKirim: cukup,
		})
	}

	var bankBSU models.BankSampah
	if err := p.db.Where("bank_id = ?", bsuID).First(&bankBSU).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data bank BSU: " + err.Error()})
		return
	}

	var bankBSI models.BankSampah
	if err := p.db.Where("bank_id = ?", bsiID).First(&bankBSI).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data bank BSI: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Preview pengangkutan berhasil dihitung",
		"data": gin.H{
			"pengangkutan_id": pengangkutanID,
			"bsi_id":          bsiID,
			"nama_bsi":        bankBSI.NamaBank,
			"bsu_id":          bsuID,
			"nama_bsu":        bankBSU.NamaBank,
			"total_item":      len(items),
			"ada_stok_kurang": adaStokKurang,
			"items":           itemPreviews,
		},
	})
}

// ─── InputSampahPengangkutan ──────────────────────────────────────────────────
// POST /pengangkutan/input/:pengangkutan_id/:admin_bsi_id/:admin_bsu_id
// Mencatat detail sampah yang diangkut dari BSU ke BSI.
// Dalam satu transaksi DB: kurangi stok BSU, tambah stok BSI, update tabungan BSU.
func (p *PengangkutanController) InputSampahPengangkutan(c *gin.Context) {
	pengangkutanID := c.Param("pengangkutan_id")
	adminBSIID := c.Param("admin_bsi_id")
	adminBSUId := c.Param("admin_bsu_id")

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal memproses form data"})
		return
	}

	itemsStr := c.PostForm("items")
	if itemsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data items tidak boleh kosong"})
		return
	}

	// Di arsitektur baru, harga/poin tidak diinput saat pengangkutan.
	// Yang dicatat hanya qty fisik. Harga akan dihitung saat penjualan (bagi hasil).
	type ItemRequest struct {
		SampahID string  `json:"sampah_id" binding:"required"`
		Qty      float64 `json:"qty" binding:"required"`
	}
	var items []ItemRequest

	importJSON := json.Unmarshal([]byte(itemsStr), &items)
	if importJSON != nil || len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data items tidak valid atau kosong"})
		return
	}

	var buktiFotoURL *string
	file, header, err := c.Request.FormFile("bukti_foto")
	if err == nil {
		defer file.Close()
		url, errUpload := p.cfStorage.UploadFile(file, header, "pengangkutan_bukti")
		if errUpload != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupload bukti foto: " + errUpload.Error()})
			return
		}
		buktiFotoURL = &url
	}

	now := time.Now()

	err = p.db.Transaction(func(tx *gorm.DB) error {
		// 0. Ambil data pengangkutan untuk dapatkan BSI & BSU
		var pengangkutan models.PengangkutanSampah
		if err := tx.Where("pengangkutan_id = ?", pengangkutanID).First(&pengangkutan).Error; err != nil {
			return fmt.Errorf("sesi pengangkutan tidak ditemukan: %w", err)
		}

		bsiID := pengangkutan.BSIID
		bsuID := pengangkutan.BSUId

		// 1. Update admin_bsi_id, admin_bsu_id, dan bukti_foto (jika ada)
		updates := map[string]interface{}{
			"admin_bsi_id": adminBSIID,
			"admin_bsu_id": adminBSUId,
		}
		if buktiFotoURL != nil {
			updates["bukti_foto"] = *buktiFotoURL
		}
		if err := tx.Model(&pengangkutan).Updates(updates).Error; err != nil {
			return fmt.Errorf("gagal update data sesi pengangkutan: %w", err)
		}

		// 2. Buat header paket_setoran_bank
		paketID := utils.GenerateID("PKT")
		paket := models.PaketSetoranBank{
			PaketID:        paketID,
			PengangkutanID: pengangkutanID,
			BSIID:          bsiID,
			BSUId:          bsuID,
			AdminBSIID:     adminBSIID,
			AdminBSUId:     adminBSUId,
			TotalItem:      len(items),
			CreatedAt:      now,
			StatusSetoran:  models.StatusBerhasil,
		}
		if err := tx.Create(&paket).Error; err != nil {
			return fmt.Errorf("gagal membuat paket setoran: %w", err)
		}

		// 3. Simpan setiap item + transfer stok BSU -> BSI + insert tabungan_sampah (FIFO)
		for _, item := range items {
			detail := models.DetailPaket{
				PaketID:  paketID,
				SampahID: item.SampahID,
				Qty:      item.Qty,
			}
			if err := tx.Create(&detail).Error; err != nil {
				return fmt.Errorf("gagal menyimpan detail item (sampah_id=%s): %w", item.SampahID, err)
			}

			// Kurangi stok BSU
			var stokBSU models.StokSampah
			resBSU := tx.Where("bank_id = ? AND sampah_id = ?", bsuID, item.SampahID).First(&stokBSU)
			if resBSU.Error != nil {
				return fmt.Errorf("stok BSU untuk sampah %s tidak ditemukan: %w", item.SampahID, resBSU.Error)
			}
			if stokBSU.Stok < item.Qty {
				return fmt.Errorf("stok BSU tidak mencukupi untuk sampah %s (stok: %g, diminta: %g)", item.SampahID, stokBSU.Stok, item.Qty)
			}
			if err := tx.Model(&stokBSU).Update("stok", gorm.Expr("stok - ?", item.Qty)).Error; err != nil {
				return fmt.Errorf("gagal mengurangi stok BSU: %w", err)
			}

			// Tambah stok BSI (upsert)
			var stokBSI models.StokSampah
			resBSI := tx.Where("bank_id = ? AND sampah_id = ?", bsiID, item.SampahID).First(&stokBSI)
			switch resBSI.Error {
			case gorm.ErrRecordNotFound:
				newStok := models.StokSampah{BankID: bsiID, SampahID: item.SampahID, Stok: item.Qty}
				if err := tx.Create(&newStok).Error; err != nil {
					return fmt.Errorf("gagal membuat stok awal BSI: %w", err)
				}
			case nil:
				if err := tx.Model(&stokBSI).Update("stok", gorm.Expr("stok + ?", item.Qty)).Error; err != nil {
					return fmt.Errorf("gagal menambah stok BSI: %w", err)
				}
			default:
				return resBSI.Error
			}
		}

		// 4. Insert tabungan_sampah per item untuk BSU (FIFO inventory bagi hasil)
		// Khusus sampah dengan reward "Sembako": skip insert tabungan_sampah,
		// karena BSU tidak mendapat bagian poin saat bagi hasil bernilai sembako.
		for _, item := range items {
			var sampahInfo models.KatalogSampah
			if err := tx.Preload("Reward").Where("sampah_id = ?", item.SampahID).First(&sampahInfo).Error; err != nil {
				return fmt.Errorf("gagal mengambil info sampah %s: %w", item.SampahID, err)
			}
			if sampahInfo.Reward.NamaReward == models.RewardEnumSembako {
				// Reward sembako: BSU tidak dapat poin dari bagi hasil, lewati tabungan
				continue
			}

			tabungan := models.TabunganSampah{
				TabunganID: utils.GenerateID("TBG"),
				BankID:     &bsuID,
				SampahID:   item.SampahID,
				Entitas:    models.EntitasBankSampah,
				Qty:        item.Qty,
				SisaQty:    item.Qty,
				CreatedAt:  now,
				SourceID:   &paketID,
			}
			if err := tx.Create(&tabungan).Error; err != nil {
				return fmt.Errorf("gagal membuat tabungan sampah BSU: %w", err)
			}
		}

		// 5. Update status_pengangkutan menjadi completed
		newRiwayat := models.RiwayatPengangkutan{
			PengangkutanID:     pengangkutanID,
			StatusPengangkutan: models.StatusCompleted,
			ChangedAt:          now,
			ChangedBy:          adminBSIID,
			Notes:              "Setoran BSU berhasil diinput dan pengangkutan selesai",
		}
		if err := tx.Create(&newRiwayat).Error; err != nil {
			return fmt.Errorf("gagal mengupdate status pengangkutan menjadi selesai: %w", err)
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ── Kirim notifikasi ke semua admin/petugas BSU (fire-and-forget) ────────────
	go func() {
		// Ambil bsiID dari pengangkutan untuk mendapatkan nama BSI
		var pengangkutanData models.PengangkutanSampah
		if err := p.db.Where("pengangkutan_id = ?", pengangkutanID).First(&pengangkutanData).Error; err != nil {
			return
		}
		var bankBSI models.BankSampah
		if err := p.db.Where("bank_id = ?", pengangkutanData.BSIID).First(&bankBSI).Error; err != nil {
			return
		}

		// Ambil semua admin BSU beserta user_id dan fcm_token
		type AdminUser struct {
			UserID   string
			FCMToken string
		}
		var adminUsers []AdminUser
		if err := p.db.Table("admin").
			Select("users.user_id, users.fcm_token").
			Joins("JOIN users ON users.user_id = admin.user_id").
			Where("admin.bank_id = ? AND admin.status_admin = ?", pengangkutanData.BSUId, models.Aktif).
			Scan(&adminUsers).Error; err != nil {
			return
		}

		for _, au := range adminUsers {
			if err := p.notifSvc.NotifPengangkutanBerhasil(
				context.Background(),
				au.UserID,
				au.FCMToken,
				len(items),
				bankBSI.NamaBank,
				pengangkutanID,
			); err != nil {
				fmt.Printf("[Notif] Gagal kirim notif pengangkutan ke user %s: %v\n", au.UserID, err)
			}
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Sampah berhasil diinput ke sesi pengangkutan",
		"total_item": len(items),
	})
}

// ─── DetailSampahPengangkutan ─────────────────────────────────────────────────
// GET /pengangkutan/detail-sampah/:pengangkutan_id
func (p *PengangkutanController) DetailSampahPengangkutan(c *gin.Context) {
	pengangkutanID := c.Param("pengangkutan_id")

	type ItemDetail struct {
		SampahID   string  `json:"sampah_id" gorm:"column:sampah_id"`
		NamaSampah string  `json:"nama_sampah" gorm:"column:nama_sampah"`
		Satuan     string  `json:"satuan" gorm:"column:satuan"`
		Qty        float64 `json:"qty" gorm:"column:qty"`
	}

	type HeaderDetail struct {
		PaketID        string    `json:"paket_id" gorm:"column:paket_id"`
		PengangkutanID string    `json:"pengangkutan_id" gorm:"column:pengangkutan_id"`
		NamaBSI        string    `json:"nama_bsi" gorm:"column:nama_bsi"`
		NamaBSU        string    `json:"nama_bsu" gorm:"column:nama_bsu"`
		NamaAdminBSI   string    `json:"nama_admin_bsi" gorm:"column:nama_admin_bsi"`
		TotalItem      int       `json:"total_item" gorm:"column:total_item"`
		StatusSetoran  string    `json:"status_setoran" gorm:"column:status_setoran"`
		CreatedAt      time.Time `json:"created_at" gorm:"column:created_at"`
		BuktiFoto      string    `json:"bukti_foto" gorm:"column:bukti_foto"`
	}

	var header HeaderDetail
	if err := p.db.Table("paket_setoran_bank psb").
		Select(`psb.paket_id, psb.pengangkutan_id, ps.bukti_foto,
			bsi.nama_bank AS nama_bsi,
			bsu.nama_bank AS nama_bsu,
			u_bsi.nama AS nama_admin_bsi,
			psb.total_item, psb.status_setoran, psb.created_at`).
		Joins("LEFT JOIN pengangkutan_sampah ps ON ps.pengangkutan_id = psb.pengangkutan_id").
		Joins("LEFT JOIN bank_sampah bsi ON bsi.bank_id = psb.bsi_id").
		Joins("LEFT JOIN bank_sampah bsu ON bsu.bank_id = psb.bsu_id").
		Joins("LEFT JOIN admin a_bsi ON a_bsi.admin_id = psb.admin_bsi_id").
		Joins("LEFT JOIN users u_bsi ON u_bsi.user_id = a_bsi.user_id").
		Where("psb.pengangkutan_id = ?", pengangkutanID).
		First(&header).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data paket pengangkutan tidak ditemukan"})
		return
	}

	var items []ItemDetail
	if err := p.db.Table("detail_paket dp").
		Select("dp.sampah_id, ks.nama_sampah, ks.satuan, dp.qty").
		Joins("LEFT JOIN katalog_sampah ks ON ks.sampah_id = dp.sampah_id").
		Where("dp.paket_id = ?", header.PaketID).
		Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail sampah: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail sampah pengangkutan berhasil diambil",
		"data": gin.H{
			"header": header,
			"items":  items,
		},
	})
}

func (p *PengangkutanController) ListSampahPengangkutan(c *gin.Context) {
	bsiID := c.Param("bsi_id")
	bsuID := c.Param("bsu_id")

	type SampahList struct {
		SampahID   string  `json:"sampah_id" gorm:"column:sampah_id"`
		NamaSampah string  `json:"nama_sampah" gorm:"column:nama_sampah"`
		FotoSampah string  `json:"foto_sampah" gorm:"column:foto_sampah"`
		Satuan     string  `json:"satuan" gorm:"column:satuan"`
		NamaReward string  `json:"nama_reward" gorm:"column:nama_reward"`
		Stok       float64 `json:"stok" gorm:"column:stok"`
	}

	var sampah []SampahList
	if err := p.db.Table("katalog_sampah ks").
		Select("ks.sampah_id, ks.nama_sampah, ks.photo_url, ks.satuan, r.nama_reward, COALESCE(ss.stok, 0) AS stok").
		Joins("LEFT JOIN stok_sampah ss ON ss.sampah_id = ks.sampah_id AND ss.bank_id = ?", bsuID).
		Joins("LEFT JOIN reward r ON r.reward_id = ks.reward_id").
		Where("ks.bank_id = ?", bsiID).
		Find(&sampah).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data sampah: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "List sampah berhasil diambil",
		"data":    sampah,
	})
}

func (p *PengangkutanController) CheckSesiActivePengangkutan(c *gin.Context) {
	bsuID := c.Param("bsu_id")

	// 1. Validasi BSU
	var bank models.BankSampah
	if err := p.db.Where("bank_id = ?", bsuID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}
	if bank.JenisBank != models.BSU {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank bukan BSU"})
		return
	}

	bsiID := bank.ParentBankID
	if bsiID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BSU tidak memiliki BSI induk"})
		return
	}

	// 2. Ambil nama BSI
	var bsi models.BankSampah
	namaBSI := ""
	if err := p.db.Where("bank_id = ?", *bsiID).First(&bsi).Error; err == nil {
		namaBSI = bsi.NamaBank
	}

	// 3. Cari sesi pengangkutan terbaru milik BSU ini
	var pengangkutan models.PengangkutanSampah
	if err := p.db.
		Where("bsu_id = ? AND bsi_id = ?", bsuID, *bsiID).
		Order("pengangkutan_id DESC").
		First(&pengangkutan).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"is_active": false,
			"bsi_id":    *bsiID,
			"nama_bsi":  namaBSI,
		})
		return
	}

	// 4. Ambil riwayat terbaru (status terkini)
	var riwayatTerbaru models.RiwayatPengangkutan
	if err := p.db.
		Where("pengangkutan_id = ?", pengangkutan.PengangkutanID).
		Order("changed_at DESC").
		First(&riwayatTerbaru).Error; err != nil {
		// Tidak ada riwayat berarti sesi baru saja dibuat, masih aktif
		c.JSON(http.StatusOK, gin.H{
			"is_active":       true,
			"pengangkutan_id": pengangkutan.PengangkutanID,
			"bsi_id":          *bsiID,
			"nama_bsi":        namaBSI,
			"status_terkini":  "",
		})
		return
	}

	// 5. Cek apakah status terkini merupakan status terminal (selesai)
	statusTerminal := riwayatTerbaru.StatusPengangkutan == models.StatusCompleted ||
		riwayatTerbaru.StatusPengangkutan == models.StatusRejected ||
		riwayatTerbaru.StatusPengangkutan == models.StatusCanceled

	c.JSON(http.StatusOK, gin.H{
		"is_active":       !statusTerminal,
		"pengangkutan_id": pengangkutan.PengangkutanID,
		"bsi_id":          *bsiID,
		"nama_bsi":        namaBSI,
		"status_terkini":  riwayatTerbaru.StatusPengangkutan,
	})
}

func (p *PengangkutanController) DetailSesiActivePengangkutan(c *gin.Context) {
	pengangkutanID := c.Param("pengangkutan_id")

	// 1. Ambil data pengangkutan beserta BSI
	var pengangkutan models.PengangkutanSampah
	if err := p.db.Where("pengangkutan_id = ?", pengangkutanID).First(&pengangkutan).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sesi pengangkutan tidak ditemukan"})
		return
	}

	// 2. Ambil nama BSI
	var bsi models.BankSampah
	namaBSI := ""
	if err := p.db.Where("bank_id = ?", pengangkutan.BSIID).First(&bsi).Error; err == nil {
		namaBSI = bsi.NamaBank
	}

	// 4. Ambil nama BSU
	var bsu models.BankSampah
	namaBSU := ""
	if err := p.db.Where("bank_id = ?", pengangkutan.BSUId).First(&bsu).Error; err == nil {
		namaBSU = bsu.NamaBank
	}

	// 3. Ambil seluruh riwayat diurutkan terbaru di atas beserta nama petugas
	type RiwayatDenganNama struct {
		models.RiwayatPengangkutan
		NamaPetugas string `gorm:"column:nama"`
	}
	var riwayat []RiwayatDenganNama
	if err := p.db.Table("riwayat_pengangkutan").
		Select("riwayat_pengangkutan.*, users.nama").
		Joins("LEFT JOIN admin ON admin.admin_id = riwayat_pengangkutan.changed_by").
		Joins("LEFT JOIN users ON users.user_id = admin.user_id").
		Where("riwayat_pengangkutan.pengangkutan_id = ?", pengangkutanID).
		Order("riwayat_pengangkutan.changed_at DESC").
		Find(&riwayat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil riwayat: " + err.Error()})
		return
	}

	type RiwayatResp struct {
		Status    models.StatusPengangkutan `json:"status"`
		ChangedAt time.Time                 `json:"changed_at"`
		ChangedBy string                    `json:"changed_by"`
		Catatan   string                    `json:"catatan"`
	}

	var riwayatResp []RiwayatResp
	var statusTerkini models.StatusPengangkutan
	for i, r := range riwayat {
		if i == 0 {
			statusTerkini = r.StatusPengangkutan
		}

		nama := r.NamaPetugas
		if nama == "" {
			nama = r.ChangedBy // fallback ke ID jika tidak ditemukan (atau jika by system)
		}

		riwayatResp = append(riwayatResp, RiwayatResp{
			Status:    r.StatusPengangkutan,
			ChangedAt: r.ChangedAt,
			ChangedBy: nama,
			Catatan:   r.Notes,
		})
	}
	if riwayatResp == nil {
		riwayatResp = []RiwayatResp{}
	}

	c.JSON(http.StatusOK, gin.H{
		"pengangkutan_id": pengangkutan.PengangkutanID,
		"bsi_id":          pengangkutan.BSIID,
		"nama_bsi":        namaBSI,
		"bsu_id":          pengangkutan.BSUId,
		"nama_bsu":        namaBSU,
		"bukti_foto":      pengangkutan.BuktiFoto,
		"status_terkini":  statusTerkini,
		"riwayat":         riwayatResp,
	})
}

func (p *PengangkutanController) GetAllActivePengangkutan(c *gin.Context) {
	bsiID := c.Param("bsi_id")
	adminID := c.Param("admin_id")

	// 1. Validasi BSI
	var bsi models.BankSampah
	if err := p.db.Where("bank_id = ?", bsiID).First(&bsi).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "BSI tidak ditemukan"})
		return
	}
	if bsi.JenisBank != models.BSI {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank bukan BSI"})
		return
	}

	// 2. Ambil semua BSU di bawah BSI ini
	var daftarBSU []models.BankSampah
	if err := p.db.Where("parent_bank_id = ? AND jenis_bank = ?", bsiID, models.BSU).Find(&daftarBSU).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data BSU: " + err.Error()})
		return
	}

	type PengangkutanAktif struct {
		PengangkutanID  string                    `json:"pengangkutan_id"`
		BsuID           string                    `json:"bsu_id"`
		NamaBsu         string                    `json:"nama_bsu"`
		StatusTerkini   models.StatusPengangkutan `json:"status_terkini"`
		IsActionAllowed bool                      `json:"is_action_allowed"`
		Tanggal         string                    `json:"tanggal,omitempty"`
		JamMulai        string                    `json:"jam_mulai,omitempty"`
		JamSelesai      string                    `json:"jam_selesai,omitempty"`
	}

	var result []PengangkutanAktif

	for _, b := range daftarBSU {
		// 3. Cari pengangkutan terbaru untuk BSU ini
		var pgk models.PengangkutanSampah
		if err := p.db.
			Where("bsu_id = ? AND bsi_id = ?", b.BankID, bsiID).
			Order("pengangkutan_id DESC").
			First(&pgk).Error; err != nil {
			// Tidak ada pengangkutan untuk BSU ini, skip
			continue
		}

		// 4. Cek status terkini dari riwayat
		var riwayatTerbaru models.RiwayatPengangkutan
		if err := p.db.
			Where("pengangkutan_id = ?", pgk.PengangkutanID).
			Order("changed_at DESC").
			First(&riwayatTerbaru).Error; err != nil {
			// Tidak ada riwayat berarti sesi baru dan belum ada perubahan status
			continue
		}

		// 5. Lewati jika sudah di status terminal (selesai)
		statusTerminal := riwayatTerbaru.StatusPengangkutan == models.StatusCompleted ||
			riwayatTerbaru.StatusPengangkutan == models.StatusRejected ||
			riwayatTerbaru.StatusPengangkutan == models.StatusCanceled
		if statusTerminal {
			continue
		}

		// 6. is_action_allowed = apakah admin ini yang menangani pengangkutan ini
		// Jika belum ada admin yang menangani (AdminBSIID == nil), maka aksi diizinkan
		isActionAllowed := pgk.AdminBSIID == nil || *pgk.AdminBSIID == adminID

		// 7. Ambil detail Jadwal
		var jadwal models.Jadwal
		p.db.Where("jadwal_id = ?", pgk.JadwalID).First(&jadwal)
		tanggalStr := ""
		if !jadwal.Tanggal.IsZero() {
			tanggalStr = jadwal.Tanggal.Format("2006-01-02")
		}

		result = append(result, PengangkutanAktif{
			PengangkutanID:  pgk.PengangkutanID,
			BsuID:           b.BankID,
			NamaBsu:         b.NamaBank,
			StatusTerkini:   riwayatTerbaru.StatusPengangkutan,
			IsActionAllowed: isActionAllowed,
			Tanggal:         tanggalStr,
			JamMulai:        jadwal.JamMulai,
			JamSelesai:      jadwal.JamSelesai,
		})
	}

	if result == nil {
		result = []PengangkutanAktif{}
	}

	c.JSON(http.StatusOK, gin.H{
		"bsi_id":   bsiID,
		"nama_bsi": bsi.NamaBank,
		"data":     result,
	})
}
