package services

import (
	"context"
	"errors"
	"log"
	"time"

	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/utils"

	"gorm.io/gorm"
)

const jamBuatSesiPending = 20

type PenimbanganRepoIface interface {
	FindPenimbangan(penimbanganID string) (*models.Penimbangan, error)
	FindPenimbanganLock(tx *gorm.DB, penimbanganID string, status models.StatusPenimbangan) (*models.Penimbangan, error)
	SavePenimbangan(tx *gorm.DB, p *models.Penimbangan) error
	CreatePenimbangan(tx *gorm.DB, p *models.Penimbangan) error
	CountSesiAktif(tx *gorm.DB, bankID string, tanggal string) (int64, error)
	CountSesiByJadwalTanggal(jadwalID string, tanggal time.Time) (int64, error)
	ListSesiTanggal(bankID, tanggal string, statuses []models.StatusPenimbangan) ([]repositories.SesiHariIniRow, error)
	ListRiwayat(bankID string, start, end *time.Time) ([]repositories.RiwayatPenimbanganRow, error)
	FindHeaderSesi(penimbanganID string) (*repositories.HeaderSesiRow, error)
	ListSetoranSesi(penimbanganID string) ([]repositories.SetoranSesiRow, error)
	TutupSesiTerlewat() (int64, error)
	FindBank(bankID string) (*models.BankSampah, error)
	FindJadwal(jadwalID string) (*models.Jadwal, error)
	FindJadwalPenimbangan(jadwalID string) (*models.Jadwal, error)
	ListJadwalJatuhTempo(hari models.HariEnum, mingguKe int, tanggal string) ([]models.Jadwal, error)
	FindNasabahTargets(bankID string) ([]repositories.NotifTarget, error)
	FindAdminTargets(bankID string) ([]repositories.NotifTarget, error)
}

type SesiHariIniItem struct {
	PenimbanganID     string                   `json:"penimbangan_id"`
	TanggalSesi       time.Time                `json:"tanggal_sesi"`
	JamMulai          string                   `json:"jam_mulai"`
	JamSelesai        string                   `json:"jam_selesai"`
	NamaJadwalSpesial string                   `json:"nama_jadwal_spesial"`
	StatusPenimbangan models.StatusPenimbangan `json:"status_penimbangan"`
}

type SesiPenimbanganAktifDetail struct {
	PenimbanganID     string    `json:"penimbangan_id"`
	TanggalSesi       time.Time `json:"tanggal_sesi"`
	JamMulai          string    `json:"jam_mulai"`
	JamSelesai        string    `json:"jam_selesai"`
	NamaJadwalSpesial string    `json:"nama_jadwal_spesial"`
}

type SesiPenimbanganAktifResult struct {
	IsActive        bool                        `json:"is_active"`
	Detail          *SesiPenimbanganAktifDetail `json:"detail"`
	PendingSessions int                         `json:"pending_sessions"`
}

type UbahStatusSesiInput struct {
	PenimbanganID string
	StatusTujuan  models.StatusPenimbangan
	ActorUserID   string
	ActorBankID   string
}

type BatalkanSesiInput struct {
	PenimbanganID    string
	JadwalID         string
	TanggalSesi      string
	AlasanPembatalan string
	ActorUserID      string
	ActorBankID      string
}

type BatalkanSesiResult struct {
	Penimbangan *models.Penimbangan
	Dibuat      bool
}

type RiwayatItem struct {
	PenimbanganID     string                   `json:"penimbangan_id"`
	TanggalSesi       time.Time                `json:"tanggal_sesi"`
	StatusPenimbangan models.StatusPenimbangan `json:"status_penimbangan"`
	JumlahSetoran     int64                    `json:"jumlah_setoran"`
}

type AksesSesiInput struct {
	ActorBankID string
	ActorRole   models.RoleAdmin
}

type PenimbanganService interface {
	ListSesiHariIni(bankID string) ([]SesiHariIniItem, error)
	CekSesiAktif(bankID string) (*SesiPenimbanganAktifResult, error)
	UbahStatusSesi(in UbahStatusSesiInput) (*models.Penimbangan, error)
	BatalkanSesi(in BatalkanSesiInput) (*BatalkanSesiResult, error)
	RiwayatPenimbangan(bankID, startDate, endDate string) ([]RiwayatItem, error)
	DetailSesi(penimbanganID string, in AksesSesiInput) (*repositories.HeaderSesiRow, error)

	JamBuatSesiPending() int
	BuatSesiPendingBesok(now time.Time) (jumlahDibuat int, sukses bool)
	TutupSesiTerlewat() (int64, error)
}

