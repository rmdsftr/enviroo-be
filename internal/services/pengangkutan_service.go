package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/pkg/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Repo interface ─────────────────────────────────────────────────────────

type PengangkutanRepoIface interface {
	GetJadwalHariIni(bsiID string, todayHari models.HariEnum, mingguKe int, todayDate string) ([]repositories.ListJadwalRow, error)
	FindBank(bankID string) (*models.BankSampah, error)
	FindBankAktif(bankID string) (*models.BankSampah, error)
	FindBSUAktifByParent(bsuID, bsiID string) (*models.BankSampah, error)
	FindJadwalStart(bsiID, bsuID string, todayHari models.HariEnum, mingguKe int, todayDate string) (*models.Jadwal, error)
	GetListByBSI(bsiID, startDate, endDate string) ([]repositories.PengangkutanListRow, error)
	GetListByBSU(bsuID, startDate, endDate string) ([]repositories.PengangkutanListRow, error)
	FindPengangkutan(pengangkutanID string) (*models.PengangkutanSampah, error)
	FindAdminByBank(adminID, bankID string) (*models.Admin, error)
	FindLastRiwayat(db *gorm.DB, pengangkutanID string) (*models.RiwayatPengangkutan, error)
	SetAdminBSI(db *gorm.DB, pgk *models.PengangkutanSampah, adminID string) error
	CheckJadwalConflict(bsiID, bsuID string, reqHari models.HariEnum, mingguKe int, reqDateStr, jamMulai, jamSelesai string) (bool, error)
	FindLatestPengangkutan(bsuID, bsiID string) (*models.PengangkutanSampah, error)
	FindBSUList(bsiID string) ([]models.BankSampah, error)
	FindJadwal(jadwalID uuid.UUID) (*models.Jadwal, error)
	GetDetailHeader(pengangkutanID string) (*repositories.PengangkutanDetailHeader, error)
	GetDetailItems(pengangkutanID string) ([]repositories.PengangkutanDetailItem, error)
	GetSampahList(bsiID, bsuID string) ([]repositories.SampahPengangkutanRow, error)
	GetRiwayatDetail(pengangkutanID string) ([]repositories.RiwayatDetailRow, error)
	FindAdminsByBank(bankID string) ([]repositories.PengangkutanAdminUser, error)
	FindAdminsBankAktif(bankID string) ([]repositories.PengangkutanAdminUser, error)
	FindStok(bankID, sampahID string) (float64, error)
	FindKatalogWithAll(sampahID string) (*models.KatalogSampah, error)
	// Tx methods
	CreatePengangkutan(tx *gorm.DB, p *models.PengangkutanSampah) error
	CreateRiwayat(tx *gorm.DB, rw *models.RiwayatPengangkutan) error
	CreateJadwal(tx *gorm.DB, j *models.Jadwal) error
	UpdatePengangkutanMeta(tx *gorm.DB, pengangkutanID string, updates map[string]interface{}) error
	CreateDetailPengangkutan(tx *gorm.DB, d *models.DetailPengangkutan) error
	DecrStokBSU(tx *gorm.DB, bankID, sampahID string, qty float64) error
	UpsertStokBSI(tx *gorm.DB, bankID, sampahID string, qty float64) error
	FindKatalogTx(tx *gorm.DB, sampahID string) (*models.KatalogSampah, error)
	CreateTabunganSampah(tx *gorm.DB, t *models.TabunganSampah) error
}

// ── Request / Response types ───────────────────────────────────────────────

type StartSesiReq struct {
	BSIID      string
	BSUID      string
	AdminBSIID string
	IsMandiri  bool
	JadwalID   string
}

type UpdateStatusReq struct {
	CurrentStatus models.StatusPengangkutan
	NewStatus     models.StatusPengangkutan
	Notes         string
}

type RequestBSUReq struct {
	Tanggal  string
	JamMulai string
	Notes    string
}

type ItemPengangkutanReq struct {
	SampahID string  `json:"sampah_id"`
	Qty      float64 `json:"qty"`
}

