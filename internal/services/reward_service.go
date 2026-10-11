package services

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"

	"gorm.io/gorm"
)

const persenEksternal = 100

type RewardRepoIface interface {
	ListRewards() ([]models.Reward, error)
	FindReward(tx *gorm.DB, rewardID int) (*models.Reward, error)
	FindRewardByJenis(nama models.RewardEnum, satuan models.SatuanRewardEnum, exceptID int) (*models.Reward, error)
	CreateReward(tx *gorm.DB, reward *models.Reward) error
	SaveReward(tx *gorm.DB, reward *models.Reward) error
	DeleteReward(tx *gorm.DB, reward *models.Reward) error
	HitungPemakaianReward(rewardID int) ([]repositories.PemakaianReward, error)
	FindBank(bankID string) (*models.BankSampah, error)
	ListNilaiRewardByBank(bankID string) ([]models.NilaiRewardBank, error)
	ListNilaiRewardByBankReward(bankID string, rewardID int) ([]models.NilaiRewardBank, error)
	FindNilaiReward(tx *gorm.DB, bankID string, rewardID int, level models.LevelUser) (*models.NilaiRewardBank, error)
	FindNilaiRewardByID(nilaiRewardID string) (*models.NilaiRewardBank, error)
	CreateNilaiReward(tx *gorm.DB, row *models.NilaiRewardBank) error
	SaveNilaiReward(tx *gorm.DB, row *models.NilaiRewardBank) error
	NonaktifkanNilaiRewardPasangan(tx *gorm.DB, bankID string, rewardID int, adminID string) (int64, error)
	ListHistoryByNilaiReward(nilaiRewardIDs []string) ([]repositories.HistoryNilaiRewardRow, error)
	CreateHistory(tx *gorm.DB, history *models.HistoryNilaiReward) error
	FindAdminIDByUserBank(userID, bankID string) (string, error)
}

type AksesRewardInput struct {
	ActorUserID string
	ActorBankID string
	ActorRole   models.RoleAdmin
}

type RewardInput struct {
	NamaReward models.RewardEnum
	Satuan     models.SatuanRewardEnum
	Deskripsi  *string
}

type PersentaseInput struct {
	LevelUser       models.LevelUser `json:"level_user"`
	PersenBagiHasil float64          `json:"persen_bagi_hasil"`
}

type NilaiRewardInput struct {
	RewardID   int
	Persentase []PersentaseInput
}

type RewardRow struct {
	RewardID   int                     `json:"reward_id"`
	NamaReward models.RewardEnum       `json:"nama_reward"`
	Satuan     models.SatuanRewardEnum `json:"satuan"`
	Deskripsi  *string                 `json:"deskripsi"`
	CreatedAt  time.Time               `json:"created_at"`
	UpdatedAt  time.Time               `json:"updated_at"`
}