type penimbanganService struct {
	db       *gorm.DB
	repo     PenimbanganRepoIface
	notifSvc NotifikasiService
}

func NewPenimbanganService(db *gorm.DB, repo PenimbanganRepoIface, notifSvc NotifikasiService) PenimbanganService {
	return &penimbanganService{db: db, repo: repo, notifSvc: notifSvc}
}

var errSesiAktifLain = errors.New("sesi penimbangan lain masih aktif")

func (s *penimbanganService) ListSesiHariIni(bankID string) ([]SesiHariIniItem, error) {
	if _, err := s.repo.FindBank(bankID); err != nil {
		return nil, errBadRequest("Bank sampah tidak ditemukan")
	}

	rows, err := s.repo.ListSesiTanggal(bankID, tanggalHariIni(), nil)
	if err != nil {
		return nil, errInternal("Gagal mengambil data sesi penimbangan hari ini")
	}

	items := make([]SesiHariIniItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, SesiHariIniItem{
			PenimbanganID:     row.PenimbanganID,
			TanggalSesi:       row.TanggalSesi,
			JamMulai:          row.JamMulai,
			JamSelesai:        row.JamSelesai,
			NamaJadwalSpesial: namaJadwalDisplay(row.IsRutin, row.NamaJadwalSpesial),
			StatusPenimbangan: row.StatusPenimbangan,
		})
	}
	return items, nil
}

func (s *penimbanganService) CekSesiAktif(bankID string) (*SesiPenimbanganAktifResult, error) {
	if _, err := s.repo.FindBank(bankID); err != nil {
		return nil, errBadRequest("Bank sampah tidak ditemukan")
	}

	rows, err := s.repo.ListSesiTanggal(bankID, tanggalHariIni(),
		[]models.StatusPenimbangan{models.StatusTimbangPending, models.StatusAktif})
	if err != nil {
		return nil, errInternal("Gagal mengecek sesi penimbangan aktif")
	}

	var sesiAktif *repositories.SesiHariIniRow
	pendingSessions := 0
	for i := range rows {
		if rows[i].StatusPenimbangan == models.StatusAktif {
			sesiAktif = &rows[i]
		} else {
			pendingSessions++
		}
	}

	if sesiAktif == nil {
		return &SesiPenimbanganAktifResult{IsActive: false, Detail: nil, PendingSessions: pendingSessions}, nil
	}

	return &SesiPenimbanganAktifResult{
		IsActive: true,
		Detail: &SesiPenimbanganAktifDetail{
			PenimbanganID:     sesiAktif.PenimbanganID,
			TanggalSesi:       sesiAktif.TanggalSesi,
			JamMulai:          sesiAktif.JamMulai,
			JamSelesai:        sesiAktif.JamSelesai,
			NamaJadwalSpesial: namaJadwalDisplay(sesiAktif.IsRutin, sesiAktif.NamaJadwalSpesial),
		},
		PendingSessions: pendingSessions,
	}, nil
}

