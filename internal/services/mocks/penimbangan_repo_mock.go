package mocks

import (
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockPenimbanganRepo struct {
	mock.Mock
}

func (m *MockPenimbanganRepo) FindPenimbangan(penimbanganID string) (*models.Penimbangan, error) {
	args := m.Called(penimbanganID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Penimbangan), args.Error(1)
}

func (m *MockPenimbanganRepo) FindPenimbanganLock(tx *gorm.DB, penimbanganID string, status models.StatusPenimbangan) (*models.Penimbangan, error) {
	args := m.Called(tx, penimbanganID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Penimbangan), args.Error(1)
}

func (m *MockPenimbanganRepo) SavePenimbangan(tx *gorm.DB, p *models.Penimbangan) error {
	args := m.Called(tx, p)
	return args.Error(0)
}

func (m *MockPenimbanganRepo) CreatePenimbangan(tx *gorm.DB, p *models.Penimbangan) error {
	args := m.Called(tx, p)
	return args.Error(0)
}

func (m *MockPenimbanganRepo) CountSesiAktif(tx *gorm.DB, bankID string, tanggal string) (int64, error) {
	args := m.Called(tx, bankID, tanggal)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPenimbanganRepo) CountSesiByJadwalTanggal(jadwalID string, tanggal time.Time) (int64, error) {
	args := m.Called(jadwalID, tanggal)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPenimbanganRepo) ListSesiTanggal(bankID, tanggal string, statuses []models.StatusPenimbangan) ([]repositories.SesiHariIniRow, error) {
	args := m.Called(bankID, tanggal, statuses)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.SesiHariIniRow), args.Error(1)
}

func (m *MockPenimbanganRepo) ListRiwayat(bankID string, start, end *time.Time) ([]repositories.RiwayatPenimbanganRow, error) {
	args := m.Called(bankID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.RiwayatPenimbanganRow), args.Error(1)
}

func (m *MockPenimbanganRepo) FindHeaderSesi(penimbanganID string) (*repositories.HeaderSesiRow, error) {
	args := m.Called(penimbanganID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.HeaderSesiRow), args.Error(1)
}

func (m *MockPenimbanganRepo) ListSetoranSesi(penimbanganID string) ([]repositories.SetoranSesiRow, error) {
	args := m.Called(penimbanganID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.SetoranSesiRow), args.Error(1)
}

func (m *MockPenimbanganRepo) TutupSesiTerlewat() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPenimbanganRepo) FindBank(bankID string) (*models.BankSampah, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockPenimbanganRepo) FindJadwal(jadwalID string) (*models.Jadwal, error) {
	args := m.Called(jadwalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Jadwal), args.Error(1)
}

func (m *MockPenimbanganRepo) FindJadwalPenimbangan(jadwalID string) (*models.Jadwal, error) {
	args := m.Called(jadwalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Jadwal), args.Error(1)
}

func (m *MockPenimbanganRepo) ListJadwalJatuhTempo(hari models.HariEnum, mingguKe int, tanggal string) ([]models.Jadwal, error) {
	args := m.Called(hari, mingguKe, tanggal)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Jadwal), args.Error(1)
}

func (m *MockPenimbanganRepo) FindNasabahTargets(bankID string) ([]repositories.NotifTarget, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.NotifTarget), args.Error(1)
}

func (m *MockPenimbanganRepo) FindAdminTargets(bankID string) ([]repositories.NotifTarget, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.NotifTarget), args.Error(1)
}