type NilaiRewardRow struct {
	NilaiRewardID   string           `json:"nilai_reward_id"`
	BankID          string           `json:"bank_id"`
	RewardID        int              `json:"reward_id"`
	LevelUser       models.LevelUser `json:"level_user"`
	PersenBagiHasil float64          `json:"persen_bagi_hasil"`
	IsActive        bool             `json:"is_active"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	CreatedBy       *string          `json:"created_by"`
	UpdatedBy       *string          `json:"updated_by"`
	Reward          *RewardRow       `json:"reward"`
}

type PersentaseLevelRow struct {
	LevelUser       models.LevelUser `json:"level_user"`
	PersenBagiHasil float64          `json:"persen_bagi_hasil"`
}

type DetailNilaiRewardResult struct {
	BankID     string                               `json:"bank_id"`
	RewardID   int                                  `json:"reward_id"`
	NamaReward models.RewardEnum                    `json:"nama_reward"`
	Satuan     models.SatuanRewardEnum              `json:"satuan"`
	Persentase []PersentaseLevelRow                 `json:"persentase"`
	History    []repositories.HistoryNilaiRewardRow `json:"history"`
}

type RewardService interface {
	ListReward() ([]RewardRow, error)
	TambahReward(in RewardInput) (*RewardRow, error)
	UbahReward(rewardIDParam string, in RewardInput) (*RewardRow, error)
	HapusReward(rewardIDParam string) error

	ListNilaiRewardBank(bankID string) ([]NilaiRewardRow, error)
	DetailNilaiReward(nilaiRewardID string, akses AksesRewardInput) (*DetailNilaiRewardResult, error)
	TambahNilaiReward(bankID string, in NilaiRewardInput, akses AksesRewardInput) ([]NilaiRewardRow, error)
	UbahNilaiReward(bankID, rewardIDParam string, in NilaiRewardInput, akses AksesRewardInput) ([]NilaiRewardRow, error)
	HistoryNilaiReward(nilaiRewardID string, akses AksesRewardInput) ([]repositories.HistoryNilaiRewardRow, error)
	HapusNilaiReward(nilaiRewardID string, akses AksesRewardInput) error
}

type rewardService struct {
	db   *gorm.DB
	repo RewardRepoIface
}

func NewRewardService(db *gorm.DB, repo RewardRepoIface) RewardService {
	return &rewardService{db: db, repo: repo}
}

func (s *rewardService) ListReward() ([]RewardRow, error) {
	rewards, err := s.repo.ListRewards()
	if err != nil {
		return nil, errInternal("Gagal mengambil data reward: " + err.Error())
	}

	rows := make([]RewardRow, 0, len(rewards))
	for _, reward := range rewards {
		rows = append(rows, toRewardRow(reward))
	}
	return rows, nil
}

func (s *rewardService) TambahReward(in RewardInput) (*RewardRow, error) {
	if err := validasiJenisReward(in.NamaReward, in.Satuan); err != nil {
		return nil, err
	}

	if err := s.pastikanJenisRewardBelumAda(in.NamaReward, in.Satuan, 0); err != nil {
		return nil, err
	}

	reward := models.Reward{
		NamaReward: in.NamaReward,
		Satuan:     in.Satuan,
		Deskripsi:  in.Deskripsi,
	}
	if err := s.repo.CreateReward(nil, &reward); err != nil {
		return nil, errInternal("Gagal menambahkan data reward: " + err.Error())
	}

	row := toRewardRow(reward)
	return &row, nil
}

func (s *rewardService) UbahReward(rewardIDParam string, in RewardInput) (*RewardRow, error) {
	rewardID, err := parseRewardID(rewardIDParam)
	if err != nil {
		return nil, err
	}

	reward, err := s.repo.FindReward(nil, rewardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNotFound("Data reward tidak ditemukan")
		}
		return nil, errInternal("Gagal mencari data reward: " + err.Error())
	}

	if err := validasiJenisReward(in.NamaReward, in.Satuan); err != nil {
		return nil, err
	}

	jenisBerubah := in.NamaReward != reward.NamaReward || in.Satuan != reward.Satuan
	if jenisBerubah {
		pemakaian, err := s.repo.HitungPemakaianReward(rewardID)
		if err != nil {
			return nil, errInternal("Gagal memeriksa pemakaian reward: " + err.Error())
		}
		if len(pemakaian) > 0 {
			return nil, errConflict(
				"Jenis dan satuan reward tidak dapat diubah karena reward ini sudah dipakai "+
					pesanPemakaianReward(pemakaian)+". Hanya deskripsi yang bisa diubah", nil)
		}
		if err := s.pastikanJenisRewardBelumAda(in.NamaReward, in.Satuan, rewardID); err != nil {
			return nil, err
		}
	}

	reward.NamaReward = in.NamaReward
	reward.Satuan = in.Satuan
	reward.Deskripsi = in.Deskripsi

	if err := s.repo.SaveReward(nil, reward); err != nil {
		return nil, errInternal("Gagal mengupdate data reward: " + err.Error())
	}

	row := toRewardRow(*reward)
	return &row, nil
}

func (s *rewardService) HapusReward(rewardIDParam string) error {
	rewardID, err := parseRewardID(rewardIDParam)
	if err != nil {
		return err
	}

	reward, err := s.repo.FindReward(nil, rewardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errNotFound("Data reward tidak ditemukan")
		}
		return errInternal("Gagal mencari data reward: " + err.Error())
	}

	pemakaian, err := s.repo.HitungPemakaianReward(rewardID)
	if err != nil {
		return errInternal("Gagal memeriksa pemakaian reward: " + err.Error())
	}
	if len(pemakaian) > 0 {
		return errConflict("Reward tidak dapat dihapus karena masih dipakai "+pesanPemakaianReward(pemakaian), nil)
	}

	if err := s.repo.DeleteReward(nil, reward); err != nil {
		return errInternal("Gagal menghapus data reward: " + err.Error())
	}
	return nil
}

func (s *rewardService) ListNilaiRewardBank(bankID string) ([]NilaiRewardRow, error) {
	targetBankID, err := s.resolveBankNilaiReward(bankID)
	if err != nil {
		return nil, err
	}

	rows, err := s.repo.ListNilaiRewardByBank(targetBankID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data nilai reward: " + err.Error())
	}
	return toNilaiRewardRows(rows), nil
}

func (s *rewardService) DetailNilaiReward(nilaiRewardID string, akses AksesRewardInput) (*DetailNilaiRewardResult, error) {
	awal, err := s.ambilNilaiReward(nilaiRewardID)
	if err != nil {
		return nil, err
	}
	if err := s.pastikanAksesBaris(awal, akses, true); err != nil {
		return nil, err
	}

	semuaLevel, err := s.repo.ListNilaiRewardByBankReward(awal.BankID, awal.RewardID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data level: " + err.Error())
	}

	ids := make([]string, 0, len(semuaLevel))
	persentase := make([]PersentaseLevelRow, 0, len(semuaLevel))
	for _, level := range semuaLevel {
		ids = append(ids, level.NilaiRewardID)
		persentase = append(persentase, PersentaseLevelRow{
			LevelUser:       level.LevelUser,
			PersenBagiHasil: level.PersenBagiHasil,
		})
	}

	history, err := s.repo.ListHistoryByNilaiReward(ids)
	if err != nil {
		return nil, errInternal("Gagal mengambil history: " + err.Error())
	}

	return &DetailNilaiRewardResult{
		BankID:     awal.BankID,
		RewardID:   awal.RewardID,
		NamaReward: awal.Reward.NamaReward,
		Satuan:     awal.Reward.Satuan,
		Persentase: persentase,
		History:    history,
	}, nil
}

func (s *rewardService) HistoryNilaiReward(nilaiRewardID string, akses AksesRewardInput) ([]repositories.HistoryNilaiRewardRow, error) {
	row, err := s.ambilNilaiReward(nilaiRewardID)
	if err != nil {
		return nil, err
	}
	if err := s.pastikanAksesBaris(row, akses, false); err != nil {
		return nil, err
	}

	history, err := s.repo.ListHistoryByNilaiReward([]string{nilaiRewardID})
	if err != nil {
		return nil, errInternal("Gagal mengambil history nilai reward: " + err.Error())
	}
	return history, nil
}

func (s *rewardService) HapusNilaiReward(nilaiRewardID string, akses AksesRewardInput) error {
	row, err := s.ambilNilaiReward(nilaiRewardID)
	if err != nil {
		return err
	}
	if err := s.pastikanAksesBaris(row, akses, false); err != nil {
		return err
	}
	adminID, err := s.adminPemanggil(akses)
	if err != nil {
		return err
	}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		_, err := s.repo.NonaktifkanNilaiRewardPasangan(tx, row.BankID, row.RewardID, adminID)
		return err
	})
	if txErr != nil {
		return errInternal("Gagal menghapus nilai reward: " + txErr.Error())
	}
	return nil
}

func (s *rewardService) TambahNilaiReward(bankID string, in NilaiRewardInput, akses AksesRewardInput) ([]NilaiRewardRow, error) {
	bank, err := s.repo.FindBank(bankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}
	if bank.JenisBank == models.BSU {
		return nil, errForbidden("BSU tidak diizinkan mengelola nilai reward, nilai mengikuti BSI")
	}

	persentase, err := validasiPersentase(in.Persentase)
	if err != nil {
		return nil, err
	}

	reward, err := s.repo.FindReward(nil, in.RewardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNotFound("Reward tidak ditemukan")
		}
		return nil, errInternal("Gagal mencari data reward: " + err.Error())
	}

	adminID, err := s.adminPemanggil(akses)
	if err != nil {
		return nil, err
	}

	hasil := []models.NilaiRewardBank{}
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		for _, p := range persentase {
			lama, errCari := s.repo.FindNilaiReward(tx, bankID, in.RewardID, p.LevelUser)
			switch {
			case errCari == nil && lama.IsActive:
				return errConflict("Konfigurasi reward untuk bank, tipe reward, dan level user "+
					string(p.LevelUser)+" sudah ada", nil)

			case errCari == nil:
				baris, err := s.aktifkanNilaiReward(tx, lama, p.PersenBagiHasil, adminID)
				if err != nil {
					return err
				}
				baris.Reward = *reward
				hasil = append(hasil, *baris)
				continue

			case !errors.Is(errCari, gorm.ErrRecordNotFound):
				return errInternal("Gagal memeriksa nilai reward: " + errCari.Error())
			}

			baris, err := s.buatNilaiReward(tx, bankID, in.RewardID, p.LevelUser, p.PersenBagiHasil, adminID)
			if err != nil {
				return err
			}
			baris.Reward = *reward
			hasil = append(hasil, *baris)
		}

		eks, errEks := s.repo.FindNilaiReward(tx, bankID, in.RewardID, models.LevelEksternal)
		switch {
		case errors.Is(errEks, gorm.ErrRecordNotFound):
			baris, err := s.buatNilaiReward(tx, bankID, in.RewardID, models.LevelEksternal, persenEksternal, adminID)
			if err != nil {
				return err
			}
			baris.Reward = *reward
			hasil = append(hasil, *baris)

		case errEks != nil:
			return errInternal("Gagal memeriksa nilai reward eksternal: " + errEks.Error())

		case !eks.IsActive:
			baris, err := s.aktifkanNilaiReward(tx, eks, persenEksternal, adminID)
			if err != nil {
				return err
			}
			baris.Reward = *reward
			hasil = append(hasil, *baris)
		}
		return nil
	})
	if txErr != nil {
		return nil, bungkusErrorTransaksi(txErr, "Gagal menambahkan nilai reward: ")
	}

	return toNilaiRewardRows(hasil), nil
}

func (s *rewardService) UbahNilaiReward(bankID, rewardIDParam string, in NilaiRewardInput, akses AksesRewardInput) ([]NilaiRewardRow, error) {
	rewardID, err := parseRewardID(rewardIDParam)
	if err != nil {
		return nil, err
	}

	persentase, err := validasiPersentase(in.Persentase)
	if err != nil {
		return nil, err
	}

	adminID, err := s.adminPemanggil(akses)
	if err != nil {
		return nil, err
	}

	hasil := []models.NilaiRewardBank{}
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		for _, p := range persentase {
			baris, err := s.repo.FindNilaiReward(tx, bankID, rewardID, p.LevelUser)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errNotFound("Data nilai reward untuk level user " + string(p.LevelUser) + " tidak ditemukan")
				}
				return errInternal("Gagal mencari nilai reward: " + err.Error())
			}

			persenLama := baris.PersenBagiHasil

			if p.PersenBagiHasil == persenLama {
				hasil = append(hasil, *baris)
				continue
			}

			baris.PersenBagiHasil = p.PersenBagiHasil
			baris.UpdatedBy = &adminID

			if err := s.repo.SaveNilaiReward(tx, baris); err != nil {
				return errInternal("Gagal mengupdate nilai reward: " + err.Error())
			}

			historyID, genErr := idgen.GenerateHistoryRewardID(tx, baris.NilaiRewardID)
			if genErr != nil {
				return errInternal("Gagal generate history reward ID: " + genErr.Error())
			}
			persenBaru := p.PersenBagiHasil
			history := models.HistoryNilaiReward{
				HistoryRewardID: historyID,
				NilaiRewardID:   baris.NilaiRewardID,
				OldPersen:       &persenLama,
				NewPersen:       &persenBaru,
				ChangedBy:       &adminID,
			}
			if err := s.repo.CreateHistory(tx, &history); err != nil {
				return errInternal("Gagal mencatat history nilai reward: " + err.Error())
			}

			hasil = append(hasil, *baris)
		}
		return nil
	})
	if txErr != nil {
		return nil, bungkusErrorTransaksi(txErr, "Gagal mengupdate nilai reward: ")
	}

	return toNilaiRewardRows(hasil), nil
}

func validasiJenisReward(nama models.RewardEnum, satuan models.SatuanRewardEnum) error {
	switch nama {
	case models.RewardEnumUang, models.RewardEnumBarang:
	default:
		return errBadRequest("nama_reward harus salah satu dari: Uang, Barang")
	}

	switch satuan {
	case models.SatuanRewardEnumRp, models.SatuanRewardEnumPoin:
	default:
		return errBadRequest("satuan harus salah satu dari: Rp, poin")
	}
	return nil
}

func (s *rewardService) pastikanJenisRewardBelumAda(nama models.RewardEnum, satuan models.SatuanRewardEnum, exceptID int) error {
	_, err := s.repo.FindRewardByJenis(nama, satuan, exceptID)
	if err == nil {
		return errConflict("Reward dengan jenis "+string(nama)+" dan satuan "+string(satuan)+" sudah ada", nil)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return errInternal("Gagal memeriksa data reward: " + err.Error())
	}
	return nil
}

func validasiPersentase(list []PersentaseInput) ([]PersentaseInput, error) {
	hasil := make([]PersentaseInput, 0, len(list))
	sudahAda := make(map[models.LevelUser]bool, len(list))

	for _, p := range list {
		switch p.LevelUser {
		case models.LevelEksternal:
			continue
		case models.LevelNasabah:
		default:
			return nil, errBadRequest("level_user harus salah satu dari: nasabah, eksternal")
		}

		if sudahAda[p.LevelUser] {
			return nil, errBadRequest("level_user " + string(p.LevelUser) + " dikirim lebih dari sekali")
		}
		sudahAda[p.LevelUser] = true

		if p.PersenBagiHasil <= 0 || p.PersenBagiHasil > 100 {
			return nil, errBadRequest("persen_bagi_hasil untuk level user " + string(p.LevelUser) + " harus lebih dari 0 dan maksimal 100")
		}

		hasil = append(hasil, p)
	}

	if len(hasil) == 0 {
		return nil, errBadRequest("persentase harus berisi minimal satu level selain eksternal")
	}
	return hasil, nil
}

func (s *rewardService) resolveBankNilaiReward(bankID string) (string, error) {
	bank, err := s.repo.FindBank(bankID)
	if err != nil {
		return "", errNotFound("Bank tidak ditemukan")
	}
	if bank.JenisBank != models.BSU {
		return bankID, nil
	}
	if bank.ParentBankID == nil {
		return "", errBadRequest("BSU tidak memiliki parent bank")
	}
	return *bank.ParentBankID, nil
}

func (s *rewardService) ambilNilaiReward(nilaiRewardID string) (*models.NilaiRewardBank, error) {
	row, err := s.repo.FindNilaiRewardByID(nilaiRewardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNotFound("Data nilai reward tidak ditemukan")
		}
		return nil, errInternal("Gagal mencari data nilai reward: " + err.Error())
	}
	return row, nil
}

func (s *rewardService) pastikanAksesBaris(row *models.NilaiRewardBank, akses AksesRewardInput, izinkanIndukBSI bool) error {
	if akses.ActorBankID == "" || akses.ActorBankID == row.BankID {
		return nil
	}

	if izinkanIndukBSI {
		bank, err := s.repo.FindBank(akses.ActorBankID)
		if err == nil && bank.JenisBank == models.BSU &&
			bank.ParentBankID != nil && *bank.ParentBankID == row.BankID {
			return nil
		}
	}

	return errForbidden("Akses ditolak. Anda tidak memiliki akses ke nilai reward bank ini")
}

func (s *rewardService) adminPemanggil(akses AksesRewardInput) (string, error) {
	adminID, err := s.repo.FindAdminIDByUserBank(akses.ActorUserID, akses.ActorBankID)
	if err != nil {
		return "", errInternal("Gagal mencari data admin: " + err.Error())
	}
	if adminID == "" {
		return "", errForbidden("Akun admin aktif untuk bank ini tidak ditemukan")
	}
	return adminID, nil
}

func (s *rewardService) buatNilaiReward(tx *gorm.DB, bankID string, rewardID int, level models.LevelUser, persen float64, adminID string) (*models.NilaiRewardBank, error) {
	nilaiRewardID, genErr := idgen.GenerateNilaiRewardID(bankID, rewardID, level)
	if genErr != nil {
		return nil, errInternal("Gagal generate nilai reward ID: " + genErr.Error())
	}

	baris := models.NilaiRewardBank{
		NilaiRewardID:   nilaiRewardID,
		BankID:          bankID,
		RewardID:        rewardID,
		LevelUser:       level,
		PersenBagiHasil: persen,
		CreatedBy:       &adminID,
		UpdatedBy:       &adminID,
	}
	if err := s.repo.CreateNilaiReward(tx, &baris); err != nil {
		return nil, errInternal("Gagal menambahkan nilai reward: " + err.Error())
	}
	return &baris, nil
}

func (s *rewardService) aktifkanNilaiReward(tx *gorm.DB, baris *models.NilaiRewardBank, persen float64, adminID string) (*models.NilaiRewardBank, error) {
	persenLama := baris.PersenBagiHasil
	baris.PersenBagiHasil = persen
	baris.IsActive = true
	baris.UpdatedBy = &adminID

	if err := s.repo.SaveNilaiReward(tx, baris); err != nil {
		return nil, errInternal("Gagal mengaktifkan kembali nilai reward: " + err.Error())
	}

	if persen == persenLama {
		return baris, nil
	}

	historyID, genErr := idgen.GenerateHistoryRewardID(tx, baris.NilaiRewardID)
	if genErr != nil {
		return nil, errInternal("Gagal generate history reward ID: " + genErr.Error())
	}
	persenBaru := persen
	history := models.HistoryNilaiReward{
		HistoryRewardID: historyID,
		NilaiRewardID:   baris.NilaiRewardID,
		OldPersen:       &persenLama,
		NewPersen:       &persenBaru,
		ChangedBy:       &adminID,
	}
	if err := s.repo.CreateHistory(tx, &history); err != nil {
		return nil, errInternal("Gagal mencatat history nilai reward: " + err.Error())
	}
	return baris, nil
}

func parseRewardID(param string) (int, error) {
	rewardID, err := strconv.Atoi(param)
	if err != nil || rewardID <= 0 {
		return 0, errBadRequest("reward_id tidak valid")
	}
	return rewardID, nil
}

func pesanPemakaianReward(pemakaian []repositories.PemakaianReward) string {
	bagian := make([]string, 0, len(pemakaian))
	for _, p := range pemakaian {
		bagian = append(bagian, fmt.Sprintf("%d %s", p.Jumlah, p.Label))
	}
	return strings.Join(bagian, ", ")
}

func bungkusErrorTransaksi(err error, prefix string) error {
	if _, ok := err.(*ServiceError); ok {
		return err
	}
	return errInternal(prefix + err.Error())
}

func toRewardRow(reward models.Reward) RewardRow {
	return RewardRow{
		RewardID:   reward.RewardID,
		NamaReward: reward.NamaReward,
		Satuan:     reward.Satuan,
		Deskripsi:  reward.Deskripsi,
		CreatedAt:  reward.CreatedAt,
		UpdatedAt:  reward.UpdatedAt,
	}
}

func toNilaiRewardRows(rows []models.NilaiRewardBank) []NilaiRewardRow {
	hasil := make([]NilaiRewardRow, 0, len(rows))
	for _, row := range rows {
		item := NilaiRewardRow{
			NilaiRewardID:   row.NilaiRewardID,
			BankID:          row.BankID,
			RewardID:        row.RewardID,
			LevelUser:       row.LevelUser,
			PersenBagiHasil: row.PersenBagiHasil,
			IsActive:        row.IsActive,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
			CreatedBy:       row.CreatedBy,
			UpdatedBy:       row.UpdatedBy,
		}
		if row.Reward.RewardID != 0 {
			reward := toRewardRow(row.Reward)
			item.Reward = &reward
		}
		hasil = append(hasil, item)
	}
	return hasil
}
