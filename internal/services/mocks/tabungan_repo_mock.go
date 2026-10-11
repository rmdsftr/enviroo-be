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
