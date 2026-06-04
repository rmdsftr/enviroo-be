package controllers

import (
	"context"
	"enviroo-be/internal/middleware"
	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PenarikanController struct {
	DB       *gorm.DB
	CF       *storage.CloudflareStorage
	NotifSvc services.NotifikasiService
}

func NewPenarikanController(db *gorm.DB, cf *storage.CloudflareStorage, notifSvc services.NotifikasiService) *PenarikanController {
	return &PenarikanController{DB: db, CF: cf, NotifSvc: notifSvc}
}

// POST /penarikan/preview/:nasabah_id
func (pc *PenarikanController) PreviewAjukanPenarikan(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	// ─── Validasi Nasabah ────────────────────────────────────────────────
	var nasabah models.Nasabah
	if err := pc.DB.Where("nasabah_id = ? AND status_nasabah = ?", nasabahID, models.Aktif).
		First(&nasabah).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Nasabah tidak ditemukan atau tidak aktif"})
		return
	}

	// ─── Parse Request ───────────────────────────────────────────────────
	type SembakoReq struct {
		SembakoID string  `json:"sembako_id" binding:"required"`
		Qty       float64 `json:"qty" binding:"required,gt=0"`
	}
	var req struct {
		RewardID         int          `json:"reward_id" binding:"required"`
		NominalPenarikan float64      `json:"nominal_penarikan"`
		ItemSembako      []SembakoReq `json:"item_sembako"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format request tidak valid: " + err.Error()})
		return
	}

	// ─── Validasi Reward ─────────────────────────────────────────────────
	var reward models.Reward
	if err := pc.DB.Where("reward_id = ?", req.RewardID).First(&reward).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jenis reward tidak ditemukan"})
		return
	}

	// ─── Validasi Nominal untuk non-Sembako ─────────────────────────────
	if reward.NamaReward != models.RewardEnumSembako && req.NominalPenarikan <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nominal penarikan harus lebih dari 0"})
		return
	}
	if reward.NamaReward == models.RewardEnumSembako && len(req.ItemSembako) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Item sembako harus dipilih"})
		return
	}

	// ─── Ambil Saldo Rekening ────────────────────────────────────────────
	var saldoRekening models.SaldoRekening
	if err := pc.DB.Where("nasabah_id = ? AND reward_id = ?", nasabahID, req.RewardID).
		First(&saldoRekening).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rekening saldo untuk reward ini tidak ditemukan"})
		return
	}

	// ─── Hitung Nominal & Detail Sembako ─────────────────────────────────
	finalNominal := req.NominalPenarikan

	type DetailSembakoResp struct {
		SembakoID    string  `json:"sembako_id"`
		NamaSembako  string  `json:"nama_sembako"`
		Qty          float64 `json:"qty"`
		NilaiPoin    float64 `json:"nilai_poin"`
		SubtotalPoin float64 `json:"subtotal_poin"`
	}
	var detailSembako []DetailSembakoResp

	if reward.NamaReward == models.RewardEnumSembako {
		var totalPoin float64
		for _, item := range req.ItemSembako {
			var sembako models.KatalogSembako
			if err := pc.DB.Where("sembako_id = ? AND bank_id = ?", item.SembakoID, nasabah.BankID).
				First(&sembako).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Barang sembako tidak ditemukan: " + item.SembakoID})
				return
			}
			if sembako.Stok < item.Qty {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Stok barang '" + sembako.NamaSembako + "' tidak mencukupi"})
				return
			}

			subtotal := item.Qty * sembako.NilaiPoin
			totalPoin += subtotal

			detailSembako = append(detailSembako, DetailSembakoResp{
				SembakoID:    item.SembakoID,
				NamaSembako:  sembako.NamaSembako,
				Qty:          item.Qty,
				NilaiPoin:    sembako.NilaiPoin,
				SubtotalPoin: subtotal,
			})
		}
		finalNominal = totalPoin
	}

	// ─── Kalkulasi Saldo Setelah ─────────────────────────────────────────
	saldoSetelah := saldoRekening.NominalSaldo - finalNominal

	// ─── Susun Response ──────────────────────────────────────────────────
	type PreviewResp struct {
		NasabahID        string                  `json:"nasabah_id"`
		NamaReward       models.RewardEnum       `json:"nama_reward"`
		Satuan           string                  `json:"satuan"`
		SaldoSekarang    float64                 `json:"saldo_sekarang"`
		SatuanSaldo      models.SatuanRewardEnum `json:"satuan_saldo"`
		NominalPenarikan float64                 `json:"nominal_penarikan"`
		SaldoSetelah     float64                 `json:"saldo_setelah"`
		SaldoCukup       bool                    `json:"saldo_cukup"`
		ItemSembako      []DetailSembakoResp     `json:"item_sembako,omitempty"`
	}

	resp := PreviewResp{
		NasabahID:        nasabahID,
		NamaReward:       reward.NamaReward,
		Satuan:           reward.Satuan,
		SaldoSekarang:    saldoRekening.NominalSaldo,
		SatuanSaldo:      saldoRekening.SatuanNominalSaldo,
		NominalPenarikan: finalNominal,
		SaldoSetelah:     saldoSetelah,
		SaldoCukup:       saldoSetelah >= 0,
	}

	if reward.NamaReward == models.RewardEnumSembako {
		resp.ItemSembako = detailSembako
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// POST /penarikan/:nasabah_id
func (pc *PenarikanController) AjukanPenarikan(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	// ─── Validasi Nasabah ────────────────────────────────────────────────
	var nasabah models.Nasabah
	if err := pc.DB.Where("nasabah_id = ? AND status_nasabah = ?", nasabahID, models.Aktif).First(&nasabah).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Nasabah tidak ditemukan atau tidak aktif"})
		return
	}

	// ─── Parse Request ───────────────────────────────────────────────────
	type SembakoReq struct {
		SembakoID string  `json:"sembako_id" binding:"required"`
		Qty       float64 `json:"qty" binding:"required,gt=0"`
	}

	var req struct {
		RewardID         int          `json:"reward_id" binding:"required"`
		NominalPenarikan float64      `json:"nominal_penarikan"`
		ItemSembako      []SembakoReq `json:"item_sembako"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format request tidak valid: " + err.Error()})
		return
	}

	// ─── Validasi Reward ─────────────────────────────────────────────────
	var reward models.Reward
	if err := pc.DB.Where("reward_id = ?", req.RewardID).First(&reward).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jenis reward tidak ditemukan"})
		return
	}

	// ─── Validasi Nominal untuk non-Sembako ─────────────────────────────
	if reward.NamaReward != models.RewardEnumSembako && req.NominalPenarikan <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nominal penarikan harus lebih dari 0"})
		return
	}

	// ─── Validasi awal item sembako ──────────────────────────────────────
	if reward.NamaReward == models.RewardEnumSembako && len(req.ItemSembako) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Item sembako harus dipilih"})
		return
	}

	// ─── TRANSACTION START ───────────────────────────────────────────────
	var penarikanID string
	// FIX: simpan finalNominalPenarikan di luar closure agar bisa dipakai di response
	var finalNominalPenarikanResp float64

	err := pc.DB.Transaction(func(tx *gorm.DB) error {
		// ── Lock row saldo SEBELUM dibaca (cegah race condition) ────
		var saldoRekening models.SaldoRekening
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("nasabah_id = ? AND reward_id = ?", nasabahID, req.RewardID).
			First(&saldoRekening).Error; err != nil {
			return &AppError{Code: http.StatusNotFound, Message: "Rekening saldo untuk reward ini tidak ditemukan"}
		}

		finalNominalPenarikan := req.NominalPenarikan
		var itemsToSave []models.DetailPenarikanSembako

		// ── Logik Khusus Sembako ─────────────────────────────────────────
		if reward.NamaReward == models.RewardEnumSembako {
			var totalPoin float64
			for _, item := range req.ItemSembako {
				// Lock row stok sembako untuk cegah race condition
				var sembako models.KatalogSembako
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
					Where("sembako_id = ? AND bank_id = ?", item.SembakoID, nasabah.BankID).
					First(&sembako).Error; err != nil {
					return &AppError{Code: http.StatusNotFound, Message: "Barang sembako tidak ditemukan: " + item.SembakoID}
				}

				if sembako.Stok < item.Qty {
					return &AppError{Code: http.StatusBadRequest, Message: "Stok barang '" + sembako.NamaSembako + "' tidak mencukupi"}
				}

				totalPoin += item.Qty * sembako.NilaiPoin
				itemsToSave = append(itemsToSave, models.DetailPenarikanSembako{
					SembakoID:    item.SembakoID,
					Qty:          item.Qty,
					NilaiPoin:    sembako.NilaiPoin,
					SubtotalPoin: item.Qty * sembako.NilaiPoin,
				})
			}
			finalNominalPenarikan = totalPoin
		}

		// ── Cek kecukupan saldo (di dalam transaksi, setelah lock) ───────
		if saldoRekening.NominalSaldo < finalNominalPenarikan {
			return &AppError{Code: http.StatusBadRequest, Message: "Saldo tidak mencukupi untuk melakukan penarikan"}
		}

		// 1. Potong Saldo
		saldoSebelum := saldoRekening.NominalSaldo
		saldoSesudah := saldoSebelum - finalNominalPenarikan

		if err := tx.Model(&saldoRekening).Update("nominal_saldo", saldoSesudah).Error; err != nil {
			return err
		}

		// 2. Catat Arus Saldo
		riwayatSaldo := models.RiwayatArusSaldo{
			RiwayatSaldoID: utils.GenerateID("RS"),
			RekeningID:     &saldoRekening.RekeningID,
			NominalSebelum: saldoSebelum,
			NominalSesudah: saldoSesudah,
			CreatedBy:      &nasabahID,
		}
		if err := tx.Create(&riwayatSaldo).Error; err != nil {
			return err
		}

		// 3. Buat Header Penarikan
		penarikanID = utils.GenerateID("RDM")
		now := time.Now()

		newPenarikan := models.Penarikan{
			PenarikanID:      penarikanID,
			NasabahID:        &nasabahID,
			BankID:           &nasabah.BankID,
			RewardID:         &req.RewardID,
			NominalPenarikan: finalNominalPenarikan,
			SatuanPenarikan:  models.SatuanRewardEnum(reward.Satuan),
			StatusPenarikan:  models.StatusPenarikanPending,
			KadaluarsaAt:     &[]time.Time{now.Add(2 * time.Hour)}[0],
			CreatedAt:        now,
			CreatedBy:        &nasabahID,
		}
		if err := tx.Create(&newPenarikan).Error; err != nil {
			return err
		}

		// 4. Jika Sembako: Simpan Detail & Kurangi Stok
		for _, item := range itemsToSave {
			item.PenarikanID = penarikanID
			if err := tx.Create(&item).Error; err != nil {
				return err
			}

			if err := tx.Model(&models.KatalogSembako{}).
				Where("sembako_id = ?", item.SembakoID).
				Update("stok", gorm.Expr("stok - ?", item.Qty)).Error; err != nil {
				return err
			}
		}

		// FIX: simpan nilai final agar bisa dipakai di response luar closure
		finalNominalPenarikanResp = finalNominalPenarikan
		return nil
	})

	// ─── Error Handling ──────────────────────────────────────────────────
	if err != nil {
		if appErr, ok := err.(*AppError); ok {
			c.JSON(appErr.Code, gin.H{"error": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses penarikan: " + err.Error()})
		return
	}

	// FIX: gunakan finalNominalPenarikanResp, bukan req.NominalPenarikan
	// (req.NominalPenarikan bisa 0 untuk kasus Sembako)
	c.JSON(http.StatusCreated, gin.H{
		"message": "Pengajuan penarikan berhasil disimpan",
		"data": gin.H{
			"penarikan_id": penarikanID,
			"nominal":      finalNominalPenarikanResp,
			"status":       models.StatusPenarikanPending,
		},
	})

	// ── Kirim notifikasi ke semua admin bank nasabah (fire-and-forget) ──────────
	snapshotBankID := nasabah.BankID
	snapshotPenarikanID := penarikanID
	snapshotNominal := finalNominalPenarikanResp
	snapshotSatuan := models.SatuanRewardEnum(reward.Satuan)
	snapshotNasabahID := nasabahID

	go func() {
		// Ambil nama nasabah lewat relasi user
		var nas models.Nasabah
		if err := pc.DB.Preload("User").Where("nasabah_id = ?", snapshotNasabahID).First(&nas).Error; err != nil {
			return
		}
		namaNasabah := nas.User.Nama
		pesanNilai := formatPenarikanNilai(snapshotNominal, snapshotSatuan)

		type adminUser struct {
			UserID   string
			FCMToken string
		}
		var admins []adminUser
		if err := pc.DB.Table("admin").
			Select("users.user_id, users.fcm_token").
			Joins("JOIN users ON users.user_id = admin.user_id").
			Where("admin.bank_id = ? AND admin.status_admin = ?", snapshotBankID, models.Aktif).
			Scan(&admins).Error; err != nil {
			return
		}

		for _, au := range admins {
			if err := pc.NotifSvc.NotifPengajuanPenarikan(
				context.Background(),
				au.UserID,
				au.FCMToken,
				namaNasabah,
				pesanNilai,
				snapshotPenarikanID,
			); err != nil {
				fmt.Printf("[Notif] Gagal kirim notif pengajuan penarikan ke user %s: %v\n", au.UserID, err)
			}
		}
	}()
}


// ─── AppError: untuk membawa HTTP status dari dalam transaksi ────────────────
type AppError struct {
	Code    int
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

// =============================================================================
// HANDLER: ListPenarikanNasabah
// =============================================================================
func (pc *PenarikanController) ListPenarikanNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	var filter FilterListPenarikan
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 10
	}

	query := pc.buildPenarikanQuery(pc.DB).Where("penarikan.nasabah_id = ?", nasabahID)

	// --- Terapkan filter ---
	if filter.Status != "" {
		query = query.Where("penarikan.status_penarikan = ?", filter.Status)
	}

	if filter.RewardID != nil {
		query = query.Where("penarikan.reward_id = ?", *filter.RewardID)
	}

	if filter.StartDate != "" {
		start, err := time.Parse("2006-01-02", filter.StartDate)
		if err == nil {
			query = query.Where("penarikan.created_at >= ?", start)
		}
	}

	if filter.EndDate != "" {
		end, err := time.Parse("2006-01-02", filter.EndDate)
		if err == nil {
			end = end.Add(24*time.Hour - time.Second)
			query = query.Where("penarikan.created_at <= ?", end)
		}
	}

	// FIX: pisahkan count query dari fetch query agar state query tidak saling mempengaruhi
	var total int64
	countQuery := query
	countQuery.Count(&total)

	offset := (filter.Page - 1) * filter.Limit
	var penarikanList []models.Penarikan
	if err := query.
		Order("penarikan.created_at DESC").
		Limit(filter.Limit).
		Offset(offset).
		Find(&penarikanList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal mengambil data penarikan"})
		return
	}

	summaries := make([]PenarikanSummary, 0, len(penarikanList))
	for _, p := range penarikanList {
		summaries = append(summaries, pc.mapToPenarikanSummary(p))
	}

	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, ListPenarikanResponse{
		Data:       summaries,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	})
}