func (s *penimbanganService) UbahStatusSesi(in UbahStatusSesiInput) (*models.Penimbangan, error) {
	if in.StatusTujuan != models.StatusAktif && in.StatusTujuan != models.StatusSelesai {
		return nil, errBadRequest("Status penimbangan tidak valid. Gunakan 'aktif' (mulai sesi) atau 'selesai' (tutup sesi). Untuk membatalkan sesi, gunakan endpoint /penimbangan/batal")
	}

	cek, err := s.repo.FindPenimbangan(in.PenimbanganID)
	if err != nil {
		return nil, errNotFound("Data penimbangan tidak ditemukan")
	}
	if err := pastikanAksesSesi(in.ActorBankID, cek.BankID, false, ""); err != nil {
		return nil, err
	}

	statusAwal := models.StatusTimbangPending
	if in.StatusTujuan == models.StatusSelesai {
		statusAwal = models.StatusAktif
	}

	nowTime := time.Now()
	userID := in.ActorUserID
	var hasil *models.Penimbangan

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		penimbangan, err := s.repo.FindPenimbanganLock(tx, in.PenimbanganID, statusAwal)
		if err != nil {
			return err
		}

		switch in.StatusTujuan {
		case models.StatusAktif:
			activeCount, err := s.repo.CountSesiAktif(tx, *penimbangan.BankID, tanggalHariIni())
			if err != nil {
				return err
			}
			if activeCount > 0 {
				return errSesiAktifLain
			}
			penimbangan.StartedBy = &userID
			penimbangan.StartedAt = &nowTime
			penimbangan.StatusPenimbangan = models.StatusAktif

		case models.StatusSelesai:
			penimbangan.EndedBy = &userID
			penimbangan.EndedAt = &nowTime
			penimbangan.StatusPenimbangan = models.StatusSelesai
		}

		if err := s.repo.SavePenimbangan(tx, penimbangan); err != nil {
			return err
		}
		hasil = penimbangan
		return nil
	})

	if txErr != nil {
		switch {
		case errors.Is(txErr, errSesiAktifLain), errors.Is(txErr, gorm.ErrDuplicatedKey):
			return nil, errConflict("Masih ada sesi penimbangan lain yang aktif hari ini. Selesaikan sesi itu dulu sebelum memulai sesi ini", nil)
		case errors.Is(txErr, gorm.ErrRecordNotFound):
			if in.StatusTujuan == models.StatusAktif {
				return nil, errConflict("Sesi ini bukan berstatus pending, tidak bisa dimulai", nil)
			}
			return nil, errConflict("Sesi ini bukan berstatus aktif, tidak bisa ditutup", nil)
		default:
			return nil, errInternal("Gagal menyimpan perubahan status penimbangan")
		}
	}

	s.kirimNotifSesiDibukaDitutup(*hasil.BankID, hasil.PenimbanganID, in.StatusTujuan == models.StatusAktif)
	return hasil, nil
}

func (s *penimbanganService) BatalkanSesi(in BatalkanSesiInput) (*BatalkanSesiResult, error) {
	if in.AlasanPembatalan == "" {
		return nil, errBadRequest("alasan_pembatalan wajib diisi")
	}
	if in.PenimbanganID != "" {
		return s.batalkanSesiPending(in)
	}
	return s.batalkanSesiSebelumDibuat(in)
}

func (s *penimbanganService) batalkanSesiPending(in BatalkanSesiInput) (*BatalkanSesiResult, error) {
	cek, err := s.repo.FindPenimbangan(in.PenimbanganID)
	if err != nil {
		return nil, errNotFound("Data penimbangan tidak ditemukan")
	}
	if err := pastikanAksesSesi(in.ActorBankID, cek.BankID, false, ""); err != nil {
		return nil, err
	}
	if cek.JadwalID == nil {
		return nil, errInternal("Data jadwal pada sesi ini tidak valid")
	}

	jadwal, err := s.repo.FindJadwal(*cek.JadwalID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data jadwal")
	}

	nowTime := time.Now()
	userID := in.ActorUserID
	var hasil *models.Penimbangan

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		penimbangan, err := s.repo.FindPenimbanganLock(tx, in.PenimbanganID, models.StatusTimbangPending)
		if err != nil {
			return err
		}
		penimbangan.EndedBy = &userID
		penimbangan.EndedAt = &nowTime
		penimbangan.StatusPenimbangan = models.StatusDibatalkan
		penimbangan.AlasanPembatalan = &in.AlasanPembatalan

		if err := s.repo.SavePenimbangan(tx, penimbangan); err != nil {
			return err
		}
		hasil = penimbangan
		return nil
	})

	if txErr != nil {
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return nil, errConflict("Sesi ini bukan berstatus pending, tidak bisa dibatalkan lewat sini", nil)
		}
		return nil, errInternal("Gagal membatalkan sesi penimbangan")
	}

	s.kirimNotifPembatalan(jadwal.BankID, hasil.TanggalSesi, jadwal.JamMulai, jadwal.JamSelesai, hasil.PenimbanganID)
	return &BatalkanSesiResult{Penimbangan: hasil}, nil
}

