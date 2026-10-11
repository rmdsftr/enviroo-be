package mocks

import (
	"enviroo-be/internal/models"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockMitraRepo struct {
	mock.Mock
}

func (m *MockMitraRepo) GetBankByID(bankID string) (*models.BankSampah, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockMitraRepo) ListByBank(bankID string) ([]models.MitraPengepul, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.MitraPengepul), args.Error(1)
}

func (m *MockMitraRepo) GetByID(db *gorm.DB, bankID, mitraID string) (*models.MitraPengepul, error) {
	args := m.Called(db, bankID, mitraID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MitraPengepul), args.Error(1)
}

func (m *MockMitraRepo) FindByNormalizedName(db *gorm.DB, bankID, normalizedName string) (*models.MitraPengepul, error) {
	args := m.Called(db, bankID, normalizedName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MitraPengepul), args.Error(1)
}

func (m *MockMitraRepo) Create(db *gorm.DB, mitra *models.MitraPengepul) error {
	args := m.Called(db, mitra)
	return args.Error(0)
}

func (m *MockMitraRepo) Update(db *gorm.DB, mitra *models.MitraPengepul) error {
	args := m.Called(db, mitra)
	return args.Error(0)
}

func (m *MockMitraRepo) IsUsedInPenjualan(db *gorm.DB, mitraID string) (bool, error) {
	args := m.Called(db, mitraID)
	return args.Bool(0), args.Error(1)
}

func (m *MockMitraRepo) Delete(db *gorm.DB, mitra *models.MitraPengepul) error {
	args := m.Called(db, mitra)
	return args.Error(0)
}