func (pc *PenarikanController) ListPenarikanNasabahByBank(c *gin.Context) {
	bankIDFromPath := c.Param("bank_id")

	// --- Ambil filter dari query params ---
	var filter struct {
		FilterListPenarikan
		BankID string `form:"bank_id"`
	}
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Prioritaskan ID dari Path, jika kosong pakai dari Query
	finalBankID := bankIDFromPath
	if finalBankID == "" {
		finalBankID = filter.BankID
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	// --- Keamanan: Cek Auth & Izin ---
	claims, exists := middleware.GetClaims(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}

	// Jika bukan SuperAdmin, pastikan dia adalah Admin/Petugas dari Bank ini
	if claims.Role != models.SuperAdmin {
		var admin models.Admin
		if err := pc.DB.Where("user_id = ?", claims.UserID).First(&admin).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Data petugas tidak ditemukan"})
			return
		}

		if admin.BankID == nil || *admin.BankID != finalBankID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki akses ke data bank ini"})
			return
		}
	}

	query := pc.buildPenarikanQuery(pc.DB).Where("penarikan.bank_id = ?", finalBankID)

	// --- Terapkan filter ---
	if filter.Status != "" {
		query = query.Where("penarikan.status_penarikan = ?", filter.Status)
	}

	if filter.RewardID != nil {
		query = query.Where("penarikan.reward_id = ?", *filter.RewardID)
	}

	if filter.StartDate != "" {
		start, err := time.Parse("2006-01-02", filter.StartDate)
		if err == nil {
			query = query.Where("penarikan.created_at >= ?", start)
		}
	}

	if filter.EndDate != "" {
		end, err := time.Parse("2006-01-02", filter.EndDate)
		if err == nil {
			end = end.Add(24*time.Hour - time.Second)
			query = query.Where("penarikan.created_at <= ?", end)
		}
	}

	// FIX: pisahkan count query dari fetch query agar state query tidak saling mempengaruhi
	var total int64
	countQuery := query
	countQuery.Count(&total)

	offset := (filter.Page - 1) * filter.Limit
	var penarikanList []models.Penarikan
	if err := query.
		Order("penarikan.created_at DESC").
		Limit(filter.Limit).
		Offset(offset).
		Find(&penarikanList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal mengambil data penarikan"})
		return
	}

	summaries := make([]PenarikanSummary, 0, len(penarikanList))
	for _, p := range penarikanList {
		summaries = append(summaries, pc.mapToPenarikanSummary(p))
	}

	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, ListPenarikanResponse{
		Data:       summaries,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	})
}

