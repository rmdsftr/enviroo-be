package mocks

import (
	"time"

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

func (m *MockPenarikanRepo) FindNasabah(tx *gorm.DB, nasabahID string) (*models.Nasabah, error) {
	args := m.Called(tx, nasabahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Nasabah), args.Error(1)
}

func (m *MockPenarikanRepo) FindNasabahWithUser(nasabahID string) (*models.Nasabah, error) {
	args := m.Called(nasabahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Nasabah), args.Error(1)
}

func (m *MockPenarikanRepo) CountNasabahMilikUser(userID, nasabahID string) (int64, error) {
	args := m.Called(userID, nasabahID)
	return args.Get(0).(int64), args.Error(1)
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

func (m *MockPenarikanRepo) FindProduk(produkID, bankID string) (*models.KatalogBarang, error) {
	args := m.Called(produkID, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KatalogBarang), args.Error(1)
}

func (m *MockPenarikanRepo) FindStokBarang(produkID, bankID string) (*models.StokBarang, error) {
	args := m.Called(produkID, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StokBarang), args.Error(1)
}

func (m *MockPenarikanRepo) FindPenarikanByID(penarikanID string) (*models.Penarikan, error) {
	args := m.Called(penarikanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Penarikan), args.Error(1)
}

func (m *MockPenarikanRepo) FindRiwayatPenarikan(penarikanID string) ([]repositories.RiwayatPenarikanRow, error) {
	args := m.Called(penarikanID)
	return args.Get(0).([]repositories.RiwayatPenarikanRow), args.Error(1)
}

func (m *MockPenarikanRepo) SaveRiwayatPenarikan(tx *gorm.DB, rw *models.RiwayatPenarikan) error {
	args := m.Called(tx, rw)
	return args.Error(0)
}

func (m *MockPenarikanRepo) FindByNasabahID(nasabahID string, f repositories.PenarikanFilter) ([]repositories.PenarikanListRow, error) {
	args := m.Called(nasabahID, f)
	return args.Get(0).([]repositories.PenarikanListRow), args.Error(1)
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

func (m *MockPenarikanRepo) LockProduk(tx *gorm.DB, produkID, bankID string) (*models.KatalogBarang, error) {
	args := m.Called(tx, produkID, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KatalogBarang), args.Error(1)
}

func (m *MockPenarikanRepo) LockStok(tx *gorm.DB, produkID, bankID string) (*models.StokBarang, error) {
	args := m.Called(tx, produkID, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StokBarang), args.Error(1)
}

func (m *MockPenarikanRepo) LockPenarikan(tx *gorm.DB, penarikanID string) (*models.Penarikan, error) {
	args := m.Called(tx, penarikanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Penarikan), args.Error(1)
}

func (m *MockPenarikanRepo) FindAdminAktif(tx *gorm.DB, userID string) (*models.Admin, error) {
	args := m.Called(tx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Admin), args.Error(1)
}

func (m *MockPenarikanRepo) FindDetailBarang(tx *gorm.DB, penarikanID string) ([]models.DetailPenarikanBarang, error) {
	args := m.Called(tx, penarikanID)
	return args.Get(0).([]models.DetailPenarikanBarang), args.Error(1)
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

func (m *MockPenarikanRepo) SaveDetailBarang(tx *gorm.DB, d *models.DetailPenarikanBarang) error {
	args := m.Called(tx, d)
	return args.Error(0)
}

func (m *MockPenarikanRepo) UpdateStok(tx *gorm.DB, produkID, bankID string, delta float64) error {
	args := m.Called(tx, produkID, bankID, delta)
	return args.Error(0)
}

func (m *MockPenarikanRepo) UpdateStatus(tx *gorm.DB, p *models.Penarikan, updates map[string]interface{}) error {
	args := m.Called(tx, p, updates)
	return args.Error(0)
}

func (m *MockPenarikanRepo) FindSettingPenarikan(bankID string) (*models.SettingPenarikan, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SettingPenarikan), args.Error(1)
}

func (m *MockPenarikanRepo) CreateSettingPenarikan(s *models.SettingPenarikan) error {
	args := m.Called(s)
	return args.Error(0)
}

func (m *MockPenarikanRepo) SaveSettingPenarikan(s *models.SettingPenarikan) error {
	args := m.Called(s)
	return args.Error(0)
}

func (m *MockPenarikanRepo) FindTanggalSetoranPertama(nasabahID string) (*time.Time, error) {
	args := m.Called(nasabahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*time.Time), args.Error(1)
}

func (m *MockPenarikanRepo) PunyaSaldoAwal(nasabahID string) (bool, error) {
	args := m.Called(nasabahID)
	return args.Bool(0), args.Error(1)
}
