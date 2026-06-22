package mocks

import (
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockSetoranRepo struct {
	mock.Mock
}

func (m *MockSetoranRepo) FindNasabahAktifWithUser(nasabahID string) (*models.Nasabah, error) {
	args := m.Called(nasabahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Nasabah), args.Error(1)
}

func (m *MockSetoranRepo) FindNasabahWithUser(nasabahID string) (*models.Nasabah, error) {
	args := m.Called(nasabahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Nasabah), args.Error(1)
}

func (m *MockSetoranRepo) FindAdminAktif(adminID string) (*models.Admin, error) {
	args := m.Called(adminID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Admin), args.Error(1)
}

func (m *MockSetoranRepo) FindPenimbanganAktif(penimbanganID string) (*models.Penimbangan, error) {
	args := m.Called(penimbanganID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Penimbangan), args.Error(1)
}

func (m *MockSetoranRepo) FindKatalogSampah(sampahID string) (*models.KatalogSampah, error) {
	args := m.Called(sampahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KatalogSampah), args.Error(1)
}

func (m *MockSetoranRepo) FindBankSampah(bankID string) (*models.BankSampah, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockSetoranRepo) GetDetailHeader(setoranID string) (*repositories.SetoranDetailHeader, error) {
	args := m.Called(setoranID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.SetoranDetailHeader), args.Error(1)
}

func (m *MockSetoranRepo) GetDetailItems(setoranID string) ([]repositories.SetoranDetailItem, error) {
	args := m.Called(setoranID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.SetoranDetailItem), args.Error(1)
}

func (m *MockSetoranRepo) GetRiwayat(nasabahID, startDate, endDate string) ([]repositories.RiwayatSetoranRow, error) {
	args := m.Called(nasabahID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.RiwayatSetoranRow), args.Error(1)
}

func (m *MockSetoranRepo) FindPenimbanganTx(tx *gorm.DB, penimbanganID string) (*models.Penimbangan, error) {
	args := m.Called(tx, penimbanganID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Penimbangan), args.Error(1)
}

func (m *MockSetoranRepo) CreateSetoran(tx *gorm.DB, s *models.SetoranNasabah) error {
	args := m.Called(tx, s)
	return args.Error(0)
}

func (m *MockSetoranRepo) CreateDetailSetoran(tx *gorm.DB, d *models.DetailSetoranNasabah) error {
	args := m.Called(tx, d)
	return args.Error(0)
}

func (m *MockSetoranRepo) UpsertStokSampah(tx *gorm.DB, bankID, sampahID string, qty float64) error {
	args := m.Called(tx, bankID, sampahID, qty)
	return args.Error(0)
}

func (m *MockSetoranRepo) CreateTabunganSampah(tx *gorm.DB, t *models.TabunganSampah) error {
	args := m.Called(tx, t)
	return args.Error(0)
}