func (s *penimbanganService) batalkanSesiSebelumDibuat(in BatalkanSesiInput) (*BatalkanSesiResult, error) {
	if in.JadwalID == "" || in.TanggalSesi == "" {
		return nil, errBadRequest("jadwal_id dan tanggal_sesi wajib diisi kalau penimbangan_id belum ada")
	}

	tanggalSesi, err := time.Parse("2006-01-02", in.TanggalSesi)
	if err != nil {
		return nil, errBadRequest("tanggal_sesi tidak valid, gunakan format YYYY-MM-DD")
	}

	jadwal, err := s.repo.FindJadwalPenimbangan(in.JadwalID)
	if err != nil {
		return nil, errNotFound("Jadwal penimbangan tidak ditemukan")
	}

	if in.ActorBankID != jadwal.BankID {
		return nil, errForbidden("Akses ditolak. Anda tidak memiliki akses ke jadwal ini")
	}

	if jadwal.IsActive == nil || !*jadwal.IsActive {
		return nil, errBadRequest("Jadwal ini sudah tidak aktif")
	}

	today := time.Now()
	todayDateOnly := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	if tanggalSesi.Before(todayDateOnly) {
		return nil, errBadRequest("tanggal_sesi tidak boleh tanggal yang sudah lewat")
	}

	if err := pastikanTanggalCocokJadwal(jadwal, tanggalSesi); err != nil {
		return nil, err
	}

	sudahAda, err := s.repo.CountSesiByJadwalTanggal(in.JadwalID, tanggalSesi)
	if err != nil {
		return nil, errInternal("Gagal mengecek sesi existing")
	}
	if sudahAda > 0 {
		return nil, errConflict("Sudah ada sesi penimbangan (atau pembatalan) untuk jadwal dan tanggal ini", nil)
	}

	var hasil *models.Penimbangan
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		penimbangan, err := s.buatSesiPenimbangan(tx, *jadwal, tanggalSesi, models.StatusDibatalkan, &in.AlasanPembatalan)
		if err != nil {
			return err
		}
		hasil = penimbangan
		return nil
	})

	if txErr != nil {
		if _, ok := txErr.(*ServiceError); ok {
			return nil, txErr
		}
		if errors.Is(txErr, gorm.ErrDuplicatedKey) {
			return nil, errConflict("Sudah ada sesi penimbangan (atau pembatalan) untuk jadwal dan tanggal ini", nil)
		}
		return nil, errInternal("Gagal mencatat pembatalan sesi penimbangan")
	}

	s.kirimNotifPembatalan(jadwal.BankID, tanggalSesi, jadwal.JamMulai, jadwal.JamSelesai, hasil.PenimbanganID)
	return &BatalkanSesiResult{Penimbangan: hasil, Dibuat: true}, nil
}

func (s *penimbanganService) RiwayatPenimbangan(bankID, startDate, endDate string) ([]RiwayatItem, error) {
	if _, err := s.repo.FindBank(bankID); err != nil {
		return nil, errBadRequest("Bank sampah tidak ditemukan")
	}

	var start, end *time.Time
	if t, err := time.Parse("2006-01-02", startDate); err == nil {
		start = &t
	}
	if t, err := time.Parse("2006-01-02", endDate); err == nil {
		end = &t
	}

	rows, err := s.repo.ListRiwayat(bankID, start, end)
	if err != nil {
		return nil, errInternal("Gagal mengambil data penimbangan")
	}

	items := make([]RiwayatItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, RiwayatItem{
			PenimbanganID:     row.PenimbanganID,
			TanggalSesi:       row.TanggalSesi,
			StatusPenimbangan: row.StatusPenimbangan,
			JumlahSetoran:     row.JumlahSetoran,
		})
	}
	return items, nil
}

func (s *penimbanganService) DetailSesi(penimbanganID string, in AksesSesiInput) (*repositories.HeaderSesiRow, error) {
	header, err := s.repo.FindHeaderSesi(penimbanganID)
	if err != nil {
		return nil, errNotFound("Penimbangan tidak ditemukan")
	}

	isBSI := in.ActorRole == models.AdminBSI || in.ActorRole == models.PetugasBSI
	if err := pastikanAksesSesi(in.ActorBankID, &header.BankID, isBSI, ""); err != nil {
		return nil, err
	}

	listSetoran, err := s.repo.ListSetoranSesi(penimbanganID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data setoran")
	}
	if listSetoran == nil {
		listSetoran = []repositories.SetoranSesiRow{}
	}
	header.ListSetoran = listSetoran

	return header, nil
}

func (s *penimbanganService) JamBuatSesiPending() int { return jamBuatSesiPending }

