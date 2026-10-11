package mocks

import (
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockDashboardRepo struct {
	mock.Mock
}

func (m *MockDashboardRepo) FindBank(bankID string) (*models.BankSampah, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockDashboardRepo) FindNasabah(nasabahID string) (*models.Nasabah, error) {
	args := m.Called(nasabahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Nasabah), args.Error(1)
}

func (m *MockDashboardRepo) CountNasabahAktif(bankID string) (int64, error) {
	args := m.Called(bankID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDashboardRepo) CountAdminAktif(bankID string) (int64, error) {
	args := m.Called(bankID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDashboardRepo) CountBSU(bsiID string) (int64, error) {
	args := m.Called(bsiID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDashboardRepo) CountNasabahBank(bankID string) (int64, error) {
	args := m.Called(bankID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDashboardRepo) SumSaldoBank(bankID string) (repositories.SaldoPerSatuan, error) {
	args := m.Called(bankID)
	return args.Get(0).(repositories.SaldoPerSatuan), args.Error(1)
}

func (m *MockDashboardRepo) SumSaldoNasabah(nasabahID string) (repositories.SaldoPerSatuan, error) {
	args := m.Called(nasabahID)
	return args.Get(0).(repositories.SaldoPerSatuan), args.Error(1)
}

func (m *MockDashboardRepo) SumSaldoSemuaNasabah(bankID string) (repositories.SaldoPerSatuan, error) {
	args := m.Called(bankID)
	return args.Get(0).(repositories.SaldoPerSatuan), args.Error(1)
}

func (m *MockDashboardRepo) ListNasabahDenganSaldo(bankID string, limit, offset int) ([]repositories.NasabahSaldoRow, error) {
	args := m.Called(bankID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.NasabahSaldoRow), args.Error(1)
}

func (m *MockDashboardRepo) FindReward(rewardID int) (*models.Reward, error) {
	args := m.Called(rewardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reward), args.Error(1)
}

func (m *MockDashboardRepo) FindRewardByNama(nama models.RewardEnum) (*models.Reward, error) {
	args := m.Called(nama)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reward), args.Error(1)
}

func (m *MockDashboardRepo) FindRewardByJenis(nama models.RewardEnum, satuan models.SatuanRewardEnum) (*models.Reward, error) {
	args := m.Called(nama, satuan)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reward), args.Error(1)
}

func (m *MockDashboardRepo) AdaNilaiRewardAktif(bankID string, rewardID int) (bool, error) {
	args := m.Called(bankID, rewardID)
	return args.Bool(0), args.Error(1)
}

func (m *MockDashboardRepo) FindRekeningBank(tx *gorm.DB, bankID string, rewardID int) (*models.SaldoRekening, error) {
	args := m.Called(tx, bankID, rewardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SaldoRekening), args.Error(1)
}

func (m *MockDashboardRepo) LockRekeningBank(tx *gorm.DB, bankID string, rewardID int) (*models.SaldoRekening, error) {
	args := m.Called(tx, bankID, rewardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SaldoRekening), args.Error(1)
}

func (m *MockDashboardRepo) FindRekeningNasabah(nasabahID string, rewardID int) (*models.SaldoRekening, error) {
	args := m.Called(nasabahID, rewardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SaldoRekening), args.Error(1)
}

func (m *MockDashboardRepo) CreateRekening(tx *gorm.DB, rekening *models.SaldoRekening) error {
	args := m.Called(tx, rekening)
	return args.Error(0)
}

func (m *MockDashboardRepo) ListArusSaldo(rekeningID string, start, end *time.Time) ([]models.RiwayatArusSaldo, error) {
	args := m.Called(rekeningID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.RiwayatArusSaldo), args.Error(1)
}

func (m *MockDashboardRepo) FindAdminIDByUserBank(userID, bankID string) (string, error) {
	args := m.Called(userID, bankID)
	return args.String(0), args.Error(1)
}