type ItemPreviewPengangkutan struct {
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

type PreviewPengangkutanResult struct {
	PengangkutanID string                    `json:"pengangkutan_id"`
	BSIID          string                    `json:"bsi_id"`
	NamaBSI        string                    `json:"nama_bsi"`
	BSUID          string                    `json:"bsu_id"`
	NamaBSU        string                    `json:"nama_bsu"`
	TotalItem      int                       `json:"total_item"`
	AdaStokKurang  bool                      `json:"ada_stok_kurang"`
	Items          []ItemPreviewPengangkutan `json:"items"`
}

type PengangkutanDetail struct {
	Header repositories.PengangkutanDetailHeader  `json:"header"`
	Items  []repositories.PengangkutanDetailItem  `json:"items"`
}

type SesiAktifResult struct {
	IsActive       bool   `json:"is_active"`
	PengangkutanID string `json:"pengangkutan_id,omitempty"`
	BSIID          string `json:"bsi_id"`
	NamaBSI        string `json:"nama_bsi"`
	StatusTerkini  string `json:"status_terkini,omitempty"`
}

type RiwayatPengangkutanResp struct {
	Status    models.StatusPengangkutan `json:"status"`
	ChangedAt time.Time                 `json:"changed_at"`
	ChangedBy string                    `json:"changed_by"`
	Catatan   string                    `json:"catatan"`
}

type DetailSesiAktifResult struct {
	PengangkutanID string                    `json:"pengangkutan_id"`
	BSIID          string                    `json:"bsi_id"`
	NamaBSI        string                    `json:"nama_bsi"`
	BSUID          string                    `json:"bsu_id"`
	NamaBSU        string                    `json:"nama_bsu"`
	BuktiFoto      *string                   `json:"bukti_foto"`
	StatusTerkini  models.StatusPengangkutan `json:"status_terkini"`
	Riwayat        []RiwayatPengangkutanResp `json:"riwayat"`
}

type PengangkutanAktifItem struct {
	PengangkutanID  string                    `json:"pengangkutan_id"`
	BsuID           string                    `json:"bsu_id"`
	NamaBsu         string                    `json:"nama_bsu"`
	StatusTerkini   models.StatusPengangkutan `json:"status_terkini"`
	IsActionAllowed bool                      `json:"is_action_allowed"`
	Tanggal         string                    `json:"tanggal,omitempty"`
	JamMulai        string                    `json:"jam_mulai,omitempty"`
	JamSelesai      string                    `json:"jam_selesai,omitempty"`
}

type AllAktifResult struct {
	BSIID   string                  `json:"bsi_id"`
	NamaBSI string                  `json:"nama_bsi"`
	Data    []PengangkutanAktifItem `json:"data"`
}

// ── Service interface ──────────────────────────────────────────────────────

type PengangkutanService interface {
	GetJadwalHariIni(bsiID string) ([]repositories.ListJadwalRow, error)
	StartSesi(req StartSesiReq) (*models.PengangkutanSampah, error)
	GetList(bankID, startDate, endDate string) ([]repositories.PengangkutanListRow, error)
	UpdateStatus(pengangkutanID, adminBSIID string, req UpdateStatusReq) (*models.RiwayatPengangkutan, error)
	RequestByBSU(bsuID, adminBSUID string, req RequestBSUReq) (*models.PengangkutanSampah, error)
	Preview(pengangkutanID string, items []ItemPengangkutanReq) (*PreviewPengangkutanResult, error)
	Input(pengangkutanID, adminBSIID, adminBSUID string, buktiURL *string, items []ItemPengangkutanReq) error
	GetDetailSampah(pengangkutanID string) (*PengangkutanDetail, error)
	GetSampahList(bsiID, bsuID string) ([]repositories.SampahPengangkutanRow, error)
	CheckSesiAktif(bsuID string) (*SesiAktifResult, error)
	GetDetailSesiAktif(pengangkutanID string) (*DetailSesiAktifResult, error)
	GetAllAktif(bsiID, adminID string) (*AllAktifResult, error)
}

// ── Implementation ─────────────────────────────────────────────────────────

type pengangkutanService struct {
	db       *gorm.DB
	repo     PengangkutanRepoIface
	notifSvc NotifikasiService
}

func NewPengangkutanService(db *gorm.DB, repo PengangkutanRepoIface, notifSvc NotifikasiService) PengangkutanService {
	return &pengangkutanService{db: db, repo: repo, notifSvc: notifSvc}
}

func (s *pengangkutanService) GetJadwalHariIni(bsiID string) ([]repositories.ListJadwalRow, error) {
	now := time.Now()
	rows, err := s.repo.GetJadwalHariIni(bsiID, hariOf(now), weekOfMonth(now), now.Format("2006-01-02"))
	if err != nil {
		return nil, errInternal("Gagal mengambil jadwal pengangkutan: " + err.Error())
	}
	if rows == nil {
		rows = []repositories.ListJadwalRow{}
	}
	return rows, nil
}

func (s *pengangkutanService) StartSesi(req StartSesiReq) (*models.PengangkutanSampah, error) {
	if _, err := s.repo.FindBankAktif(req.BSIID); err != nil {
		return nil, errBadRequest("BSI tidak ditemukan atau tidak aktif")
	}
	if _, err := s.repo.FindBSUAktifByParent(req.BSUID, req.BSIID); err != nil {
		return nil, errBadRequest("BSU tidak ditemukan atau tidak aktif")
	}

	now := time.Now()
	jadwal, err := s.repo.FindJadwalStart(req.BSIID, req.BSUID, hariOf(now), weekOfMonth(now), now.Format("2006-01-02"))
	if err != nil {
		return nil, errNotFound("Jadwal tidak cocok dengan jadwal rutin ataupun khusus")
	}
	if req.JadwalID != jadwal.JadwalID.String() {
		return nil, errBadRequest("jadwal_id tidak cocok dengan jadwal hari ini")
	}

	pengangkutanID := utils.GenerateID("PGK")
	initStatus := models.StatusOTW
	initNotes := "Sesi pengangkutan dimulai"
	if req.IsMandiri {
		initStatus = models.StatusArrived
		initNotes = "BSU mengantar sendiri, langsung tiba di BSI"
	}

	pgk := &models.PengangkutanSampah{
		PengangkutanID: pengangkutanID,
		BSIID:          req.BSIID,
		BSUId:          req.BSUID,
		AdminBSIID:     &req.AdminBSIID,
		JadwalID:       jadwal.JadwalID,
		IsMandiri:      req.IsMandiri,
	}
	riwayat := &models.RiwayatPengangkutan{
		PengangkutanID:     pengangkutanID,
		StatusPengangkutan: initStatus,
		ChangedAt:          now,
		ChangedBy:          req.AdminBSIID,
		Notes:              initNotes,
	}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.CreatePengangkutan(tx, pgk); err != nil {
			return errInternal("Gagal membuat sesi pengangkutan: " + err.Error())
		}
		if err := s.repo.CreateRiwayat(tx, riwayat); err != nil {
			return errInternal("Gagal mencatat riwayat pengangkutan: " + err.Error())
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return pgk, nil
}

func (s *pengangkutanService) GetList(bankID, startDate, endDate string) ([]repositories.PengangkutanListRow, error) {
	bank, err := s.repo.FindBankAktif(bankID)
	if err != nil {
		return nil, errNotFound("Bank sampah tidak ditemukan")
	}

	switch bank.JenisBank {
	case models.BSU:
		return s.repo.GetListByBSU(bankID, startDate, endDate)
	case models.BSI:
		return s.repo.GetListByBSI(bankID, startDate, endDate)
	default:
		return nil, errForbidden("Akses ditolak: role bank tidak diizinkan")
	}
}

func (s *pengangkutanService) UpdateStatus(pengangkutanID, adminBSIID string, req UpdateStatusReq) (*models.RiwayatPengangkutan, error) {
	pgk, err := s.repo.FindPengangkutan(pengangkutanID)
	if err != nil {
		return nil, errNotFound("Data pengangkutan tidak ditemukan")
	}

	if _, err := s.repo.FindAdminByBank(adminBSIID, pgk.BSIID); err != nil {
		return nil, errForbidden("Anda tidak memiliki akses ke pengangkutan ini")
	}

	if pgk.AdminBSIID == nil {
		_ = s.repo.SetAdminBSI(s.db, pgk, adminBSIID)
	}

	lastRiwayat, err := s.repo.FindLastRiwayat(s.db, pengangkutanID)
	if err != nil {
		return nil, errInternal("Gagal mengambil riwayat status")
	}

	if lastRiwayat.StatusPengangkutan != req.CurrentStatus {
		return nil, &ServiceError{
			Code:    409,
			Message: "Status pengangkutan telah berubah. Silakan muat ulang data.",
			Data:    map[string]interface{}{"db_status": lastRiwayat.StatusPengangkutan},
		}
	}

	if !validTransitionBSI(req.CurrentStatus, req.NewStatus) {
		return nil, errForbidden(fmt.Sprintf("Admin BSI tidak diizinkan mengubah status dari %s ke %s", req.CurrentStatus, req.NewStatus))
	}

	if (req.NewStatus == models.StatusRejected || req.NewStatus == models.StatusCanceled) && req.Notes == "" {
		return nil, errBadRequest("Catatan/Alasan diperlukan untuk pembatalan atau penolakan")
	}

	newRiwayat := &models.RiwayatPengangkutan{
		PengangkutanID:     pengangkutanID,
		StatusPengangkutan: req.NewStatus,
		ChangedAt:          time.Now(),
		ChangedBy:          adminBSIID,
		Notes:              req.Notes,
	}
	if err := s.repo.CreateRiwayat(s.db, newRiwayat); err != nil {
		return nil, errInternal("Gagal memperbarui status pengangkutan: " + err.Error())
	}
	return newRiwayat, nil
}

func (s *pengangkutanService) RequestByBSU(bsuID, adminBSUID string, req RequestBSUReq) (*models.PengangkutanSampah, error) {
	reqDate, err := parseFlexibleDate(req.Tanggal)
	if err != nil {
		return nil, errBadRequest("Format tanggal tidak valid. Gunakan YYYY-MM-DD atau RFC3339")
	}

	bsu, err := s.repo.FindBank(bsuID)
	if err != nil || bsu.JenisBank != models.BSU || !bsu.IsActive {
		return nil, errNotFound("BSU tidak ditemukan atau tidak aktif")
	}
	if bsu.ParentBankID == nil || *bsu.ParentBankID == "" {
		return nil, errBadRequest("BSU ini tidak memiliki BSI induk")
	}
	bsiID := *bsu.ParentBankID

	if _, err := s.repo.FindAdminByBank(adminBSUID, bsuID); err != nil {
		return nil, errForbidden("Admin BSU tidak valid")
	}

	startTime, errParse := time.Parse("15:04", req.JamMulai)
	if errParse != nil {
		return nil, errBadRequest("Format jam_mulai tidak valid (harap gunakan HH:mm)")
	}
	endTime := startTime.Add(2 * time.Hour)
	jamSelesai := endTime.Format("15:04")
	if endTime.Day() != startTime.Day() {
		jamSelesai = "23:59"
	}

	reqDateStr := reqDate.Format("2006-01-02")
	reqHari := hariOf(reqDate)
	mingguKe := weekOfMonth(reqDate)

	conflict, err := s.repo.CheckJadwalConflict(bsiID, bsuID, reqHari, mingguKe, reqDateStr, req.JamMulai, jamSelesai)
	if err != nil {
		return nil, errInternal("Gagal memeriksa konflik jadwal")
	}
	if conflict {
		return nil, errConflict("Permintaan ditolak. Jadwal pengangkutan pada waktu tersebut bertabrakan dengan jadwal yang sudah ada.", nil)
	}

	pengangkutanID := utils.GenerateID("PGK")
	trueVal, falseVal := true, false

	newJadwal := &models.Jadwal{
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
		CreatedBy:         adminBSUID,
	}
	pgk := &models.PengangkutanSampah{
		PengangkutanID: pengangkutanID,
		BSIID:          bsiID,
		BSUId:          bsuID,
		AdminBSIID:     nil,
		AdminBSUId:     &adminBSUID,
	}
	riwayat := &models.RiwayatPengangkutan{
		PengangkutanID:     pengangkutanID,
		StatusPengangkutan: models.StatusRequested,
		ChangedAt:          time.Now(),
		ChangedBy:          adminBSUID,
		Notes:              req.Notes,
	}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.CreateJadwal(tx, newJadwal); err != nil {
			return errInternal("Gagal membuat jadwal baru: " + err.Error())
		}
		pgk.JadwalID = newJadwal.JadwalID
		if err := s.repo.CreatePengangkutan(tx, pgk); err != nil {
			return errInternal("Gagal membuat sesi pengangkutan: " + err.Error())
		}
		if err := s.repo.CreateRiwayat(tx, riwayat); err != nil {
			return errInternal("Gagal membuat riwayat status: " + err.Error())
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	go func() {
		admins, err := s.repo.FindAdminsByBank(bsiID)
		if err != nil {
			log.Printf("[Notif] Gagal ambil admin BSI %s: %v", bsiID, err)
			return
		}
		for _, a := range admins {
			_ = s.notifSvc.NotifRequestPengangkutan(context.Background(), a.UserID, a.FCMToken, bsu.NamaBank, pengangkutanID)
		}
	}()

	return pgk, nil
}

func (s *pengangkutanService) Preview(pengangkutanID string, items []ItemPengangkutanReq) (*PreviewPengangkutanResult, error) {
	pgk, err := s.repo.FindPengangkutan(pengangkutanID)
	if err != nil {
		return nil, errNotFound("Sesi pengangkutan tidak ditemukan")
	}

	bankBSI, err := s.repo.FindBank(pgk.BSIID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data bank BSI")
	}
	bankBSU, err := s.repo.FindBank(pgk.BSUId)
	if err != nil {
		return nil, errInternal("Gagal mengambil data bank BSU")
	}

	adaStokKurang := false
	var previews []ItemPreviewPengangkutan

	for _, item := range items {
		if item.Qty <= 0 {
			return nil, errBadRequest("Qty harus lebih dari 0")
		}
		katalog, err := s.repo.FindKatalogWithAll(item.SampahID)
		if err != nil {
			return nil, errBadRequest("Sampah tidak ditemukan: " + item.SampahID)
		}

		stokBSU, err := s.repo.FindStok(pgk.BSUId, item.SampahID)
		if err != nil {
			return nil, errInternal("Gagal membaca stok BSU")
		}
		stokBSI, err := s.repo.FindStok(pgk.BSIID, item.SampahID)
		if err != nil {
			return nil, errInternal("Gagal membaca stok BSI")
		}

		cukup := stokBSU >= item.Qty
		if !cukup {
			adaStokKurang = true
		}
		previews = append(previews, ItemPreviewPengangkutan{
			SampahID:        item.SampahID,
			NamaSampah:      katalog.Sarok.NamaSampah,
			NamaReward:      string(katalog.Reward.NamaReward),
			Qty:             item.Qty,
			StokBSUSebelum:  stokBSU,
			StokBSUSetelah:  stokBSU - item.Qty,
			StokBSISebelum:  stokBSI,
			StokBSISetelah:  stokBSI + item.Qty,
			CukupUntukKirim: cukup,
		})
	}

	return &PreviewPengangkutanResult{
		PengangkutanID: pengangkutanID,
		BSIID:          pgk.BSIID,
		NamaBSI:        bankBSI.NamaBank,
		BSUID:          pgk.BSUId,
		NamaBSU:        bankBSU.NamaBank,
		TotalItem:      len(items),
		AdaStokKurang:  adaStokKurang,
		Items:          previews,
	}, nil
}

func (s *pengangkutanService) Input(pengangkutanID, adminBSIID, adminBSUID string, buktiURL *string, items []ItemPengangkutanReq) error {
	now := time.Now()
	totalItem := len(items)

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		var pgk models.PengangkutanSampah
		if err := tx.Where("pengangkutan_id = ?", pengangkutanID).First(&pgk).Error; err != nil {
			return errNotFound("Sesi pengangkutan tidak ditemukan")
		}

		updates := map[string]interface{}{
			"admin_bsi_id": adminBSIID,
			"admin_bsu_id": adminBSUID,
			"total_item":   totalItem,
		}
		if buktiURL != nil {
			updates["bukti_foto"] = *buktiURL
		}
		if err := s.repo.UpdatePengangkutanMeta(tx, pengangkutanID, updates); err != nil {
			return errInternal("Gagal update data sesi pengangkutan")
		}

		for _, item := range items {
			detail := &models.DetailPengangkutan{
				PengangkutanID: pengangkutanID,
				SampahID:       item.SampahID,
				Qty:            item.Qty,
			}
			if err := s.repo.CreateDetailPengangkutan(tx, detail); err != nil {
				return errInternal(fmt.Sprintf("Gagal menyimpan detail item (sampah_id=%s)", item.SampahID))
			}
			if err := s.repo.DecrStokBSU(tx, pgk.BSUId, item.SampahID, item.Qty); err != nil {
				return &ServiceError{Code: 422, Message: err.Error()}
			}
			if err := s.repo.UpsertStokBSI(tx, pgk.BSIID, item.SampahID, item.Qty); err != nil {
				return errInternal("Gagal menambah stok BSI")
			}
		}

		// Tabungan BSU (FIFO) — skip sampah bereward sembako
		for _, item := range items {
			katalog, err := s.repo.FindKatalogTx(tx, item.SampahID)
			if err != nil {
				return errInternal(fmt.Sprintf("Gagal mengambil info sampah %s", item.SampahID))
			}
			if katalog.Reward.NamaReward == models.RewardEnumSembako {
				continue
			}
			tabungan := &models.TabunganSampah{
				TabunganID: utils.GenerateID("TBG"),
				BankID:     &pgk.BSUId,
				SampahID:   item.SampahID,
				Entitas:    models.EntitasBankSampah,
				Qty:        item.Qty,
				SisaQty:    item.Qty,
				CreatedAt:  now,
				SourceID:   &pengangkutanID,
			}
			if err := s.repo.CreateTabunganSampah(tx, tabungan); err != nil {
				return errInternal("Gagal membuat tabungan sampah BSU")
			}
		}

		newRiwayat := &models.RiwayatPengangkutan{
			PengangkutanID:     pengangkutanID,
			StatusPengangkutan: models.StatusCompleted,
			ChangedAt:          now,
			ChangedBy:          adminBSIID,
			Notes:              "Setoran BSU berhasil diinput dan pengangkutan selesai",
		}
		if err := s.repo.CreateRiwayat(tx, newRiwayat); err != nil {
			return errInternal("Gagal mengupdate status pengangkutan menjadi selesai")
		}

		return nil
	})
	if txErr != nil {
		return txErr
	}

	go func() {
		pgk, err := s.repo.FindPengangkutan(pengangkutanID)
		if err != nil {
			log.Printf("[Notif] Gagal ambil pengangkutan %s: %v", pengangkutanID, err)
			return
		}
		bankBSI, err := s.repo.FindBank(pgk.BSIID)
		if err != nil {
			log.Printf("[Notif] Gagal ambil BSI %s: %v", pgk.BSIID, err)
			return
		}
		admins, err := s.repo.FindAdminsBankAktif(pgk.BSUId)
		if err != nil {
			log.Printf("[Notif] Gagal ambil admin BSU %s: %v", pgk.BSUId, err)
			return
		}
		for _, a := range admins {
			if err := s.notifSvc.NotifPengangkutanBerhasil(context.Background(), a.UserID, a.FCMToken, totalItem, bankBSI.NamaBank, pengangkutanID); err != nil {
				log.Printf("[Notif] Gagal kirim notif pengangkutan ke %s: %v", a.UserID, err)
			}
		}
	}()

	return nil
}

func (s *pengangkutanService) GetDetailSampah(pengangkutanID string) (*PengangkutanDetail, error) {
	header, err := s.repo.GetDetailHeader(pengangkutanID)
	if err != nil {
		return nil, errNotFound("Data pengangkutan tidak ditemukan")
	}
	items, err := s.repo.GetDetailItems(pengangkutanID)
	if err != nil {
		return nil, errInternal("Gagal mengambil detail sampah")
	}
	if items == nil {
		items = []repositories.PengangkutanDetailItem{}
	}
	return &PengangkutanDetail{Header: *header, Items: items}, nil
}

func (s *pengangkutanService) GetSampahList(bsiID, bsuID string) ([]repositories.SampahPengangkutanRow, error) {
	rows, err := s.repo.GetSampahList(bsiID, bsuID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data sampah")
	}
	return rows, nil
}

func (s *pengangkutanService) CheckSesiAktif(bsuID string) (*SesiAktifResult, error) {
	bank, err := s.repo.FindBank(bsuID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}
	if bank.JenisBank != models.BSU {
		return nil, errBadRequest("Bank bukan BSU")
	}
	if bank.ParentBankID == nil {
		return nil, errBadRequest("BSU tidak memiliki BSI induk")
	}
	bsiID := *bank.ParentBankID

	namaBSI := ""
	if bsi, err := s.repo.FindBank(bsiID); err == nil {
		namaBSI = bsi.NamaBank
	}

	pgk, err := s.repo.FindLatestPengangkutan(bsuID, bsiID)
	if err != nil {
		return &SesiAktifResult{IsActive: false, BSIID: bsiID, NamaBSI: namaBSI}, nil
	}

	riwayat, err := s.repo.FindLastRiwayat(s.db, pgk.PengangkutanID)
	if err != nil {
		return &SesiAktifResult{
			IsActive: true, PengangkutanID: pgk.PengangkutanID,
			BSIID: bsiID, NamaBSI: namaBSI,
		}, nil
	}

	isTerminal := riwayat.StatusPengangkutan == models.StatusCompleted ||
		riwayat.StatusPengangkutan == models.StatusRejected ||
		riwayat.StatusPengangkutan == models.StatusCanceled

	return &SesiAktifResult{
		IsActive:       !isTerminal,
		PengangkutanID: pgk.PengangkutanID,
		BSIID:          bsiID,
		NamaBSI:        namaBSI,
		StatusTerkini:  string(riwayat.StatusPengangkutan),
	}, nil
}

func (s *pengangkutanService) GetDetailSesiAktif(pengangkutanID string) (*DetailSesiAktifResult, error) {
	pgk, err := s.repo.FindPengangkutan(pengangkutanID)
	if err != nil {
		return nil, errNotFound("Sesi pengangkutan tidak ditemukan")
	}

	namaBSI, namaBSU := "", ""
	if bsi, err := s.repo.FindBank(pgk.BSIID); err == nil {
		namaBSI = bsi.NamaBank
	}
	if bsu, err := s.repo.FindBank(pgk.BSUId); err == nil {
		namaBSU = bsu.NamaBank
	}

	rows, err := s.repo.GetRiwayatDetail(pengangkutanID)
	if err != nil {
		return nil, errInternal("Gagal mengambil riwayat")
	}

	var resp []RiwayatPengangkutanResp
	var statusTerkini models.StatusPengangkutan
	for i, r := range rows {
		if i == 0 {
			statusTerkini = r.Status
		}
		changedBy := r.NamaPetugas
		if changedBy == "" {
			changedBy = r.ChangedBy
		}
		resp = append(resp, RiwayatPengangkutanResp{
			Status:    r.Status,
			ChangedAt: r.ChangedAt,
			ChangedBy: changedBy,
			Catatan:   r.Notes,
		})
	}
	if resp == nil {
		resp = []RiwayatPengangkutanResp{}
	}

	return &DetailSesiAktifResult{
		PengangkutanID: pgk.PengangkutanID,
		BSIID:          pgk.BSIID,
		NamaBSI:        namaBSI,
		BSUID:          pgk.BSUId,
		NamaBSU:        namaBSU,
		BuktiFoto:      pgk.BuktiFoto,
		StatusTerkini:  statusTerkini,
		Riwayat:        resp,
	}, nil
}

func (s *pengangkutanService) GetAllAktif(bsiID, adminID string) (*AllAktifResult, error) {
	bsi, err := s.repo.FindBank(bsiID)
	if err != nil {
		return nil, errNotFound("BSI tidak ditemukan")
	}
	if bsi.JenisBank != models.BSI {
		return nil, errBadRequest("Bank bukan BSI")
	}

	daftarBSU, err := s.repo.FindBSUList(bsiID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data BSU")
	}

	var result []PengangkutanAktifItem
	for _, b := range daftarBSU {
		pgk, err := s.repo.FindLatestPengangkutan(b.BankID, bsiID)
		if err != nil {
			continue
		}
		riwayat, err := s.repo.FindLastRiwayat(s.db, pgk.PengangkutanID)
		if err != nil {
			continue
		}
		isTerminal := riwayat.StatusPengangkutan == models.StatusCompleted ||
			riwayat.StatusPengangkutan == models.StatusRejected ||
			riwayat.StatusPengangkutan == models.StatusCanceled
		if isTerminal {
			continue
		}

		isActionAllowed := pgk.AdminBSIID == nil || *pgk.AdminBSIID == adminID

		tanggalStr := ""
		jamMulai, jamSelesai := "", ""
		if jadwal, err := s.repo.FindJadwal(pgk.JadwalID); err == nil {
			if !jadwal.Tanggal.IsZero() {
				tanggalStr = jadwal.Tanggal.Format("2006-01-02")
			}
			jamMulai = jadwal.JamMulai
			jamSelesai = jadwal.JamSelesai
		}

		result = append(result, PengangkutanAktifItem{
			PengangkutanID:  pgk.PengangkutanID,
			BsuID:           b.BankID,
			NamaBsu:         b.NamaBank,
			StatusTerkini:   riwayat.StatusPengangkutan,
			IsActionAllowed: isActionAllowed,
			Tanggal:         tanggalStr,
			JamMulai:        jamMulai,
			JamSelesai:      jamSelesai,
		})
	}
	if result == nil {
		result = []PengangkutanAktifItem{}
	}

	return &AllAktifResult{BSIID: bsiID, NamaBSI: bsi.NamaBank, Data: result}, nil
}

// ── Private helpers ────────────────────────────────────────────────────────

func hariOf(t time.Time) models.HariEnum {
	return []models.HariEnum{
		models.Minggu, models.Senin, models.Selasa, models.Rabu,
		models.Kamis, models.Jumat, models.Sabtu,
	}[t.Weekday()]
}

func weekOfMonth(date time.Time) int {
	firstOfMonth := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
	firstWeekday := int(firstOfMonth.Weekday())
	adjustedDay := date.Day() + firstWeekday - 1
	return (adjustedDay / 7) + 1
}

func parseFlexibleDate(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

func validTransitionBSI(current, next models.StatusPengangkutan) bool {
	allowed := map[models.StatusPengangkutan][]models.StatusPengangkutan{
		models.StatusRequested: {models.StatusApproved, models.StatusRejected},
		models.StatusApproved:  {models.StatusOTW, models.StatusCanceled},
		models.StatusOTW:       {models.StatusArrived, models.StatusCanceled},
		models.StatusArrived:   {models.StatusCompleted, models.StatusCanceled},
	}
	for _, a := range allowed[current] {
		if a == next {
			return true
		}
	}
	return false
}
