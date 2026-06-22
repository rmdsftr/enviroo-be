package mocks

import (
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockPenarikanRepo struct {
	mock.Mock
}

func (m *MockPenarikanRepo) FindNasabahAktif(nasabahID string) (*models.Nasabah, error) {
	args := m.Called(nasabahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Nasabah), args.Error(1)
}

func (m *MockPenarikanRepo) FindNasabahByUserID(userID string) (*models.Nasabah, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Nasabah), args.Error(1)
}

func (m *MockPenarikanRepo) FindReward(rewardID int) (*models.Reward, error) {
	args := m.Called(rewardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reward), args.Error(1)
}

func (m *MockPenarikanRepo) FindSaldoRekening(nasabahID string, rewardID int) (*models.SaldoRekening, error) {
	args := m.Called(nasabahID, rewardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SaldoRekening), args.Error(1)
}

func (m *MockPenarikanRepo) FindSembako(sembakoID, bankID string) (*models.KatalogSembako, error) {
	args := m.Called(sembakoID, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KatalogSembako), args.Error(1)
}

func (m *MockPenarikanRepo) FindStokSembako(sembakoID, bankID string) (*models.StokSembako, error) {
	args := m.Called(sembakoID, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StokSembako), args.Error(1)
}

func (m *MockPenarikanRepo) FindPenarikanByID(penarikanID string) (*models.Penarikan, error) {
	args := m.Called(penarikanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Penarikan), args.Error(1)
}

func (m *MockPenarikanRepo) FindByNasabahID(nasabahID string, f repositories.PenarikanFilter) ([]models.Penarikan, error) {
	args := m.Called(nasabahID, f)
	return args.Get(0).([]models.Penarikan), args.Error(1)
}

func (m *MockPenarikanRepo) FindByBankID(bankID string, f repositories.PenarikanFilter) ([]models.Penarikan, error) {
	args := m.Called(bankID, f)
	return args.Get(0).([]models.Penarikan), args.Error(1)
}

func (m *MockPenarikanRepo) FindAdminsByBankID(bankID string) ([]repositories.AdminInfo, error) {
	args := m.Called(bankID)
	return args.Get(0).([]repositories.AdminInfo), args.Error(1)
}

func (m *MockPenarikanRepo) LockSaldo(tx *gorm.DB, nasabahID string, rewardID int) (*models.SaldoRekening, error) {
	args := m.Called(tx, nasabahID, rewardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SaldoRekening), args.Error(1)
}

func (m *MockPenarikanRepo) LockSembako(tx *gorm.DB, sembakoID, bankID string) (*models.KatalogSembako, error) {
	args := m.Called(tx, sembakoID, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KatalogSembako), args.Error(1)
}

func (m *MockPenarikanRepo) LockStok(tx *gorm.DB, sembakoID, bankID string) (*models.StokSembako, error) {
	args := m.Called(tx, sembakoID, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StokSembako), args.Error(1)
}

func (m *MockPenarikanRepo) LockPenarikan(tx *gorm.DB, penarikanID string) (*models.Penarikan, error) {
	args := m.Called(tx, penarikanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Penarikan), args.Error(1)
}

func (m *MockPenarikanRepo) FindAdminTx(tx *gorm.DB, userID string) (*models.Admin, error) {
	args := m.Called(tx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Admin), args.Error(1)
}

func (m *MockPenarikanRepo) FindDetailSembako(tx *gorm.DB, penarikanID string) ([]models.DetailPenarikanSembako, error) {
	args := m.Called(tx, penarikanID)
	return args.Get(0).([]models.DetailPenarikanSembako), args.Error(1)
}

func (m *MockPenarikanRepo) UpdateSaldo(tx *gorm.DB, saldo *models.SaldoRekening, nominal float64) error {
	args := m.Called(tx, saldo, nominal)
	return args.Error(0)
}

func (m *MockPenarikanRepo) CreateRiwayat(tx *gorm.DB, rw *models.RiwayatArusSaldo) error {
	args := m.Called(tx, rw)
	return args.Error(0)
}

func (m *MockPenarikanRepo) SavePenarikan(tx *gorm.DB, p *models.Penarikan) error {
	args := m.Called(tx, p)
	return args.Error(0)
}

func (m *MockPenarikanRepo) SaveDetailSembako(tx *gorm.DB, d *models.DetailPenarikanSembako) error {
	args := m.Called(tx, d)
	return args.Error(0)
}

func (m *MockPenarikanRepo) UpdateStok(tx *gorm.DB, sembakoID, bankID string, delta float64) error {
	args := m.Called(tx, sembakoID, bankID, delta)
	return args.Error(0)
}

func (m *MockPenarikanRepo) UpdateStatus(tx *gorm.DB, p *models.Penarikan, updates map[string]interface{}) error {
	args := m.Called(tx, p, updates)
	return args.Error(0)
}
