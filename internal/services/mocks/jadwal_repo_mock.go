package mocks

import (
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockJadwalRepo struct {
	mock.Mock
}

func (m *MockJadwalRepo) FindBank(bankID string) (*models.BankSampah, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockJadwalRepo) FindJadwal(tx *gorm.DB, jadwalID string) (*models.Jadwal, error) {
	args := m.Called(tx, jadwalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Jadwal), args.Error(1)
}

func (m *MockJadwalRepo) ListJadwalBank(bankID string, jenis models.JadwalEnum) ([]models.Jadwal, error) {
	args := m.Called(bankID, jenis)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Jadwal), args.Error(1)
}

func (m *MockJadwalRepo) ListJadwalBankWithNama(bankID string, jenis models.JadwalEnum, sebagaiTarget bool, month, year int) ([]repositories.JadwalBankRow, error) {
	args := m.Called(bankID, jenis, sebagaiTarget, month, year)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.JadwalBankRow), args.Error(1)
}

func (m *MockJadwalRepo) ListSemuaJadwal(jenis models.JadwalEnum) ([]repositories.JadwalLengkapRow, error) {
	args := m.Called(jenis)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.JadwalLengkapRow), args.Error(1)
}

func (m *MockJadwalRepo) ListStatusSesi(jadwalIDs []string, dari, sampai time.Time) (map[string]string, error) {
	args := m.Called(jadwalIDs, dari, sampai)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockJadwalRepo) CreateJadwal(tx *gorm.DB, jadwal *models.Jadwal) error {
	args := m.Called(tx, jadwal)
	return args.Error(0)
}

func (m *MockJadwalRepo) SaveJadwal(tx *gorm.DB, jadwal *models.Jadwal) error {
	args := m.Called(tx, jadwal)
	return args.Error(0)
}

func (m *MockJadwalRepo) DeleteJadwal(tx *gorm.DB, jadwal *models.Jadwal) error {
	args := m.Called(tx, jadwal)
	return args.Error(0)
}

func (m *MockJadwalRepo) CountPemakaianJadwal(jadwalID string) ([]repositories.PemakaianJadwal, error) {
	args := m.Called(jadwalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.PemakaianJadwal), args.Error(1)
}

func (m *MockJadwalRepo) CountSesiPenimbangan(jadwalID string) (int64, error) {
	args := m.Called(jadwalID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockJadwalRepo) FindAdminIDByUserBank(userID, bankID string) (string, error) {
	args := m.Called(userID, bankID)
	return args.String(0), args.Error(1)
}
