package services

import (
	"errors"
	"time"

	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/utils"

	"gorm.io/gorm"
)

const saldoNasabahPageLimit = 20

type DashboardRepoIface interface {
	FindBank(bankID string) (*models.BankSampah, error)
	FindNasabah(nasabahID string) (*models.Nasabah, error)
	CountNasabahAktif(bankID string) (int64, error)
	CountAdminAktif(bankID string) (int64, error)
	CountBSU(bsiID string) (int64, error)
	CountNasabahBank(bankID string) (int64, error)
	SumSaldoBank(bankID string) (repositories.SaldoPerSatuan, error)
	SumSaldoNasabah(nasabahID string) (repositories.SaldoPerSatuan, error)
	SumSaldoSemuaNasabah(bankID string) (repositories.SaldoPerSatuan, error)
	ListNasabahDenganSaldo(bankID string, limit, offset int) ([]repositories.NasabahSaldoRow, error)
	FindReward(rewardID int) (*models.Reward, error)
	FindRewardByNama(nama models.RewardEnum) (*models.Reward, error)
	FindRewardByJenis(nama models.RewardEnum, satuan models.SatuanRewardEnum) (*models.Reward, error)
	AdaNilaiRewardAktif(bankID string, rewardID int) (bool, error)
	FindRekeningBank(tx *gorm.DB, bankID string, rewardID int) (*models.SaldoRekening, error)
	LockRekeningBank(tx *gorm.DB, bankID string, rewardID int) (*models.SaldoRekening, error)
	FindRekeningNasabah(nasabahID string, rewardID int) (*models.SaldoRekening, error)
	CreateRekening(tx *gorm.DB, rekening *models.SaldoRekening) error
	ListArusSaldo(rekeningID string, start, end *time.Time) ([]models.RiwayatArusSaldo, error)
	FindAdminIDByUserBank(userID, bankID string) (string, error)
}

type AksesDashboardInput struct {
	ActorUserID string
	ActorBankID string
	ActorRole   models.RoleAdmin
}

type FilterMutasiInput struct {
	RewardID  int
	StartDate string
	EndDate   string
}

type CatatMutasiInput struct {
	TipeMutasi  string
	JenisReward string
	Nominal     float64
	Keterangan  string
}

type KasBank struct {
	TotalUang float64  `json:"total_uang"`
	TotalPoin *float64 `json:"total_poin,omitempty"`
}

type DashboardPetugasResult struct {
	NamaBank      string  `json:"nama_bank"`
	PhotoBank     string  `json:"photo_bank"`
	AlamatBank    string  `json:"alamat_bank"`
	NamaBankPusat *string `json:"nama_bank_pusat,omitempty"`
	JumlahNasabah int64   `json:"jumlah_nasabah"`
	JumlahBSU     *int64  `json:"jumlah_bsu,omitempty"`
	JumlahStaff   int64   `json:"jumlah_staff"`
	Kas           KasBank `json:"kas"`
}

type SaldoUangItem struct {
	TotalUang  float64                 `json:"total_uang"`
	SatuanUang models.SatuanRewardEnum `json:"satuan_uang"`
}

type SaldoPoinItem struct {
	TotalPoin  float64                 `json:"total_poin"`
	SatuanPoin models.SatuanRewardEnum `json:"satuan_poin"`
}

type SaldoBankResult struct {
	Uang SaldoUangItem  `json:"uang"`
	Poin *SaldoPoinItem `json:"poin,omitempty"`
}

type SaldoNasabahResult struct {
	Uang SaldoUangItem `json:"uang"`
	Poin SaldoPoinItem `json:"poin"`
}

type MutasiItem struct {
	IsPositive       bool      `json:"is_positive"`
	Nominal          float64   `json:"nominal"`
	Keterangan       string    `json:"keterangan"`
	TanggalTransaksi time.Time `json:"tanggal_transaksi"`
}

