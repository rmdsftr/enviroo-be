package controllers

import (
	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"
	"encoding/json"
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
}

func NewPengangkutanController(db *gorm.DB, cfStorage *storage.CloudflareStorage) *PengangkutanController {
	return &PengangkutanController{db: db, cfStorage: cfStorage}
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
		"(is_rutin = false AND DATE(tanggal) = ?))",
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
		"(is_rutin = false AND DATE(tanggal) = ?))",
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
		PengangkutanID string `json:"pengangkutan_id" gorm:"column:pengangkutan_id"`
		BSIID          string `json:"bsi_id" gorm:"column:bsi_id"`
		BSUId          string `json:"bsu_id" gorm:"column:bsu_id"`
		AdminBSIID     *string `json:"admin_bsi_id" gorm:"column:admin_bsi_id"`
		AdminBSUId     *string `json:"admin_bsu_id" gorm:"column:admin_bsu_id"`
		JadwalID       string  `json:"jadwal_id" gorm:"column:jadwal_id"`

		NamaBSI        string  `json:"nama_bsi" gorm:"column:nama_bsi"`
		NamaBSU        string  `json:"nama_bsu" gorm:"column:nama_bsu"`
		NamaAdminBSI   *string `json:"nama_admin_bsi" gorm:"column:nama_admin_bsi"`
		NamaAdminBSU   *string `json:"nama_admin_bsu" gorm:"column:nama_admin_bsu"`

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
		"message": fmt.Sprintf("Status pengangkutan berhasil diperbarui ke %s", req.NewStatus),
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

	// 3. Cek apakah jadwal sudah ada di tanggal tersebut
	reqDateStr := reqDate.Format("2006-01-02")
	reqHari := []models.HariEnum{
		models.Minggu, models.Senin, models.Selasa, models.Rabu, models.Kamis, models.Jumat, models.Sabtu,
	}[reqDate.Weekday()]
	mingguKe := getWeekOfMonth(reqDate)

	var jadwal models.Jadwal
	err = p.db.Where("bank_id = ? AND target_bank_id = ? AND jenis_jadwal = ? AND is_active = ? AND ("+
		"(is_rutin = true AND hari = ? AND (minggu_ke = ? OR minggu_ke = 0)) OR "+
		"(is_rutin = false AND DATE(tanggal) = ?))",
		bsiID, bsuID, models.JadwalPengangkutan, true, reqHari, mingguKe, reqDateStr).First(&jadwal).Error

	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Permintaan ditolak. Sudah ada jadwal pengangkutan pada tanggal tersebut.",
		})
		return
	}

	// 4. Kalkulasi Jam Selesai (tambah 2 jam, cegah error beda hari)
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
		AdminBSIID:     nil,          // Kosong karena belum di-handle oleh BSI
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

	c.JSON(http.StatusCreated, gin.H{
		"message": "Permintaan pengangkutan berhasil diajukan",
		"data":    newPengangkutan,
	})
}

// ─── InputSampahPengangkutan ──────────────────────────────────────────────────
// POST /pengangkutan/input/:pengangkutan_id/:admin_bsi_id/:admin_bsu_id
// Mencatat detail sampah yang diangkut dari BSU ke BSI.
// Dalam satu transaksi DB: kurangi stok BSU, tambah stok BSI, update saldo kedua bank.
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

	type ItemRequest struct {
		SampahID  string  `json:"sampah_id" binding:"required"`
		Qty       float64 `json:"qty" binding:"required"`
		NilaiPoin float64 `json:"nilai_poin" binding:"required"`
	}
	var items []ItemRequest

	importJSON := json.Unmarshal([]byte(itemsStr), &items)
	if importJSON != nil || len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data items tidak valid atau kosong"})
		return
	}

	// Hitung total poin dari semua item
	totalPoin := float64(0)
	for _, item := range items {
		totalPoin += item.Qty * item.NilaiPoin
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
			TotalPoin:      totalPoin,
			CreatedAt:      now,
			StatusSetoran:  models.StatusBerhasil,
		}
		if err := tx.Create(&paket).Error; err != nil {
			return fmt.Errorf("gagal membuat paket setoran: %w", err)
		}

		// 3. Simpan setiap item + transfer stok BSU -> BSI
		for _, item := range items {
			detail := models.DetailPaket{
				PaketID:      paketID,
				SampahID:     item.SampahID,
				Qty:          item.Qty,
				NilaiPoin:    item.NilaiPoin,
				SubtotalPoin: item.Qty * item.NilaiPoin,
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

		// 4. Update saldo BSU (bertambah sebagai poin tabungan)
		var saldoBSU models.SaldoBank
		resSaldoBSU := tx.Where("bank_id = ?", bsuID).First(&saldoBSU)
		if resSaldoBSU.Error == gorm.ErrRecordNotFound {
			saldoBSU = models.SaldoBank{
				SaldoBankID:   utils.GenerateID("SLB"),
				BankID:        bsuID,
				TotalPoin:     0,
				LastUpdatedAt: now,
				LastUpdatedBy: adminBSIID,
			}
			if err := tx.Create(&saldoBSU).Error; err != nil {
				return fmt.Errorf("gagal membuat saldo BSU: %w", err)
			}
		} else if resSaldoBSU.Error != nil {
			return resSaldoBSU.Error
		}

		saldoBSUBefore := saldoBSU.TotalPoin
		saldoBSUAfter := saldoBSUBefore + totalPoin

		if err := tx.Model(&saldoBSU).Updates(map[string]interface{}{
			"total_poin":      saldoBSUAfter,
			"last_updated_at": now,
			"last_updated_by": adminBSIID,
		}).Error; err != nil {
			return fmt.Errorf("gagal update saldo BSU: %w", err)
		}

		trxBSU := models.TransaksiSaldoBank{
			TransaksiBankID: utils.GenerateID("TBK"),
			SaldoBankID:     saldoBSU.SaldoBankID,
			JenisTransaksi:  models.TransaksiPengangkutan,
			Jumlah:          totalPoin,
			SaldoSebelum:    saldoBSUBefore,
			SaldoSesudah:    saldoBSUAfter,
			UpdatedAt:       now,
		}
		if err := tx.Create(&trxBSU).Error; err != nil {
			return fmt.Errorf("gagal mencatat transaksi saldo BSU: %w", err)
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

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Sampah berhasil diinput ke sesi pengangkutan",
		"total_poin": totalPoin,
		"total_item": len(items),
	})
}

