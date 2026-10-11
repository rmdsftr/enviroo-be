package services

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/utils"

	"gorm.io/gorm"
)

type JadwalRepoIface interface {
	FindBank(bankID string) (*models.BankSampah, error)
	FindJadwal(tx *gorm.DB, jadwalID string) (*models.Jadwal, error)
	ListJadwalBank(bankID string, jenis models.JadwalEnum) ([]models.Jadwal, error)
	ListJadwalBankWithNama(bankID string, jenis models.JadwalEnum, sebagaiTarget bool, month, year int) ([]repositories.JadwalBankRow, error)
	ListSemuaJadwal(jenis models.JadwalEnum) ([]repositories.JadwalLengkapRow, error)
	ListStatusSesi(jadwalIDs []string, dari, sampai time.Time) (map[string]string, error)
	CreateJadwal(tx *gorm.DB, jadwal *models.Jadwal) error
	SaveJadwal(tx *gorm.DB, jadwal *models.Jadwal) error
	DeleteJadwal(tx *gorm.DB, jadwal *models.Jadwal) error
	CountPemakaianJadwal(jadwalID string) ([]repositories.PemakaianJadwal, error)
	CountSesiPenimbangan(jadwalID string) (int64, error)
	FindAdminIDByUserBank(userID, bankID string) (string, error)
}

type AksesJadwalInput struct {
	ActorUserID string
	ActorBankID string
	ActorRole   models.RoleAdmin
}

type TambahJadwalInput struct {
	Hari              models.HariEnum
	MingguKe          int
	JamMulai          string
	JamSelesai        string
	JenisJadwal       models.JadwalEnum
	TargetBankID      string
	IsActive          *bool
	IsRutin           bool
	Tanggal           string
	NamaJadwalSpesial string
}

type BatchJadwalItem struct {
	Hari       models.HariEnum `json:"hari"`
	MingguKe   int             `json:"minggu_ke"`
	JamMulai   string          `json:"jam_mulai"`
	JamSelesai string          `json:"jam_selesai"`
}

type UbahJadwalInput struct {
	Hari              models.HariEnum
	MingguKe          int
	JamMulai          string
	JamSelesai        string
	IsActive          *bool
	Tanggal           string
	NamaJadwalSpesial string
}

type JadwalPenimbanganBankItem struct {
	models.Jadwal
	BankName string `json:"bank_name"`
}

type JadwalPengangkutanBankItem struct {
	models.Jadwal
	BankName       string `json:"bank_name"`
	TargetBankName string `json:"target_bank_name"`
}

type JadwalBankResult struct {
	Penimbangan  []JadwalPenimbanganBankItem  `json:"penimbangan"`
	Pengangkutan []JadwalPengangkutanBankItem `json:"pengangkutan"`
}

type JadwalPenimbanganItem struct {
	JadwalID     string `json:"jadwal_id"`
	Tanggal      string `json:"tanggal"`
	JamMulai     string `json:"jam_mulai"`
	JamSelesai   string `json:"jam_selesai"`
	NamaJadwal   string `json:"nama_jadwal"`
	IsRutin      bool   `json:"is_rutin"`
	IsActive     bool   `json:"is_active"`
	StatusJadwal string `json:"status_jadwal"`
}

type JadwalRutinItem struct {
	JadwalID   string          `json:"jadwal_id"`
	Hari       models.HariEnum `json:"hari"`
	MingguKe   int             `json:"minggu_ke"`
	JamMulai   string          `json:"jam_mulai"`
	JamSelesai string          `json:"jam_selesai"`
	IsActive   *bool           `json:"is_active"`
}

type JadwalKhususItem struct {
	JadwalID   string `json:"jadwal_id"`
	NamaJadwal string `json:"nama_jadwal"`
	Tanggal    string `json:"tanggal"`
	JamMulai   string `json:"jam_mulai"`
	JamSelesai string `json:"jam_selesai"`
	IsActive   *bool  `json:"is_active"`
}

type MasterJadwalResult struct {
	Rutin  []JadwalRutinItem  `json:"rutin"`
	Khusus []JadwalKhususItem `json:"khusus"`
}