type MutasiSaldoResult struct {
	NamaReward   models.RewardEnum       `json:"nama_reward"`
	SatuanReward models.SatuanRewardEnum `json:"satuan_reward"`
	TotalDebit   float64                 `json:"total_debit"`
	TotalKredit  float64                 `json:"total_kredit"`
	MutasiItems  []MutasiItem            `json:"mutasi_items"`
}

type TotalSaldoNasabahResult struct {
	TotalUang float64 `json:"total_uang"`
	TotalPoin float64 `json:"total_poin"`
}

type NasabahSaldoItem struct {
	NasabahID     string            `json:"nasabah_id"`
	NamaNasabah   string            `json:"nama_nasabah"`
	StatusNasabah models.StatusAkun `json:"status_nasabah"`
	SaldoUang     float64           `json:"saldo_uang"`
	SaldoPoin     float64           `json:"saldo_poin"`
}

type DashboardService interface {
	DashboardPetugas(bankID string) (*DashboardPetugasResult, error)
	SaldoBank(bankID string) (*SaldoBankResult, error)
	SaldoNasabah(nasabahID string) (*SaldoNasabahResult, error)
	MutasiBank(bankID string, in FilterMutasiInput) (*MutasiSaldoResult, error)
	MutasiNasabah(nasabahID string, in FilterMutasiInput) (*MutasiSaldoResult, error)
	CatatManualMutasiBank(bankID string, in CatatMutasiInput, akses AksesDashboardInput) error
	TotalSaldoSemuaNasabah(bankID string) (*TotalSaldoNasabahResult, error)
	ListSaldoSemuaNasabah(bankID string, page int) ([]NasabahSaldoItem, *utils.PageMeta, error)
}

type dashboardService struct {
	db   *gorm.DB
	repo DashboardRepoIface
}

func NewDashboardService(db *gorm.DB, repo DashboardRepoIface) DashboardService {
	return &dashboardService{db: db, repo: repo}
}

func (s *dashboardService) DashboardPetugas(bankID string) (*DashboardPetugasResult, error) {
	bank, err := s.ambilBank(bankID)
	if err != nil {
		return nil, err
	}

	switch bank.JenisBank {
	case models.BSI, models.BSM, models.BSU:
	default:
		return nil, errBadRequest("Jenis bank tidak dikenali")
	}

	jumlahNasabah, err := s.repo.CountNasabahAktif(bankID)
	if err != nil {
		return nil, errInternal("Gagal menghitung nasabah: " + err.Error())
	}
	jumlahStaff, err := s.repo.CountAdminAktif(bankID)
	if err != nil {
		return nil, errInternal("Gagal menghitung staff: " + err.Error())
	}

	saldo, err := s.repo.SumSaldoBank(bankID)
	if err != nil {
		return nil, errInternal("Gagal mengambil kas bank: " + err.Error())
	}

	hasil := &DashboardPetugasResult{
		NamaBank:      bank.NamaBank,
		PhotoBank:     bank.PhotoURL,
		AlamatBank:    bank.Alamat,
		JumlahNasabah: jumlahNasabah,
		JumlahStaff:   jumlahStaff,
		Kas:           KasBank{TotalUang: saldo.Uang},
	}

	if bank.JenisBank == models.BSU {
		namaPusat := ""
		if bank.ParentBankID != nil {
			if parent, err := s.repo.FindBank(*bank.ParentBankID); err == nil {
				namaPusat = parent.NamaBank
			}
		}
		hasil.NamaBankPusat = &namaPusat
		return hasil, nil
	}

	totalPoin := saldo.Poin
	hasil.Kas.TotalPoin = &totalPoin

	if bank.JenisBank == models.BSI {
		jumlahBSU, err := s.repo.CountBSU(bankID)
		if err != nil {
			return nil, errInternal("Gagal menghitung BSU: " + err.Error())
		}
		hasil.JumlahBSU = &jumlahBSU
	}
	return hasil, nil
}