// =============================================================================
// HANDLER: DetailPenarikanNasabah
// =============================================================================
func (pc *PenarikanController) DetailPenarikanNasabah(c *gin.Context) {
	penarikanID := c.Param("penarikan_id")

	// --- Ambil Claims dari middleware ---
	claims, exists := middleware.GetClaims(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}

	var penarikan models.Penarikan
	if err := pc.DB.
		Preload("Nasabah.User").
		Preload("Bank").
		Preload("Reward").
		Preload("DetailPenarikanSembako.KatalogSembako").
		Where("penarikan_id = ?", penarikanID).
		First(&penarikan).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Penarikan tidak ditemukan"})
		return
	}

	// --- Validasi Akses ---
	if claims.Role == "nasabah" {
		var nasabah models.Nasabah
		if err := pc.DB.Where("user_id = ?", claims.UserID).First(&nasabah).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Data nasabah tidak ditemukan"})
			return
		}
		if penarikan.NasabahID == nil || *penarikan.NasabahID != nasabah.NasabahID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki akses ke data ini"})
			return
		}
	} else if claims.Role != models.SuperAdmin {
		// Admin / Petugas
		var admin models.Admin
		if err := pc.DB.Where("user_id = ?", claims.UserID).First(&admin).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Data petugas tidak ditemukan"})
			return
		}
		if admin.BankID == nil || penarikan.BankID == nil || *admin.BankID != *penarikan.BankID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki akses ke data bank ini"})
			return
		}
	}

	resp := PenarikanDetail{
		PenarikanSummary: pc.mapToPenarikanSummary(penarikan),
	}

	if len(penarikan.DetailPenarikanSembako) > 0 {
		details := make([]DetailSembakoResponse, 0, len(penarikan.DetailPenarikanSembako))
		for _, d := range penarikan.DetailPenarikanSembako {
			item := DetailSembakoResponse{
				SembakoID:    d.SembakoID,
				Qty:          d.Qty,
				NilaiPoin:    d.NilaiPoin,
				SubtotalPoin: d.SubtotalPoin,
			}
			if d.KatalogSembako != nil {
				item.NamaSembako = d.KatalogSembako.NamaSembako
				item.PhotoURL = d.KatalogSembako.PhotoURL
			}
			details = append(details, item)
		}
		resp.DetailSembako = details
	}

	c.JSON(http.StatusOK, resp)
}