type JadwalRingkasItem struct {
	JadwalID          string           `json:"jadwal_id"`
	Hari              *models.HariEnum `json:"hari"`
	MingguKe          *int             `json:"minggu_ke"`
	JamMulai          string           `json:"jam_mulai"`
	JamSelesai        string           `json:"jam_selesai"`
	IsActive          *bool            `json:"is_active"`
	Tanggal           *time.Time       `json:"tanggal,omitempty"`
	NamaJadwalSpesial string           `json:"nama_jadwal_spesial,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
}

type JadwalBankGroup struct {
	BankID     string              `json:"bank_id"`
	NamaBank   string              `json:"nama_bank"`
	Rutin      []JadwalRingkasItem `json:"rutin"`
	TidakRutin []JadwalRingkasItem `json:"tidak_rutin"`
}

type SemuaJadwalPenimbanganResult struct {
	BSI []JadwalBankGroup `json:"bsi"`
	BSM []JadwalBankGroup `json:"bsm"`
	BSU []JadwalBankGroup `json:"bsu"`
}

type RuteBSUJadwal struct {
	BSUID      string              `json:"bsu_id"`
	NamaBSU    string              `json:"nama_bsu"`
	Rutin      []JadwalRingkasItem `json:"rutin"`
	TidakRutin []JadwalRingkasItem `json:"tidak_rutin"`
}

type RuteBSIJadwal struct {
	BSIID   string          `json:"bsi_id"`
	NamaBSI string          `json:"nama_bsi"`
	RuteBSU []RuteBSUJadwal `json:"rute_bsu"`
}

type OccurrenceJadwal struct {
	JadwalID          string
	Tanggal           time.Time
	JamMulai          string
	JamSelesai        string
	IsRutin           bool
	IsActive          bool
	NamaJadwalSpesial string
	CreatedAt         time.Time
}

type JadwalService interface {
	TambahJadwal(bankID string, in TambahJadwalInput, akses AksesJadwalInput) (*models.Jadwal, error)
	TambahJadwalBatch(bankID string, items []BatchJadwalItem, akses AksesJadwalInput) ([]models.Jadwal, error)
	JadwalBank(bankID string, month, year int) (*JadwalBankResult, error)
	JadwalPenimbanganBank(bankID string, month, year int) ([]JadwalPenimbanganItem, error)
	MasterJadwalPenimbanganBank(bankID string, month, year int) (*MasterJadwalResult, error)
	SemuaJadwal(jenisParam string) (any, error)
	UbahJadwal(jadwalID string, in UbahJadwalInput, akses AksesJadwalInput) (*models.Jadwal, error)
	HapusJadwal(jadwalID string, akses AksesJadwalInput) error
}

type jadwalService struct {
	db   *gorm.DB
	repo JadwalRepoIface
}

func NewJadwalService(db *gorm.DB, repo JadwalRepoIface) JadwalService {
	return &jadwalService{db: db, repo: repo}
}

func (s *jadwalService) TambahJadwal(bankID string, in TambahJadwalInput, akses AksesJadwalInput) (*models.Jadwal, error) {
	if bankID == "" {
		return nil, errBadRequest("bank_id wajib diisi")
	}

	bank, err := s.repo.FindBank(bankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}

	switch in.JenisJadwal {
	case models.JadwalPenimbangan, models.JadwalPengangkutan:
	default:
		return nil, errBadRequest("jenis_jadwal harus salah satu dari: penimbangan, pengangkutan")
	}

	if in.JenisJadwal == models.JadwalPenimbangan || bank.JenisBank == models.BSU {
		in.TargetBankID = ""
	}
	if in.JenisJadwal == models.JadwalPengangkutan && in.TargetBankID == "" {
		return nil, errBadRequest("target_bank_id wajib diisi untuk jadwal pengangkutan")
	}

	jamMulai, jamSelesai, err := validasiJam(in.JamMulai, in.JamSelesai)
	if err != nil {
		return nil, err
	}

	var (
		hari     *models.HariEnum
		mingguKe *int
		tanggal  *time.Time
	)
	namaSpesial := in.NamaJadwalSpesial

	if in.IsRutin {
		if err := validasiHari(in.Hari); err != nil {
			return nil, err
		}
		if err := validasiMingguKe(in.MingguKe); err != nil {
			return nil, err
		}
		h, m := in.Hari, in.MingguKe
		hari, mingguKe = &h, &m
		namaSpesial = ""
	} else {
		if in.Tanggal == "" {
			return nil, errBadRequest("tanggal wajib diisi untuk jadwal khusus")
		}
		t, errParse := utils.ParseFlexibleDate(in.Tanggal)
		if errParse != nil {
			return nil, errBadRequest("Format tanggal tidak valid. Gunakan YYYY-MM-DD")
		}
		tanggal = &t

		if err := validasiTanggalKhusus(t); err != nil {
			return nil, err
		}
	}

	kandidat := slotDariPola(hari, mingguKe, tanggal, in.IsRutin, jamMulai, jamSelesai)
	if err := s.pastikanTidakBentrok(bankID, in.JenisJadwal, in.TargetBankID, kandidat, ""); err != nil {
		return nil, err
	}

	adminID, err := s.adminPemanggil(akses)
	if err != nil {
		return nil, err
	}

	aktif := true
	if in.IsActive != nil {
		aktif = *in.IsActive
	}
	rutin := in.IsRutin

	idBankSource := bankID
	if in.JenisJadwal == models.JadwalPengangkutan {
		idBankSource = in.TargetBankID
	}

	var targetBankID *string
	if in.TargetBankID != "" {
		targetBankID = &in.TargetBankID
	}

	jadwal := models.Jadwal{
		BankID:            bankID,
		Hari:              hari,
		MingguKe:          mingguKe,
		JamMulai:          jamMulai,
		JamSelesai:        jamSelesai,
		JenisJadwal:       in.JenisJadwal,
		TargetBankID:      targetBankID,
		IsActive:          &aktif,
		IsRutin:           &rutin,
		Tanggal:           tanggal,
		NamaJadwalSpesial: namaSpesial,
		CreatedBy:         adminID,
	}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		jadwalID, genErr := idgen.GenerateJadwalID(tx, in.IsRutin, in.JenisJadwal, idBankSource)
		if genErr != nil {
			return errInternal("Gagal generate jadwal ID: " + genErr.Error())
		}
		jadwal.JadwalID = jadwalID
		if err := s.repo.CreateJadwal(tx, &jadwal); err != nil {
			return errInternal("Gagal menyimpan jadwal: " + err.Error())
		}
		return nil
	})
	if txErr != nil {
		return nil, bungkusErrorTransaksi(txErr, "Gagal menyimpan jadwal: ")
	}

	return &jadwal, nil
}

func (s *jadwalService) TambahJadwalBatch(bankID string, items []BatchJadwalItem, akses AksesJadwalInput) ([]models.Jadwal, error) {
	if bankID == "" {
		return nil, errBadRequest("bank_id wajib diisi")
	}
	if len(items) == 0 {
		return nil, errBadRequest("Minimal satu jadwal harus dikirim")
	}
	if _, err := s.repo.FindBank(bankID); err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}

	existing, err := s.repo.ListJadwalBank(bankID, models.JadwalPenimbangan)
	if err != nil {
		return nil, errInternal("Gagal memeriksa jadwal yang sudah ada: " + err.Error())
	}

	terpakai := make([]slotJadwal, 0, len(existing)+len(items))
	for _, e := range existing {
		terpakai = append(terpakai, slotDariJadwal(e))
	}

	kandidat := make([]slotJadwal, 0, len(items))
	for _, it := range items {
		if err := validasiHari(it.Hari); err != nil {
			return nil, err
		}
		if err := validasiMingguKe(it.MingguKe); err != nil {
			return nil, err
		}
		jamMulai, jamSelesai, err := validasiJam(it.JamMulai, it.JamSelesai)
		if err != nil {
			return nil, err
		}

		slot := slotJadwal{
			Hari:     it.Hari,
			MingguKe: it.MingguKe,
			IsRutin:  true,
			Mulai:    jamMulai,
			Selesai:  jamSelesai,
		}
		for _, t := range terpakai {
			if slotBentrok(t, slot) {
				return nil, errConflict("Jadwal hari "+string(it.Hari)+" jam "+jamMulai+"–"+jamSelesai+
					" bentrok dengan jadwal yang sudah ada", nil)
			}
		}
		terpakai = append(terpakai, slot)
		kandidat = append(kandidat, slot)
	}

	adminID, err := s.adminPemanggil(akses)
	if err != nil {
		return nil, err
	}

	aktif, rutin := true, true
	hasil := make([]models.Jadwal, 0, len(kandidat))

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		hasil = hasil[:0]
		for _, k := range kandidat {
			jadwalID, genErr := idgen.GenerateJadwalID(tx, true, models.JadwalPenimbangan, bankID)
			if genErr != nil {
				return errInternal("Gagal generate jadwal ID: " + genErr.Error())
			}

			hari, mingguKe := k.Hari, k.MingguKe
			jadwal := models.Jadwal{
				JadwalID:    jadwalID,
				BankID:      bankID,
				Hari:        &hari,
				MingguKe:    &mingguKe,
				JamMulai:    k.Mulai,
				JamSelesai:  k.Selesai,
				JenisJadwal: models.JadwalPenimbangan,
				IsActive:    &aktif,
				IsRutin:     &rutin,
				CreatedBy:   adminID,
			}
			if err := s.repo.CreateJadwal(tx, &jadwal); err != nil {
				return errInternal("Gagal menyimpan jadwal: " + err.Error())
			}
			hasil = append(hasil, jadwal)
		}
		return nil
	})
	if txErr != nil {
		return nil, bungkusErrorTransaksi(txErr, "Gagal menyimpan jadwal: ")
	}

	return hasil, nil
}

func (s *jadwalService) JadwalBank(bankID string, month, year int) (*JadwalBankResult, error) {
	if bankID == "" {
		return nil, errBadRequest("bank_id wajib diisi")
	}
	bank, err := s.repo.FindBank(bankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}

	rowsPenimbangan, err := s.repo.ListJadwalBankWithNama(bankID, models.JadwalPenimbangan, false, month, year)
	if err != nil {
		return nil, errInternal("Gagal mengambil jadwal penimbangan: " + err.Error())
	}

	rowsPengangkutan, err := s.repo.ListJadwalBankWithNama(bankID, models.JadwalPengangkutan, bank.JenisBank == models.BSU, month, year)
	if err != nil {
		return nil, errInternal("Gagal mengambil jadwal pengangkutan: " + err.Error())
	}

	hasil := &JadwalBankResult{
		Penimbangan:  make([]JadwalPenimbanganBankItem, 0, len(rowsPenimbangan)),
		Pengangkutan: make([]JadwalPengangkutanBankItem, 0, len(rowsPengangkutan)),
	}
	for _, row := range rowsPenimbangan {
		hasil.Penimbangan = append(hasil.Penimbangan, JadwalPenimbanganBankItem{
			Jadwal:   row.Jadwal,
			BankName: row.BankName,
		})
	}
	for _, row := range rowsPengangkutan {
		hasil.Pengangkutan = append(hasil.Pengangkutan, JadwalPengangkutanBankItem{
			Jadwal:         row.Jadwal,
			BankName:       row.BankName,
			TargetBankName: row.TargetBankName,
		})
	}
	return hasil, nil
}

func (s *jadwalService) JadwalPenimbanganBank(bankID string, month, year int) ([]JadwalPenimbanganItem, error) {
	if bankID == "" {
		return nil, errBadRequest("bank_id wajib diisi")
	}
	if month < 1 || month > 12 || year <= 0 {
		return nil, errBadRequest("Query param month (1-12) dan year wajib diisi")
	}
	if _, err := s.repo.FindBank(bankID); err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}

	dari, sampai := rentangBulan(month, year)
	occurrences, err := s.occurrencePenimbangan(bankID, dari, sampai, false)
	if err != nil {
		return nil, err
	}

	dipakai := make([]OccurrenceJadwal, 0, len(occurrences))
	jadwalIDs := make([]string, 0, len(occurrences))
	for _, occ := range occurrences {
		if occ.Tanggal.Format("2006-01-02") < occ.CreatedAt.Format("2006-01-02") {
			continue
		}
		dipakai = append(dipakai, occ)
		jadwalIDs = append(jadwalIDs, occ.JadwalID)
	}

	statusMap, err := s.repo.ListStatusSesi(jadwalIDs, dari, sampai)
	if err != nil {
		return nil, errInternal("Gagal mengambil status sesi: " + err.Error())
	}

	items := make([]JadwalPenimbanganItem, 0, len(dipakai))
	for _, occ := range dipakai {
		tanggalStr := occ.Tanggal.Format("2006-01-02")
		status, adaSesi := statusMap[occ.JadwalID+"|"+tanggalStr]

		if !occ.IsActive && !adaSesi {
			continue
		}

		namaJadwal := occ.NamaJadwalSpesial
		if occ.IsRutin {
			namaJadwal = "Penimbangan rutin"
		}

		statusJadwal := "upcoming"
		if adaSesi {
			statusJadwal = status
		}

		items = append(items, JadwalPenimbanganItem{
			JadwalID:     occ.JadwalID,
			Tanggal:      tanggalStr,
			JamMulai:     occ.JamMulai,
			JamSelesai:   occ.JamSelesai,
			NamaJadwal:   namaJadwal,
			IsRutin:      occ.IsRutin,
			IsActive:     occ.IsActive,
			StatusJadwal: statusJadwal,
		})
	}

	sort.Slice(items, func(i, k int) bool { return items[i].Tanggal < items[k].Tanggal })
	return items, nil
}

func (s *jadwalService) MasterJadwalPenimbanganBank(bankID string, month, year int) (*MasterJadwalResult, error) {
	if bankID == "" {
		return nil, errBadRequest("bank_id wajib diisi")
	}
	if _, err := s.repo.FindBank(bankID); err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}

	jadwals, err := s.repo.ListJadwalBank(bankID, models.JadwalPenimbangan)
	if err != nil {
		return nil, errInternal("Gagal mengambil jadwal penimbangan: " + err.Error())
	}

	pakaiFilter := month >= 1 && month <= 12 && year > 0
	hasil := &MasterJadwalResult{
		Rutin:  make([]JadwalRutinItem, 0),
		Khusus: make([]JadwalKhususItem, 0),
	}

	for _, j := range jadwals {
		if j.IsRutin != nil && *j.IsRutin {
			item := JadwalRutinItem{
				JadwalID:   j.JadwalID,
				JamMulai:   j.JamMulai,
				JamSelesai: j.JamSelesai,
				IsActive:   j.IsActive,
			}
			if j.Hari != nil {
				item.Hari = *j.Hari
			}
			if j.MingguKe != nil {
				item.MingguKe = *j.MingguKe
			}
			hasil.Rutin = append(hasil.Rutin, item)
			continue
		}

		if j.Tanggal == nil {
			continue
		}
		if pakaiFilter {
			if int(j.Tanggal.Month()) != month || j.Tanggal.Year() != year {
				continue
			}
		}
		hasil.Khusus = append(hasil.Khusus, JadwalKhususItem{
			JadwalID:   j.JadwalID,
			NamaJadwal: j.NamaJadwalSpesial,
			Tanggal:    j.Tanggal.Format("2006-01-02"),
			JamMulai:   j.JamMulai,
			JamSelesai: j.JamSelesai,
			IsActive:   j.IsActive,
		})
	}
	return hasil, nil
}

func (s *jadwalService) SemuaJadwal(jenisParam string) (any, error) {
	if jenisParam != string(models.JadwalPenimbangan) && jenisParam != string(models.JadwalPengangkutan) {
		return nil, errBadRequest("Query param 'type' harus 'penimbangan' atau 'pengangkutan'")
	}
	jenis := models.JadwalEnum(jenisParam)

	rows, err := s.repo.ListSemuaJadwal(jenis)
	if err != nil {
		return nil, errInternal("Gagal mengambil data jadwal: " + err.Error())
	}

	if jenis == models.JadwalPenimbangan {
		return rekapPenimbangan(rows), nil
	}
	return rekapPengangkutan(rows), nil
}

func rekapPenimbangan(rows []repositories.JadwalLengkapRow) *SemuaJadwalPenimbanganResult {
	grup := map[models.JenisBank]map[string]*JadwalBankGroup{
		models.BSI: {},
		models.BSM: {},
		models.BSU: {},
	}

	for _, row := range rows {
		bankMap, ok := grup[row.JenisBank]
		if !ok {
			continue
		}
		if _, ada := bankMap[row.BankID]; !ada {
			bankMap[row.BankID] = &JadwalBankGroup{
				BankID:     row.BankID,
				NamaBank:   row.NamaBank,
				Rutin:      make([]JadwalRingkasItem, 0),
				TidakRutin: make([]JadwalRingkasItem, 0),
			}
		}
		g := bankMap[row.BankID]
		if row.IsRutin != nil && *row.IsRutin {
			g.Rutin = append(g.Rutin, ringkasJadwal(row))
		} else {
			g.TidakRutin = append(g.TidakRutin, ringkasJadwal(row))
		}
	}

	keSlice := func(m map[string]*JadwalBankGroup) []JadwalBankGroup {
		hasil := make([]JadwalBankGroup, 0, len(m))
		for _, v := range m {
			hasil = append(hasil, *v)
		}
		return hasil
	}

	return &SemuaJadwalPenimbanganResult{
		BSI: keSlice(grup[models.BSI]),
		BSM: keSlice(grup[models.BSM]),
		BSU: keSlice(grup[models.BSU]),
	}
}

func rekapPengangkutan(rows []repositories.JadwalLengkapRow) []RuteBSIJadwal {
	bsiMap := map[string]map[string]*RuteBSUJadwal{}
	bsiNama := map[string]string{}
	urutanBSI := []string{}

	for _, row := range rows {
		if _, ada := bsiMap[row.BankID]; !ada {
			bsiMap[row.BankID] = map[string]*RuteBSUJadwal{}
			bsiNama[row.BankID] = row.NamaBank
			urutanBSI = append(urutanBSI, row.BankID)
		}
		bsuMap := bsiMap[row.BankID]
		if _, ada := bsuMap[row.TargetBankID]; !ada {
			bsuMap[row.TargetBankID] = &RuteBSUJadwal{
				BSUID:      row.TargetBankID,
				NamaBSU:    row.TargetBankName,
				Rutin:      make([]JadwalRingkasItem, 0),
				TidakRutin: make([]JadwalRingkasItem, 0),
			}
		}
		rute := bsuMap[row.TargetBankID]
		if row.IsRutin != nil && *row.IsRutin {
			rute.Rutin = append(rute.Rutin, ringkasJadwal(row))
		} else {
			rute.TidakRutin = append(rute.TidakRutin, ringkasJadwal(row))
		}
	}

	hasil := make([]RuteBSIJadwal, 0, len(urutanBSI))
	for _, bsiID := range urutanBSI {
		bsuMap := bsiMap[bsiID]
		rutes := make([]RuteBSUJadwal, 0, len(bsuMap))
		for _, v := range bsuMap {
			rutes = append(rutes, *v)
		}
		hasil = append(hasil, RuteBSIJadwal{
			BSIID:   bsiID,
			NamaBSI: bsiNama[bsiID],
			RuteBSU: rutes,
		})
	}
	return hasil
}

func ringkasJadwal(row repositories.JadwalLengkapRow) JadwalRingkasItem {
	return JadwalRingkasItem{
		JadwalID:          row.JadwalID,
		Hari:              row.Hari,
		MingguKe:          row.MingguKe,
		JamMulai:          row.JamMulai,
		JamSelesai:        row.JamSelesai,
		IsActive:          row.IsActive,
		Tanggal:           row.Tanggal,
		NamaJadwalSpesial: row.NamaJadwalSpesial,
		CreatedAt:         row.CreatedAt,
	}
}

func (s *jadwalService) UbahJadwal(jadwalID string, in UbahJadwalInput, akses AksesJadwalInput) (*models.Jadwal, error) {
	jadwal, err := s.ambilJadwal(jadwalID, akses)
	if err != nil {
		return nil, err
	}
	isRutin := jadwal.IsRutin != nil && *jadwal.IsRutin
	slotLama := slotDariJadwal(*jadwal)

	jumlahSesi, err := s.repo.CountSesiPenimbangan(jadwalID)
	if err != nil {
		return nil, errInternal("Gagal memeriksa histori penimbangan: " + err.Error())
	}
	if jumlahSesi > 0 {
		if isRutin {
			if in.Hari != "" || in.MingguKe != 0 || in.JamMulai != "" || in.JamSelesai != "" {
				return nil, errConflict("Jadwal tidak bisa diubah (hari/minggu ke/jam mulai/jam selesai) karena sudah tercatat sebagai histori penimbangan", nil)
			}
		} else {
			if in.Tanggal != "" || in.JamMulai != "" || in.JamSelesai != "" {
				return nil, errConflict("Jadwal tidak bisa diubah (tanggal/jam mulai/jam selesai) karena sudah tercatat sebagai histori penimbangan", nil)
			}
		}
	}

	if isRutin {
		if in.Hari != "" {
			if err := validasiHari(in.Hari); err != nil {
				return nil, err
			}
			hari := in.Hari
			jadwal.Hari = &hari
		}
		if in.MingguKe != 0 {
			if err := validasiMingguKe(in.MingguKe); err != nil {
				return nil, err
			}
			mingguKe := in.MingguKe
			jadwal.MingguKe = &mingguKe
		}
	}

	if in.JamMulai != "" || in.JamSelesai != "" {
		mulai, selesai := in.JamMulai, in.JamSelesai
		if mulai == "" {
			mulai = jadwal.JamMulai
		}
		if selesai == "" {
			selesai = jadwal.JamSelesai
		}
		mulai, selesai, err = validasiJam(mulai, selesai)
		if err != nil {
			return nil, err
		}
		jadwal.JamMulai, jadwal.JamSelesai = mulai, selesai
	}

	if in.IsActive != nil {
		jadwal.IsActive = in.IsActive
	}

	if !isRutin && in.Tanggal != "" {
		t, errParse := utils.ParseFlexibleDate(in.Tanggal)
		if errParse != nil {
			return nil, errBadRequest("Format tanggal tidak valid. Gunakan YYYY-MM-DD")
		}
		jadwal.Tanggal = &t
	}
	if !isRutin && in.NamaJadwalSpesial != "" {
		jadwal.NamaJadwalSpesial = in.NamaJadwalSpesial
	}

	slotBaru := slotDariJadwal(*jadwal)
	if slotBaru != slotLama {
		if !isRutin && jadwal.Tanggal != nil {
			if err := validasiTanggalKhusus(*jadwal.Tanggal); err != nil {
				return nil, err
			}
		}
		if err := s.pastikanTidakBentrok(jadwal.BankID, jadwal.JenisJadwal,
			nilaiTargetBank(jadwal.TargetBankID), slotBaru, jadwal.JadwalID); err != nil {
			return nil, err
		}
	}

	if err := s.repo.SaveJadwal(nil, jadwal); err != nil {
		return nil, errInternal("Gagal mengupdate jadwal: " + err.Error())
	}
	return jadwal, nil
}

func (s *jadwalService) HapusJadwal(jadwalID string, akses AksesJadwalInput) error {
	jadwal, err := s.ambilJadwal(jadwalID, akses)
	if err != nil {
		return err
	}

	pemakaian, err := s.repo.CountPemakaianJadwal(jadwalID)
	if err != nil {
		return errInternal("Gagal memeriksa pemakaian jadwal: " + err.Error())
	}
	if len(pemakaian) > 0 {
		return errConflict("Jadwal tidak dapat dihapus karena sudah dipakai "+pesanPemakaianJadwal(pemakaian), nil)
	}

	if err := s.repo.DeleteJadwal(nil, jadwal); err != nil {
		return errInternal("Gagal menghapus jadwal: " + err.Error())
	}
	return nil
}

type slotJadwal struct {
	Hari     models.HariEnum
	MingguKe int
	IsRutin  bool
	Tanggal  string
	Mulai    string
	Selesai  string
}

func slotDariJadwal(j models.Jadwal) slotJadwal {
	isRutin := j.IsRutin != nil && *j.IsRutin
	return slotDariPola(j.Hari, j.MingguKe, j.Tanggal, isRutin, j.JamMulai, j.JamSelesai)
}

func slotDariPola(hari *models.HariEnum, mingguKe *int, tanggal *time.Time, isRutin bool, jamMulai, jamSelesai string) slotJadwal {
	slot := slotJadwal{
		IsRutin: isRutin,
		Mulai:   normJam(jamMulai),
		Selesai: normJam(jamSelesai),
	}

	if isRutin {
		if hari != nil {
			slot.Hari = *hari
		}
		if mingguKe != nil {
			slot.MingguKe = *mingguKe
		}
		return slot
	}

	if tanggal != nil {
		slot.Tanggal = tanggal.Format("2006-01-02")
		slot.Hari = utils.HariOf(*tanggal)
		slot.MingguKe = utils.WeekOfMonth(*tanggal)
	}
	return slot
}

func slotBentrok(a, b slotJadwal) bool {
	if !jamBeririsan(a.Mulai, a.Selesai, b.Mulai, b.Selesai) {
		return false
	}

	switch {
	case a.IsRutin && b.IsRutin:
		return a.Hari == b.Hari && a.MingguKe == b.MingguKe
	case !a.IsRutin && !b.IsRutin:
		return a.Tanggal != "" && a.Tanggal == b.Tanggal
	default:
		rutin, khusus := a, b
		if b.IsRutin {
			rutin, khusus = b, a
		}
		return khusus.Hari == rutin.Hari && khusus.MingguKe == rutin.MingguKe
	}
}

func jamBeririsan(aMulai, aSelesai, bMulai, bSelesai string) bool {
	return aMulai < bSelesai && bMulai < aSelesai
}

func normJam(jam string) string {
	if len(jam) >= 5 {
		return jam[:5]
	}
	return jam
}

func validasiJam(jamMulai, jamSelesai string) (string, string, error) {
	if jamMulai == "" || jamSelesai == "" {
		return "", "", errBadRequest("jam_mulai dan jam_selesai wajib diisi")
	}

	mulai := normJam(jamMulai)
	if _, err := time.Parse("15:04", mulai); err != nil {
		return "", "", errBadRequest("jam_mulai tidak valid, gunakan format HH:MM")
	}
	selesai := normJam(jamSelesai)
	if _, err := time.Parse("15:04", selesai); err != nil {
		return "", "", errBadRequest("jam_selesai tidak valid, gunakan format HH:MM")
	}
	if selesai <= mulai {
		return "", "", errBadRequest("jam_selesai harus lebih besar dari jam_mulai")
	}
	return mulai, selesai, nil
}

func validasiTanggalKhusus(tanggal time.Time) error {
	if tanggal.Format("2006-01-02") <= time.Now().Format("2006-01-02") {
		return errBadRequest("Jadwal khusus paling lambat dibuat H-1, tanggal minimal besok")
	}
	return nil
}

func validasiHari(hari models.HariEnum) error {
	switch hari {
	case models.Senin, models.Selasa, models.Rabu, models.Kamis,
		models.Jumat, models.Sabtu, models.Minggu:
		return nil
	}
	return errBadRequest("hari harus salah satu dari: senin, selasa, rabu, kamis, jumat, sabtu, minggu")
}

func validasiMingguKe(mingguKe int) error {
	if mingguKe < 1 || mingguKe > 5 {
		return errBadRequest("minggu_ke harus antara 1 dan 5")
	}
	return nil
}

func rentangBulan(month, year int) (time.Time, time.Time) {
	dari := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	return dari, dari.AddDate(0, 1, -1)
}

func nilaiTargetBank(targetBankID *string) string {
	if targetBankID == nil {
		return ""
	}
	return *targetBankID
}

func (s *jadwalService) pastikanTidakBentrok(bankID string, jenis models.JadwalEnum, targetBankID string, kandidat slotJadwal, kecualiID string) error {
	existing, err := s.repo.ListJadwalBank(bankID, jenis)
	if err != nil {
		return errInternal("Gagal memeriksa jadwal yang sudah ada: " + err.Error())
	}

	for _, e := range existing {
		if e.JadwalID == kecualiID {
			continue
		}
		if jenis == models.JadwalPengangkutan && nilaiTargetBank(e.TargetBankID) != targetBankID {
			continue
		}
		if slotBentrok(slotDariJadwal(e), kandidat) {
			return errConflict("Jadwal bentrok dengan jadwal yang sudah ada pada waktu tersebut", nil)
		}
	}
	return nil
}

func (s *jadwalService) occurrencePenimbangan(bankID string, dari, sampai time.Time, hanyaAktif bool) ([]OccurrenceJadwal, error) {
	if bankID == "" {
		return nil, errBadRequest("bank_id wajib diisi")
	}

	jadwals, err := s.repo.ListJadwalBank(bankID, models.JadwalPenimbangan)
	if err != nil {
		return nil, errInternal("Gagal mengambil jadwal penimbangan: " + err.Error())
	}
	return ekspansiOccurrence(jadwals, dari, sampai, hanyaAktif), nil
}

func ekspansiOccurrence(jadwals []models.Jadwal, dari, sampai time.Time, hanyaAktif bool) []OccurrenceJadwal {
	hasil := make([]OccurrenceJadwal, 0, len(jadwals))

	for _, j := range jadwals {
		isRutin := j.IsRutin != nil && *j.IsRutin
		isActive := j.IsActive != nil && *j.IsActive
		if hanyaAktif && !isActive {
			continue
		}

		buat := func(t time.Time) OccurrenceJadwal {
			return OccurrenceJadwal{
				JadwalID:          j.JadwalID,
				Tanggal:           t,
				JamMulai:          j.JamMulai,
				JamSelesai:        j.JamSelesai,
				IsRutin:           isRutin,
				IsActive:          isActive,
				NamaJadwalSpesial: j.NamaJadwalSpesial,
				CreatedAt:         j.CreatedAt,
			}
		}

		if !isRutin {
			if j.Tanggal == nil {
				continue
			}
			tgl := time.Date(j.Tanggal.Year(), j.Tanggal.Month(), j.Tanggal.Day(), 0, 0, 0, 0, dari.Location())
			if tgl.Before(dari) || tgl.After(sampai) {
				continue
			}
			hasil = append(hasil, buat(tgl))
			continue
		}

		if j.Hari == nil || j.MingguKe == nil {
			continue
		}
		for d := dari; !d.After(sampai); d = d.AddDate(0, 0, 1) {
			if utils.HariOf(d) == *j.Hari && utils.WeekOfMonth(d) == *j.MingguKe {
				hasil = append(hasil, buat(d))
			}
		}
	}
	return hasil
}

func (s *jadwalService) ambilJadwal(jadwalID string, akses AksesJadwalInput) (*models.Jadwal, error) {
	if jadwalID == "" {
		return nil, errBadRequest("jadwal_id wajib diisi")
	}

	jadwal, err := s.repo.FindJadwal(nil, jadwalID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNotFound("Jadwal tidak ditemukan")
		}
		return nil, errInternal("Gagal mencari jadwal: " + err.Error())
	}
	if err := pastikanAksesJadwal(jadwal, akses); err != nil {
		return nil, err
	}
	return jadwal, nil
}

func pastikanAksesJadwal(jadwal *models.Jadwal, akses AksesJadwalInput) error {
	if akses.ActorBankID == "" ||
		akses.ActorBankID == jadwal.BankID ||
		akses.ActorBankID == nilaiTargetBank(jadwal.TargetBankID) {
		return nil
	}
	return errForbidden("Akses ditolak. Anda tidak memiliki akses ke jadwal bank ini")
}

func (s *jadwalService) adminPemanggil(akses AksesJadwalInput) (string, error) {
	adminID, err := s.repo.FindAdminIDByUserBank(akses.ActorUserID, akses.ActorBankID)
	if err != nil {
		return "", errInternal("Gagal mencari data admin: " + err.Error())
	}
	if adminID == "" {
		return "", errForbidden("Akun admin aktif untuk bank ini tidak ditemukan")
	}
	return adminID, nil
}

func pesanPemakaianJadwal(pemakaian []repositories.PemakaianJadwal) string {
	bagian := make([]string, 0, len(pemakaian))
	for _, p := range pemakaian {
		bagian = append(bagian, fmt.Sprintf("%d %s", p.Jumlah, p.Label))
	}
	return strings.Join(bagian, ", ")
}
