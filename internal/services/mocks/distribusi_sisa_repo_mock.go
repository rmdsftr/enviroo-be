package mocks

import (
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"time"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockDistribusiSisaRepo struct {
	mock.Mock
}

func (m *MockDistribusiSisaRepo) GetBankByID(bankID string) (*models.BankSampah, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockDistribusiSisaRepo) GetBankByIDAndJenis(bankID string, jenis []models.JenisBank) (*models.BankSampah, error) {
	args := m.Called(bankID, jenis)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockDistribusiSisaRepo) GetSetting(bankID string) (*models.SettingSisaBagiHasil, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SettingSisaBagiHasil), args.Error(1)
}

func (m *MockDistribusiSisaRepo) CreateSetting(s *models.SettingSisaBagiHasil) error {
	args := m.Called(s)
	return args.Error(0)
}

func (m *MockDistribusiSisaRepo) SaveSetting(s *models.SettingSisaBagiHasil) error {
	args := m.Called(s)
	return args.Error(0)
}

func (m *MockDistribusiSisaRepo) GetBagiHasilWithBank(bagiHasilID string) (*models.BagiHasil, error) {
	args := m.Called(bagiHasilID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BagiHasil), args.Error(1)
}

func (m *MockDistribusiSisaRepo) FindDistribusiByBagiHasil(bagiHasilID string) (*models.DistribusiSisa, bool) {
	args := m.Called(bagiHasilID)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*models.DistribusiSisa), args.Bool(1)
}

func (m *MockDistribusiSisaRepo) GetBSUList(db *gorm.DB, bsiID string) ([]models.BankSampah, error) {
	args := m.Called(db, bsiID)
	return args.Get(0).([]models.BankSampah), args.Error(1)
}

func (m *MockDistribusiSisaRepo) GetRewardByNama(nama models.RewardEnum) (*models.Reward, error) {
	args := m.Called(nama)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reward), args.Error(1)
}

func (m *MockDistribusiSisaRepo) GetDetailPenjualan(db *gorm.DB, penjualanID string) ([]models.DetailPenjualan, error) {
	args := m.Called(db, penjualanID)
	return args.Get(0).([]models.DetailPenjualan), args.Error(1)
}

func (m *MockDistribusiSisaRepo) GetKontribusiPerBSU(db *gorm.DB, bagiHasilID string) ([]repositories.KontribusiBSURow, error) {
	args := m.Called(db, bagiHasilID)
	return args.Get(0).([]repositories.KontribusiBSURow), args.Error(1)
}

func (m *MockDistribusiSisaRepo) GetTransportSplitPerBSU(db *gorm.DB, bagiHasilID string) ([]repositories.TransportSplitRow, error) {
	args := m.Called(db, bagiHasilID)
	return args.Get(0).([]repositories.TransportSplitRow), args.Error(1)
}

func (m *MockDistribusiSisaRepo) CreateDistribusiSisa(db *gorm.DB, d *models.DistribusiSisa) error {
	args := m.Called(db, d)
	return args.Error(0)
}

func (m *MockDistribusiSisaRepo) CreatePenerimaSisa(db *gorm.DB, p *models.PenerimaDistribusiSisa) error {
	args := m.Called(db, p)
	return args.Error(0)
}

func (m *MockDistribusiSisaRepo) UpdateSaldoRekening(db *gorm.DB, bankID *string, nasabahID *string, rewardID int, reward models.Reward, nilai float64, entitas models.EntitasEnum, adminID string, now time.Time) error {
	args := m.Called(db, bankID, nasabahID, rewardID, reward, nilai, entitas, adminID, now)
	return args.Error(0)
}

func (m *MockDistribusiSisaRepo) SetPenerimaSisaID(db *gorm.DB, bagiHasilID, bsuID, penerimaSisaID string) error {
	args := m.Called(db, bagiHasilID, bsuID, penerimaSisaID)
	return args.Error(0)
}

func (m *MockDistribusiSisaRepo) GetAdminsByBankID(bankID string) ([]models.Admin, error) {
	args := m.Called(bankID)
	return args.Get(0).([]models.Admin), args.Error(1)
}

func (m *MockDistribusiSisaRepo) GetDistribusiSisa(distribusiID string) (*models.DistribusiSisa, error) {
	args := m.Called(distribusiID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DistribusiSisa), args.Error(1)
}

func (m *MockDistribusiSisaRepo) GetAdminNama(adminID string) string {
	args := m.Called(adminID)
	return args.String(0)
}

func (m *MockDistribusiSisaRepo) GetPenerimaBSI(distribusiID string) (*models.PenerimaDistribusiSisa, error) {
	args := m.Called(distribusiID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PenerimaDistribusiSisa), args.Error(1)
}

func (m *MockDistribusiSisaRepo) GetListPenerimaBSU(distribusiID string) ([]models.PenerimaDistribusiSisa, error) {
	args := m.Called(distribusiID)
	return args.Get(0).([]models.PenerimaDistribusiSisa), args.Error(1)
}

func (m *MockDistribusiSisaRepo) GetPenerimaByBank(bankID string, jenis models.JenisTerimaEnum, startDate, endDate time.Time) ([]models.PenerimaDistribusiSisa, error) {
	args := m.Called(bankID, jenis, startDate, endDate)
	return args.Get(0).([]models.PenerimaDistribusiSisa), args.Error(1)
}

func (m *MockDistribusiSisaRepo) GetPenerimaSisaWithDetails(penerimaSisaID string) (*models.PenerimaDistribusiSisa, error) {
	args := m.Called(penerimaSisaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PenerimaDistribusiSisa), args.Error(1)
}

func (m *MockDistribusiSisaRepo) GetPerhitunganSisa(penerimaSisaID string) ([]models.PerhitunganSisa, error) {
	args := m.Called(penerimaSisaID)
	return args.Get(0).([]models.PerhitunganSisa), args.Error(1)
}