// =============================================================================
// HANDLER: KonfirmasiPenarikanNasabah
// =============================================================================
func (pc *PenarikanController) KonfirmasiPenarikanNasabah(c *gin.Context) {
	penarikanID := c.Param("penarikan_id")

	claims, exists := middleware.GetClaims(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}

	var admin models.Admin
	if err := pc.DB.Where("user_id = ?", claims.UserID).First(&admin).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya admin/petugas yang dapat melakukan konfirmasi"})
		return
	}
	adminID := admin.AdminID

	var req KonfirmasiPenarikanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// --- Proses Upload Foto ke Cloudflare ---
	var fotoURL string
	if req.BuktiFoto != "" {
		// Asumsi: jika diawali data:image atau base64, kita upload
		// Jika sudah URL (http), kita pakai langsung
		if len(req.BuktiFoto) > 100 { // Ciri khas base64 biasanya sangat panjang
			path := fmt.Sprintf("penarikan/%s_%d.jpg", penarikanID, time.Now().Unix())
			url, err := pc.CF.UploadBase64(req.BuktiFoto, path)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengunggah bukti foto: " + err.Error()})
				return
			}
			fotoURL = url
		} else {
			fotoURL = req.BuktiFoto
		}
	}

	err := pc.DB.Transaction(func(tx *gorm.DB) error {
		// FIX: validasi admin & bank sebelum update
		var admin models.Admin
		if err := tx.Where("admin_id = ?", adminID).First(&admin).Error; err != nil {
			return &AppError{Code: http.StatusUnauthorized, Message: "admin tidak ditemukan"}
		}
		if admin.BankID == nil {
			return &AppError{Code: http.StatusForbidden, Message: "admin tidak terhubung ke bank manapun"}
		}

		var p models.Penarikan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("penarikan_id = ?", penarikanID).First(&p).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return &AppError{Code: http.StatusNotFound, Message: "penarikan tidak ditemukan"}
			}
			return err
		}

		// FIX: validasi bank admin vs bank penarikan
		if p.BankID == nil || *p.BankID != *admin.BankID {
			return &AppError{Code: http.StatusForbidden, Message: "tidak memiliki akses untuk mengkonfirmasi penarikan ini"}
		}

		if p.NasabahID != nil {
			var nasabah models.Nasabah
			if err := tx.Where("nasabah_id = ?", *p.NasabahID).First(&nasabah).Error; err == nil {
				if nasabah.UserID == claims.UserID {
					return &AppError{Code: http.StatusForbidden, Message: "Petugas tidak boleh melayani penarikan saldo nasabah sendiri"}
				}
			}
		}

		if p.StatusPenarikan != models.StatusPenarikanPending {
			return &AppError{Code: http.StatusConflict, Message: "penarikan tidak dapat dikonfirmasi, status saat ini: " + string(p.StatusPenarikan)}
		}

		now := time.Now()
		if err := tx.Model(&p).Updates(map[string]interface{}{
			"status_penarikan": models.StatusPenarikanBerhasil,
			"bukti_foto":       fotoURL,
			"updated_at":       now,
			"updated_by":       adminID,
		}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if appErr, ok := err.(*AppError); ok {
			c.JSON(appErr.Code, gin.H{"error": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Penarikan berhasil dikonfirmasi"})

	// ── Kirim notifikasi ke nasabah (fire-and-forget) ───────────────────────────
	go func() {
		// Ambil data penarikan lengkap (nasabah_id, nominal, satuan)
		var penarikan models.Penarikan
		if err := pc.DB.Preload("Nasabah.User").Where("penarikan_id = ?", penarikanID).First(&penarikan).Error; err != nil {
			return
		}
		if penarikan.NasabahID == nil || penarikan.Nasabah == nil {
			return
		}

		pesanNilai := formatPenarikanNilai(penarikan.NominalPenarikan, penarikan.SatuanPenarikan)
		if err := pc.NotifSvc.NotifPenarikanBerhasil(
			context.Background(),
			penarikan.Nasabah.User.UserID,
			penarikan.Nasabah.User.FCMToken,
			pesanNilai,
			penarikanID,
		); err != nil {
			fmt.Printf("[Notif] Gagal kirim notif penarikan ke user %s: %v\n", penarikan.Nasabah.User.UserID, err)
		}
	}()
}

// =============================================================================
// REQUEST / RESPONSE STRUCTS
// =============================================================================

type KonfirmasiPenarikanRequest struct {
	BuktiFoto string `json:"bukti_foto" binding:"required"`
}

type FilterListPenarikan struct {
	Status    string `form:"status"`     // pending | berhasil | kadaluarsa
	RewardID  *int   `form:"reward_id"`  // ID reward (1=Uang, 2=Emas, 3=Sembako, dll)
	StartDate string `form:"start_date"` // format: 2006-01-02
	EndDate   string `form:"end_date"`   // format: 2006-01-02
	Page      int    `form:"page"`
	Limit     int    `form:"limit"`
}

type PenarikanSummary struct {
	PenarikanID      string                     `json:"penarikan_id"`
	NasabahID        *string                    `json:"nasabah_id"`
	NamaNasabah      string                     `json:"nama_nasabah"`
	BankID           *string                    `json:"bank_id"`
	NamaBank         string                     `json:"nama_bank"`
	RewardID         *int                       `json:"reward_id"`
	NamaReward       string                     `json:"nama_reward"`
	NominalPenarikan float64                    `json:"nominal_penarikan"`
	SatuanPenarikan  models.SatuanRewardEnum    `json:"satuan_penarikan"`
	StatusPenarikan  models.StatusPenarikanEnum `json:"status_penarikan"`
	KadaluarsaAt     *time.Time                 `json:"kadaluarsa_at"`
	BuktiFoto        *string                    `json:"bukti_foto"`
	CreatedAt        time.Time                  `json:"created_at"`
	UpdatedAt        time.Time                  `json:"updated_at"`
}

type PenarikanDetail struct {
	PenarikanSummary
	DetailSembako []DetailSembakoResponse `json:"detail_sembako,omitempty"`
}

type DetailSembakoResponse struct {
	SembakoID    string  `json:"sembako_id"`
	NamaSembako  string  `json:"nama_sembako"`
	PhotoURL     *string `json:"photo_url"`
	Qty          float64 `json:"qty"`
	NilaiPoin    float64 `json:"nilai_poin"`
	SubtotalPoin float64 `json:"subtotal_poin"`
}

type ListPenarikanResponse struct {
	Data       []PenarikanSummary `json:"data"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"total_pages"`
}

// =============================================================================
// HELPERS
// =============================================================================

// getPetugasAdminID mengambil admin_id dari context yang di-set oleh middleware auth.
func getPetugasAdminID(c *gin.Context) (string, bool) {
	adminID, exists := c.Get("admin_id")
	if !exists {
		return "", false
	}
	id, ok := adminID.(string)
	return id, ok
}

func (pc *PenarikanController) mapToPenarikanSummary(p models.Penarikan) PenarikanSummary {
	s := PenarikanSummary{
		PenarikanID:      p.PenarikanID,
		NasabahID:        p.NasabahID,
		BankID:           p.BankID,
		RewardID:         p.RewardID,
		NominalPenarikan: p.NominalPenarikan,
		SatuanPenarikan:  p.SatuanPenarikan,
		StatusPenarikan:  p.StatusPenarikan,
		KadaluarsaAt:     p.KadaluarsaAt,
		BuktiFoto:        p.BuktiFoto,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}

	if p.Nasabah != nil && p.Nasabah.User.UserID != "" {
		s.NamaNasabah = p.Nasabah.User.Nama
	}
	if p.Bank != nil {
		s.NamaBank = p.Bank.NamaBank
	}
	if p.Reward != nil {
		s.NamaReward = string(p.Reward.NamaReward)
	}

	return s
}

// buildPenarikanQuery membangun query dasar dengan preload relasi yang dibutuhkan.
// FIX: hapus kode orphan yang nyangkut di dalam method ini
func (pc *PenarikanController) buildPenarikanQuery(db *gorm.DB) *gorm.DB {
	return db.Model(&models.Penarikan{}).
		Preload("Nasabah.User").
		Preload("Bank").
		Preload("Reward")
}

// BatalPenarikan membatalkan pengajuan penarikan yang masih pending
// PATCH /penarikan/batal/:penarikan_id
func (pc *PenarikanController) BatalPenarikan(c *gin.Context) {
	penarikanID := c.Param("penarikan_id")

	// 1. Ambil claims dari middleware
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak ditemukan"})
		return
	}

	// 2. Cari NasabahID berdasarkan UserID dari token
	var nasabah models.Nasabah
	if err := pc.DB.Where("user_id = ?", claims.UserID).First(&nasabah).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Hanya nasabah yang dapat membatalkan pengajuan"})
		return
	}
	nasabahID := nasabah.NasabahID

	err := pc.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Lock & ambil data penarikan
		var p models.Penarikan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Reward").
			Where("penarikan_id = ?", penarikanID).First(&p).Error; err != nil {
			return &AppError{Code: http.StatusNotFound, Message: "Pengajuan penarikan tidak ditemukan"}
		}

		// 2. Validasi kepemilikan
		if p.NasabahID == nil || *p.NasabahID != nasabahID {
			return &AppError{Code: http.StatusForbidden, Message: "Anda tidak memiliki akses untuk membatalkan pengajuan ini"}
		}

		// 3. Validasi status (hanya bisa batal jika masih pending)
		if p.StatusPenarikan != models.StatusPenarikanPending {
			return &AppError{Code: http.StatusBadRequest, Message: "Pengajuan tidak dapat dibatalkan (sudah diproses/kadaluarsa/batal)"}
		}

		// 4. Update status jadi dibatalkan
		now := time.Now()
		if err := tx.Model(&p).Updates(map[string]interface{}{
			"status_penarikan": models.StatusPenarikanDibatalkan,
			"updated_at":       now,
			"updated_by":       &nasabahID,
		}).Error; err != nil {
			return err
		}

		// 5. Kembalikan Saldo (Refund)
		var saldo models.SaldoRekening
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("nasabah_id = ? AND reward_id = ?", nasabahID, p.RewardID).
			First(&saldo).Error; err != nil {
			return err
		}

		saldoSebelum := saldo.NominalSaldo
		saldoSesudah := saldoSebelum + p.NominalPenarikan

		if err := tx.Model(&saldo).Update("nominal_saldo", saldoSesudah).Error; err != nil {
			return err
		}

		// 6. Catat Arus Saldo (Refund)
		riwayat := models.RiwayatArusSaldo{
			RiwayatSaldoID: utils.GenerateID("RS"),
			RekeningID:     &saldo.RekeningID,
			NominalSebelum: saldoSebelum,
			NominalSesudah: saldoSesudah,
			CreatedBy:      &nasabahID,
		}
		if err := tx.Create(&riwayat).Error; err != nil {
			return err
		}

		// 7. Jika sembako: Kembalikan Stok
		if p.Reward != nil && p.Reward.NamaReward == models.RewardEnumSembako {
			var details []models.DetailPenarikanSembako
			if err := tx.Where("penarikan_id = ?", p.PenarikanID).Find(&details).Error; err != nil {
				return err
			}

			for _, d := range details {
				if err := tx.Model(&models.KatalogSembako{}).
					Where("sembako_id = ?", d.SembakoID).
					Update("stok", gorm.Expr("stok + ?", d.Qty)).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		if appErr, ok := err.(*AppError); ok {
			c.JSON(appErr.Code, gin.H{"error": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membatalkan penarikan: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pengajuan penarikan berhasil dibatalkan dan saldo telah dikembalikan"})
}

// formatPenarikanNilai memformat nilai penarikan sesuai satuannya.
// - Uang   : Rp1.500.000  (satuan di depan)
// - Emas   : 5.5 gram     (satuan di belakang)
// - Sembako: 1500 poin    (satuan di belakang)
func formatPenarikanNilai(nominal float64, satuan models.SatuanRewardEnum) string {
	switch satuan {
	case models.SatuanRewardEnumRp:
		return fmt.Sprintf("Rp%.0f", nominal)
	case models.SatuanRewardEnumPoin:
		return fmt.Sprintf("%.0f poin", nominal)
	default:
		return fmt.Sprintf("%.2f %s", nominal, satuan)
	}
}