func (s *penimbanganService) BuatSesiPendingBesok(now time.Time) (int, bool) {
	tomorrow := now.AddDate(0, 0, 1)
	tomorrowDate := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, tomorrow.Location())
	tomorrowStr := tomorrowDate.Format("2006-01-02")

	jadwals, err := s.repo.ListJadwalJatuhTempo(utils.HariOf(tomorrow), utils.WeekOfMonth(tomorrow), tomorrowStr)
	if err != nil {
		log.Printf("[Penimbangan] Gagal query jadwal penimbangan H-1: %v", err)
		return 0, false
	}
	if len(jadwals) == 0 {
		return 0, true
	}

	log.Printf("[Penimbangan] Ditemukan %d jadwal penimbangan untuk besok (%s)", len(jadwals), tomorrowStr)
	dibuat := 0
	sukses := true

	for _, jadwal := range jadwals {
		sudahAda, err := s.repo.CountSesiByJadwalTanggal(jadwal.JadwalID, tomorrowDate)
		if err != nil {
			log.Printf("[Penimbangan] Gagal cek sesi existing jadwal %s: %v", jadwal.JadwalID, err)
			sukses = false
			continue
		}
		if sudahAda > 0 {
			continue
		}

		var baru *models.Penimbangan
		txErr := s.db.Transaction(func(tx *gorm.DB) error {
			p, err := s.buatSesiPenimbangan(tx, jadwal, tomorrowDate, models.StatusTimbangPending, nil)
			if err != nil {
				return err
			}
			baru = p
			return nil
		})

		if txErr != nil {
			if errors.Is(txErr, gorm.ErrDuplicatedKey) {
				log.Printf("[Penimbangan] Sesi jadwal %s tanggal %s sudah dibuat pihak lain, skip", jadwal.JadwalID, tomorrowStr)
				continue
			}
			log.Printf("[Penimbangan] Gagal membuat sesi pending untuk jadwal %s: %v", jadwal.JadwalID, txErr)
			sukses = false
			continue
		}

		dibuat++
		log.Printf("[Penimbangan] Sesi pending %s dibuat untuk jadwal %s (bank %s, tanggal %s)",
			baru.PenimbanganID, jadwal.JadwalID, jadwal.BankID, tomorrowStr)
		s.kirimReminderH1(jadwal, baru.PenimbanganID)
	}

	return dibuat, sukses
}

func (s *penimbanganService) TutupSesiTerlewat() (int64, error) {
	rows, err := s.repo.TutupSesiTerlewat()
	if err != nil {
		log.Printf("[Penimbangan] Gagal menutup sesi terlewat: %v", err)
		return 0, err
	}
	if rows > 0 {
		log.Printf("[Penimbangan] %d sesi penimbangan ditutup/dibatalkan otomatis", rows)
	}
	return rows, nil
}

func (s *penimbanganService) buatSesiPenimbangan(tx *gorm.DB, jadwal models.Jadwal, tanggal time.Time, status models.StatusPenimbangan, alasan *string) (*models.Penimbangan, error) {
	penimbanganID, err := idgen.GeneratePenimbanganID(tx, jadwal.BankID)
	if err != nil {
		return nil, errInternal("Gagal generate penimbangan ID: " + err.Error())
	}

	jadwalID := jadwal.JadwalID
	bankID := jadwal.BankID
	penimbangan := &models.Penimbangan{
		PenimbanganID:     penimbanganID,
		JadwalID:          &jadwalID,
		BankID:            &bankID,
		StatusPenimbangan: status,
		TanggalSesi:       tanggal,
		AlasanPembatalan:  alasan,
		CreatedAt:         time.Now(),
	}

	if err := s.repo.CreatePenimbangan(tx, penimbangan); err != nil {
		return nil, err
	}
	return penimbangan, nil
}

func (s *penimbanganService) kirimNotifKeBank(bankID, konteks string, kirimNasabah, kirimAdmin func(ctx context.Context, t repositories.NotifTarget) error) {
	go func() {
		ctx := context.Background()

		if kirimNasabah != nil {
			targets, err := s.repo.FindNasabahTargets(bankID)
			if err != nil {
				log.Printf("[%s] Gagal query nasabah bank %s: %v", konteks, bankID, err)
			} else {
				for _, t := range targets {
					if err := kirimNasabah(ctx, t); err != nil {
						log.Printf("[%s] Gagal kirim notif ke nasabah %s: %v", konteks, t.UserID, err)
					}
				}
			}
		}

		if kirimAdmin != nil {
			targets, err := s.repo.FindAdminTargets(bankID)
			if err != nil {
				log.Printf("[%s] Gagal query admin/petugas bank %s: %v", konteks, bankID, err)
				return
			}
			for _, t := range targets {
				if err := kirimAdmin(ctx, t); err != nil {
					log.Printf("[%s] Gagal kirim notif ke admin/petugas %s: %v", konteks, t.UserID, err)
				}
			}
		}
	}()
}