func (s *dashboardService) SaldoBank(bankID string) (*SaldoBankResult, error) {
	bank, err := s.ambilBank(bankID)
	if err != nil {
		return nil, err
	}

	saldo, err := s.repo.SumSaldoBank(bankID)
	if err != nil {
		return nil, errInternal("Gagal mengambil saldo bank: " + err.Error())
	}

	hasil := &SaldoBankResult{
		Uang: SaldoUangItem{TotalUang: saldo.Uang, SatuanUang: models.SatuanRewardEnumRp},
	}
	if bank.JenisBank != models.BSU {
		hasil.Poin = &SaldoPoinItem{TotalPoin: saldo.Poin, SatuanPoin: models.SatuanRewardEnumPoin}
	}
	return hasil, nil
}

func (s *dashboardService) SaldoNasabah(nasabahID string) (*SaldoNasabahResult, error) {
	if _, err := s.repo.FindNasabah(nasabahID); err != nil {
		return nil, errNotFound("Nasabah tidak ditemukan")
	}

	saldo, err := s.repo.SumSaldoNasabah(nasabahID)
	if err != nil {
		return nil, errInternal("Gagal mengambil saldo nasabah: " + err.Error())
	}

	return &SaldoNasabahResult{
		Uang: SaldoUangItem{TotalUang: saldo.Uang, SatuanUang: models.SatuanRewardEnumRp},
		Poin: SaldoPoinItem{TotalPoin: saldo.Poin, SatuanPoin: models.SatuanRewardEnumPoin},
	}, nil
}

func (s *dashboardService) TotalSaldoSemuaNasabah(bankID string) (*TotalSaldoNasabahResult, error) {
	if _, err := s.ambilBank(bankID); err != nil {
		return nil, err
	}

	saldo, err := s.repo.SumSaldoSemuaNasabah(bankID)
	if err != nil {
		return nil, errInternal("Gagal mengambil total saldo nasabah: " + err.Error())
	}
	return &TotalSaldoNasabahResult{TotalUang: saldo.Uang, TotalPoin: saldo.Poin}, nil
}

func (s *dashboardService) ListSaldoSemuaNasabah(bankID string, page int) ([]NasabahSaldoItem, *utils.PageMeta, error) {
	if _, err := s.ambilBank(bankID); err != nil {
		return nil, nil, err
	}

	if page < 1 {
		page = 1
	}

	total, err := s.repo.CountNasabahBank(bankID)
	if err != nil {
		return nil, nil, errInternal("Gagal menghitung nasabah: " + err.Error())
	}

	rows, err := s.repo.ListNasabahDenganSaldo(bankID, saldoNasabahPageLimit, utils.Offset(page, saldoNasabahPageLimit))
	if err != nil {
		return nil, nil, errInternal("Gagal mengambil daftar saldo nasabah: " + err.Error())
	}

	items := make([]NasabahSaldoItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, NasabahSaldoItem{
			NasabahID:     row.NasabahID,
			NamaNasabah:   row.NamaNasabah,
			StatusNasabah: row.StatusNasabah,
			SaldoUang:     row.SaldoUang,
			SaldoPoin:     row.SaldoPoin,
		})
	}

	return items, utils.NewPageMeta(page, saldoNasabahPageLimit, int(total)), nil
}

func (s *dashboardService) MutasiBank(bankID string, in FilterMutasiInput) (*MutasiSaldoResult, error) {
	if _, err := s.ambilBank(bankID); err != nil {
		return nil, err
	}

	reward, err := s.resolveReward(in.RewardID)
	if err != nil {
		return nil, err
	}

	rekening, err := s.repo.FindRekeningBank(nil, bankID, reward.RewardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return mutasiKosong(reward), nil
		}
		return nil, errInternal("Gagal mencari rekening bank: " + err.Error())
	}

	return s.susunMutasiRekening(reward, rekening.RekeningID, in)
}

