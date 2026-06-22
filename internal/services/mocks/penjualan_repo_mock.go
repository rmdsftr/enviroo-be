package mocks

import (
	"enviroo-be/internal/models"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockPenjualanRepo struct {
	mock.Mock
}

func (m *MockPenjualanRepo) GetBankByID(bankID string) (*models.BankSampah, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockPenjualanRepo) GetRewardByID(db *gorm.DB, rewardID int) (*models.Reward, error) {
	args := m.Called(db, rewardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reward), args.Error(1)
}

func (m *MockPenjualanRepo) GetPersenNasabah(db *gorm.DB, bankID string, rewardID int) float64 {
	args := m.Called(db, bankID, rewardID)
	return args.Get(0).(float64)
}

func (m *MockPenjualanRepo) GetKatalogSampah(db *gorm.DB, sampahID string) (*models.KatalogSampah, error) {
	args := m.Called(db, sampahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KatalogSampah), args.Error(1)
}

func (m *MockPenjualanRepo) GetStokSampah(db *gorm.DB, bankID, sampahID string) (*models.StokSampah, error) {
	args := m.Called(db, bankID, sampahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StokSampah), args.Error(1)
}

func (m *MockPenjualanRepo) GetSchemaHarga(db *gorm.DB, sampahID string, level models.LevelUser) (*models.SchemaHargaSampah, error) {
	args := m.Called(db, sampahID, level)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SchemaHargaSampah), args.Error(1)
}

func (m *MockPenjualanRepo) CreateSchemaHarga(db *gorm.DB, s *models.SchemaHargaSampah) error {
	args := m.Called(db, s)
	return args.Error(0)
}

func (m *MockPenjualanRepo) UpdateSchemaHarga(db *gorm.DB, s *models.SchemaHargaSampah) error {
	args := m.Called(db, s)
	return args.Error(0)
}

func (m *MockPenjualanRepo) CreateHistoryHarga(db *gorm.DB, h *models.KatalogSampahHistory) error {
	args := m.Called(db, h)
	return args.Error(0)
}

func (m *MockPenjualanRepo) DecrStokSampah(db *gorm.DB, bankID, sampahID string, newStok float64) error {
	args := m.Called(db, bankID, sampahID, newStok)
	return args.Error(0)
}

func (m *MockPenjualanRepo) CreatePenjualan(db *gorm.DB, p *models.Penjualan) error {
	args := m.Called(db, p)
	return args.Error(0)
}

func (m *MockPenjualanRepo) CreateDetailPenjualan(db *gorm.DB, details []models.DetailPenjualan) error {
	args := m.Called(db, details)
	return args.Error(0)
}

func (m *MockPenjualanRepo) GetPenjualan(penjualanID string) (*models.Penjualan, error) {
	args := m.Called(penjualanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Penjualan), args.Error(1)
}

func (m *MockPenjualanRepo) GetDetailItems(penjualanID string) ([]models.DetailPenjualan, error) {
	args := m.Called(penjualanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.DetailPenjualan), args.Error(1)
}

func (m *MockPenjualanRepo) GetAdminNama(adminID string) string {
	args := m.Called(adminID)
	return args.String(0)
}

func (m *MockPenjualanRepo) GetDistinctMitra(bankID string) ([]string, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}