func (s *penimbanganService) kirimNotifSesiDibukaDitutup(bankID, penimbanganID string, dibuka bool) {
	s.kirimNotifKeBank(bankID, "UpdatePenimbangan", func(ctx context.Context, t repositories.NotifTarget) error {
		if dibuka {
			return s.notifSvc.NotifSesiPenimbanganDibuka(ctx, t.UserID, t.FCMToken, penimbanganID)
		}
		return s.notifSvc.NotifSesiPenimbanganDitutup(ctx, t.UserID, t.FCMToken, penimbanganID)
	}, nil)
}

func (s *penimbanganService) kirimNotifPembatalan(bankID string, tanggalSesi time.Time, jamMulai, jamSelesai, refID string) {
	s.kirimNotifKeBank(bankID, "BatalkanPenimbangan", func(ctx context.Context, t repositories.NotifTarget) error {
		return s.notifSvc.NotifPembatalanPenimbangan(ctx, t.UserID, t.FCMToken, tanggalSesi, jamMulai, jamSelesai, refID)
	}, func(ctx context.Context, t repositories.NotifTarget) error {
		return s.notifSvc.NotifPembatalanPenimbanganAdmin(ctx, t.UserID, t.FCMToken, tanggalSesi, jamMulai, jamSelesai, refID)
	})
}

func (s *penimbanganService) kirimReminderH1(jadwal models.Jadwal, penimbanganID string) {
	bank, err := s.repo.FindBank(jadwal.BankID)
	if err != nil {
		log.Printf("[Penimbangan] Bank tidak ditemukan untuk jadwal %s: %v", jadwal.JadwalID, err)
		return
	}

	namaBank := bank.NamaBank
	s.kirimNotifKeBank(jadwal.BankID, "ReminderPenimbanganH1", func(ctx context.Context, t repositories.NotifTarget) error {
		return s.notifSvc.NotifReminderPenimbanganH1(ctx, t.UserID, t.FCMToken, namaBank, jadwal.JamMulai, jadwal.JamSelesai, penimbanganID)
	}, func(ctx context.Context, t repositories.NotifTarget) error {
		return s.notifSvc.NotifReminderPenimbanganAdminH1(ctx, t.UserID, t.FCMToken, namaBank, jadwal.JamMulai, jadwal.JamSelesai, penimbanganID)
	})
}

func pastikanAksesSesi(actorBankID string, bankID *string, bypass bool, pesan string) error {
	if bypass {
		return nil
	}
	if pesan == "" {
		pesan = "Akses ditolak. Anda tidak memiliki akses ke sesi penimbangan ini"
	}
	if bankID == nil || actorBankID != *bankID {
		return errForbidden(pesan)
	}
	return nil
}

func pastikanTanggalCocokJadwal(jadwal *models.Jadwal, tanggal time.Time) error {
	isRutin := jadwal.IsRutin != nil && *jadwal.IsRutin
	if !isRutin {
		if jadwal.Tanggal == nil || jadwal.Tanggal.Format("2006-01-02") != tanggal.Format("2006-01-02") {
			return errBadRequest("tanggal_sesi tidak cocok dengan tanggal jadwal ini")
		}
		return nil
	}

	if jadwal.Hari == nil || jadwal.MingguKe == nil {
		return errBadRequest("tanggal_sesi tidak cocok dengan pola jadwal rutin ini")
	}
	hariCocok := utils.HariOf(tanggal) == *jadwal.Hari
	mingguCocok := *jadwal.MingguKe == utils.WeekOfMonth(tanggal)
	if !hariCocok || !mingguCocok {
		return errBadRequest("tanggal_sesi tidak cocok dengan pola jadwal rutin ini")
	}
	return nil
}

func namaJadwalDisplay(isRutin *bool, namaSpesial string) string {
	if isRutin == nil || *isRutin {
		return "Jadwal penimbangan rutin"
	}
	return namaSpesial
}

func tanggalHariIni() string {
	return time.Now().Format("2006-01-02")
}