func (s *dashboardService) MutasiNasabah(nasabahID string, in FilterMutasiInput) (*MutasiSaldoResult, error) {
	if _, err := s.repo.FindNasabah(nasabahID); err != nil {
		return nil, errNotFound("Nasabah tidak ditemukan")
	}

	reward, err := s.resolveReward(in.RewardID)
	if err != nil {
		return nil, err
	}

	rekening, err := s.repo.FindRekeningNasabah(nasabahID, reward.RewardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return mutasiKosong(reward), nil
		}
		return nil, errInternal("Gagal mencari rekening nasabah: " + err.Error())
	}

	return s.susunMutasiRekening(reward, rekening.RekeningID, in)
}

func (s *dashboardService) CatatManualMutasiBank(bankID string, in CatatMutasiInput, akses AksesDashboardInput) error {
	bank, err := s.ambilBank(bankID)
	if err != nil {
		return err
	}

	if in.TipeMutasi != "debit" && in.TipeMutasi != "kredit" {
		return errBadRequest("tipe_mutasi harus salah satu dari: debit, kredit")
	}
	if in.JenisReward != "uang" && in.JenisReward != "poin" {
		return errBadRequest("jenis_reward harus salah satu dari: uang, poin")
	}
	if in.Nominal <= 0 {
		return errBadRequest("nominal harus lebih besar dari 0")
	}

	if in.JenisReward == "poin" && bank.JenisBank == models.BSU {
		return errBadRequest("BSU tidak mengelola saldo poin")
	}

	namaReward, satuan := models.RewardEnumUang, models.SatuanRewardEnumRp
	if in.JenisReward == "poin" {
		namaReward, satuan = models.RewardEnumBarang, models.SatuanRewardEnumPoin
	}

	reward, err := s.repo.FindRewardByJenis(namaReward, satuan)
	if err != nil {
		return errNotFound("Jenis reward tidak ditemukan")
	}

	if in.TipeMutasi == "kredit" {
		if err := s.pastikanRewardDisediakan(bank, reward); err != nil {
			return err
		}
	}

	adminID, err := s.adminPemanggil(akses)
	if err != nil {
		return err
	}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		rekening, findErr := s.repo.LockRekeningBank(tx, bankID, reward.RewardID)
		if findErr != nil {
			if !errors.Is(findErr, gorm.ErrRecordNotFound) {
				return errInternal("Gagal mencari rekening bank: " + findErr.Error())
			}
			rekening, err = s.buatRekeningBank(tx, bankID, reward.RewardID, satuan, adminID)
			if err != nil {
				return err
			}
		}

		delta := in.Nominal
		if in.TipeMutasi == "debit" {
			if rekening.NominalSaldo < in.Nominal {
				return errBadRequest("saldo tidak mencukupi")
			}
			delta = -in.Nominal
		}

		if err := repositories.MutasiSaldo(tx, rekening, delta, in.Keterangan, adminID, time.Now()); err != nil {
			return errInternal("Gagal mencatat mutasi: " + err.Error())
		}
		return nil
	})
	if txErr != nil {
		return bungkusErrorTransaksi(txErr, "Gagal mencatat mutasi: ")
	}
	return nil
}

func (s *dashboardService) ambilBank(bankID string) (*models.BankSampah, error) {
	if bankID == "" {
		return nil, errBadRequest("bank_id wajib diisi")
	}
	bank, err := s.repo.FindBank(bankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}
	return bank, nil
}

func (s *dashboardService) pastikanRewardDisediakan(bank *models.BankSampah, reward *models.Reward) error {
	bankKonfigurasi := bank.BankID
	if bank.JenisBank == models.BSU {
		if bank.ParentBankID == nil {
			return errBadRequest("BSU tidak memiliki parent bank")
		}
		bankKonfigurasi = *bank.ParentBankID
	}

	ada, err := s.repo.AdaNilaiRewardAktif(bankKonfigurasi, reward.RewardID)
	if err != nil {
		return errInternal("Gagal memeriksa nilai reward bank: " + err.Error())
	}
	if !ada {
		return errBadRequest("Bank ini belum/tidak lagi menyediakan reward " + string(reward.NamaReward) +
			". Atur nilai reward terlebih dahulu")
	}
	return nil
}