// ─── DetailSampahPengangkutan ─────────────────────────────────────────────────
// GET /pengangkutan/detail-sampah/:pengangkutan_id
func (p *PengangkutanController) DetailSampahPengangkutan(c *gin.Context) {
	pengangkutanID := c.Param("pengangkutan_id")

	type ItemDetail struct {
		SampahID     string  `json:"sampah_id" gorm:"column:sampah_id"`
		NamaSampah   string  `json:"nama_sampah" gorm:"column:nama_sampah"`
		Satuan       string  `json:"satuan" gorm:"column:satuan"`
		Qty          float64 `json:"qty" gorm:"column:qty"`
		NilaiPoin    float64 `json:"nilai_poin" gorm:"column:nilai_poin"`
		SubtotalPoin float64 `json:"subtotal_poin" gorm:"column:subtotal_poin"`
	}

	type HeaderDetail struct {
		PaketID            string    `json:"paket_id" gorm:"column:paket_id"`
		PengangkutanID     string    `json:"pengangkutan_id" gorm:"column:pengangkutan_id"`
		NamaBSI            string    `json:"nama_bsi" gorm:"column:nama_bsi"`
		NamaBSU            string    `json:"nama_bsu" gorm:"column:nama_bsu"`
		NamaAdminBSI       string    `json:"nama_admin_bsi" gorm:"column:nama_admin_bsi"`
		TotalItem          int       `json:"total_item" gorm:"column:total_item"`
		TotalPoin          float64   `json:"total_poin" gorm:"column:total_poin"`
		StatusSetoran      string    `json:"status_setoran" gorm:"column:status_setoran"`
		CreatedAt          time.Time `json:"created_at" gorm:"column:created_at"`
	}

	var header HeaderDetail
	if err := p.db.Table("paket_setoran_bank psb").
		Select(`psb.paket_id, psb.pengangkutan_id,
			bsi.nama_bank AS nama_bsi,
			bsu.nama_bank AS nama_bsu,
			u_bsi.nama AS nama_admin_bsi,
			psb.total_item, psb.total_poin, psb.status_setoran, psb.created_at`).
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
		Select("dp.sampah_id, ks.nama_sampah, ks.satuan, dp.qty, dp.nilai_poin, dp.subtotal_poin").
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
		Satuan     string  `json:"satuan" gorm:"column:satuan"`
		NilaiPoin  float64 `json:"nilai_poin" gorm:"column:poin_harga"`
		Stok       float64 `json:"stok" gorm:"column:stok"`
	}

	var sampah []SampahList
	if err := p.db.Table("katalog_sampah ks").
		Select("ks.sampah_id, ks.nama_sampah, ks.satuan, sh.poin_harga, COALESCE(ss.stok, 0) AS stok").
		Joins("INNER JOIN schema_harga_sampah sh ON sh.sampah_id = ks.sampah_id").
		Joins("LEFT JOIN stok_sampah ss ON ss.sampah_id = ks.sampah_id AND ss.bank_id = ?", bsuID).
		Where("ks.bank_id = ? AND sh.level_user = ?", bsiID, models.LevelBSU).
		Find(&sampah).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data sampah: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "List sampah berhasil diambil",
		"data":    sampah,
	})
}