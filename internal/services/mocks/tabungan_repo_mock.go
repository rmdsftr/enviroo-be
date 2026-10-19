package mocks

import (
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"

	"github.com/stretchr/testify/mock"
)

type MockTabunganRepo struct {
	mock.Mock
}

func (m *MockTabunganRepo) ListTabunganNasabah(nasabahID string, start, end *time.Time) ([]models.TabunganSampah, error) {
	args := m.Called(nasabahID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TabunganSampah), args.Error(1)
}

func (m *MockTabunganRepo) ListTabunganBank(bankID string, start, end *time.Time) ([]models.TabunganSampah, error) {
	args := m.Called(bankID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.TabunganSampah), args.Error(1)
}

func (m *MockTabunganRepo) ListPencairanTabungan(tabunganIDs []string) ([]repositories.PencairanTabunganRow, error) {
	args := m.Called(tabunganIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.PencairanTabunganRow), args.Error(1)
}

func (m *MockTabunganRepo) ListTanggalPengangkutanCompleted(pengangkutanIDs []string) (map[string]string, error) {
	args := m.Called(pengangkutanIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockTabunganRepo) FindBank(bankID string) (*models.BankSampah, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockTabunganRepo) ListAntrianSampaiNasabah(bankID, nasabahID string) ([]repositories.AntrianSampahRow, error) {
	args := m.Called(bankID, nasabahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.AntrianSampahRow), args.Error(1)
}

func (m *MockTabunganRepo) FindKatalogBank(bankID, sampahID string) (*models.KatalogSampah, error) {
	args := m.Called(bankID, sampahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KatalogSampah), args.Error(1)
}

func (m *MockTabunganRepo) ListAntrianPerSampah(bankID, sampahID string) ([]repositories.AntrianPencairanRow, error) {
	args := m.Called(bankID, sampahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.AntrianPencairanRow), args.Error(1)
}