func (s *dashboardService) resolveReward(rewardID int) (*models.Reward, error) {
	if rewardID == 0 {
		reward, err := s.repo.FindRewardByNama(models.RewardEnumUang)
		if err != nil {
			return nil, errNotFound("Jenis reward Uang belum terdaftar")
		}
		return reward, nil
	}

	reward, err := s.repo.FindReward(rewardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNotFound("Jenis reward tidak ditemukan")
		}
		return nil, errInternal("Gagal mencari jenis reward: " + err.Error())
	}
	return reward, nil
}

func (s *dashboardService) susunMutasiRekening(reward *models.Reward, rekeningID string, in FilterMutasiInput) (*MutasiSaldoResult, error) {
	start, end := rentangTanggal(in.StartDate, in.EndDate)

	arusSaldo, err := s.repo.ListArusSaldo(rekeningID, start, end)
	if err != nil {
		return nil, errInternal("Gagal mengambil riwayat arus saldo: " + err.Error())
	}
	return susunMutasi(reward, arusSaldo), nil
}

func susunMutasi(reward *models.Reward, arusSaldo []models.RiwayatArusSaldo) *MutasiSaldoResult {
	hasil := &MutasiSaldoResult{
		NamaReward:   reward.NamaReward,
		SatuanReward: reward.Satuan,
		MutasiItems:  make([]MutasiItem, 0, len(arusSaldo)),
	}

	for _, s := range arusSaldo {
		var nominal float64
		var isPositive bool

		switch {
		case s.NominalSesudah > s.NominalSebelum:
			nominal = s.NominalSesudah - s.NominalSebelum
			isPositive = true
			hasil.TotalKredit += nominal
		case s.NominalSebelum > s.NominalSesudah:
			nominal = s.NominalSebelum - s.NominalSesudah
			hasil.TotalDebit += nominal
		default:
			continue
		}

		hasil.MutasiItems = append(hasil.MutasiItems, MutasiItem{
			IsPositive:       isPositive,
			Nominal:          nominal,
			Keterangan:       s.Keterangan,
			TanggalTransaksi: s.CreatedAt,
		})
	}
	return hasil
}

func mutasiKosong(reward *models.Reward) *MutasiSaldoResult {
	return &MutasiSaldoResult{
		NamaReward:   reward.NamaReward,
		SatuanReward: reward.Satuan,
		MutasiItems:  []MutasiItem{},
	}
}

func (s *dashboardService) buatRekeningBank(tx *gorm.DB, bankID string, rewardID int, satuan models.SatuanRewardEnum, adminID string) (*models.SaldoRekening, error) {
	rekeningID, err := idgen.GenerateRekeningID(rewardID, bankID)
	if err != nil {
		return nil, errInternal("Gagal generate rekening ID: " + err.Error())
	}

	rekening := &models.SaldoRekening{
		RekeningID:         rekeningID,
		BankID:             &bankID,
		RewardID:           &rewardID,
		Entitas:            models.EntitasBankSampah,
		NominalSaldo:       0,
		SatuanNominalSaldo: satuan,
		LastUpdatedBy:      &adminID,
	}
	if err := s.repo.CreateRekening(tx, rekening); err != nil {
		return nil, errInternal("Gagal membuat rekening bank: " + err.Error())
	}
	return rekening, nil
}

func (s *dashboardService) adminPemanggil(akses AksesDashboardInput) (string, error) {
	adminID, err := s.repo.FindAdminIDByUserBank(akses.ActorUserID, akses.ActorBankID)
	if err != nil {
		return "", errInternal("Gagal mencari data admin: " + err.Error())
	}
	if adminID == "" {
		return "", errForbidden("Akun admin aktif untuk bank ini tidak ditemukan")
	}
	return adminID, nil
}
