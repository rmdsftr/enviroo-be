package mocks

import (
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"time"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockBagiHasilRepo struct {
	mock.Mock
}

func (m *MockBagiHasilRepo) GetBankByID(bankID string) (*models.BankSampah, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockBagiHasilRepo) GetPenjualanWithReward(penjualanID, bankID string) (*models.Penjualan, error) {
	args := m.Called(penjualanID, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Penjualan), args.Error(1)
}

func (m *MockBagiHasilRepo) GetDetailPenjualan(db *gorm.DB, penjualanID string) ([]models.DetailPenjualan, error) {
	args := m.Called(db, penjualanID)
	return args.Get(0).([]models.DetailPenjualan), args.Error(1)
}

func (m *MockBagiHasilRepo) GetBSUList(db *gorm.DB, parentBankID string) ([]models.BankSampah, error) {
	args := m.Called(db, parentBankID)
	return args.Get(0).([]models.BankSampah), args.Error(1)
}

func (m *MockBagiHasilRepo) GetTabunganNasabah(db *gorm.DB, sampahID string, bankIDs []string) ([]models.TabunganSampah, error) {
	args := m.Called(db, sampahID, bankIDs)
	return args.Get(0).([]models.TabunganSampah), args.Error(1)
}

func (m *MockBagiHasilRepo) GetTabunganBSU(db *gorm.DB, sampahID string, bsuIDs []string) ([]models.TabunganSampah, error) {
	args := m.Called(db, sampahID, bsuIDs)
	return args.Get(0).([]models.TabunganSampah), args.Error(1)
}

func (m *MockBagiHasilRepo) SaveTabungan(db *gorm.DB, t *models.TabunganSampah) error {
	args := m.Called(db, t)
	return args.Error(0)
}

func (m *MockBagiHasilRepo) UpsertSchemaHargaNasabah(db *gorm.DB, sampahID string, harga float64, satuan models.SatuanRewardEnum, adminID string) error {
	args := m.Called(db, sampahID, harga, satuan, adminID)
	return args.Error(0)
}

func (m *MockBagiHasilRepo) CreateBagiHasil(db *gorm.DB, bh *models.BagiHasil) error {
	args := m.Called(db, bh)
	return args.Error(0)
}

func (m *MockBagiHasilRepo) CreatePerhitunganSisa(db *gorm.DB, ps *models.PerhitunganSisa) error {
	args := m.Called(db, ps)
	return args.Error(0)
}

func (m *MockBagiHasilRepo) CreatePenerimaBagiHasil(db *gorm.DB, p *models.PenerimaBagiHasil) error {
	args := m.Called(db, p)
	return args.Error(0)
}

func (m *MockBagiHasilRepo) CreateDetailBagiHasil(db *gorm.DB, d *models.DetailBagiHasil) error {
	args := m.Called(db, d)
	return args.Error(0)
}

func (m *MockBagiHasilRepo) CreatePencairanTabungan(db *gorm.DB, p *models.PencairanTabungan) error {
	args := m.Called(db, p)
	return args.Error(0)
}

func (m *MockBagiHasilRepo) UpdatePenjualanStatus(db *gorm.DB, penjualan *models.Penjualan, status models.StatusBagiHasilEnum) error {
	args := m.Called(db, penjualan, status)
	return args.Error(0)
}

func (m *MockBagiHasilRepo) UpdateSaldoRekening(db *gorm.DB, bankID *string, nasabahID *string, rewardID int, reward models.Reward, nilai float64, entitas models.EntitasEnum, adminID string, now time.Time) error {
	args := m.Called(db, bankID, nasabahID, rewardID, reward, nilai, entitas, adminID, now)
	return args.Error(0)
}

func (m *MockBagiHasilRepo) GetNasabahWithUser(nasabahID string) (*models.Nasabah, error) {
	args := m.Called(nasabahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Nasabah), args.Error(1)
}

func (m *MockBagiHasilRepo) GetNasabahInfoBatch(nasabahIDs []string) ([]repositories.NasabahRow, error) {
	args := m.Called(nasabahIDs)
	return args.Get(0).([]repositories.NasabahRow), args.Error(1)
}

func (m *MockBagiHasilRepo) GetBagiHasilByPenjualan(penjualanID string) (*models.BagiHasil, error) {
	args := m.Called(penjualanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BagiHasil), args.Error(1)
}

func (m *MockBagiHasilRepo) GetBagiHasilWithAll(bagiHasilID string) (*models.BagiHasil, error) {
	args := m.Called(bagiHasilID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BagiHasil), args.Error(1)
}

func (m *MockBagiHasilRepo) GetPenerimaBagiHasil(bagiHasilID string) ([]models.PenerimaBagiHasil, error) {
	args := m.Called(bagiHasilID)
	return args.Get(0).([]models.PenerimaBagiHasil), args.Error(1)
}

func (m *MockBagiHasilRepo) GetPenerimaNasabahFull(bagiHasilID string) ([]models.PenerimaBagiHasil, error) {
	args := m.Called(bagiHasilID)
	return args.Get(0).([]models.PenerimaBagiHasil), args.Error(1)
}

func (m *MockBagiHasilRepo) GetPenerimaByNasabah(nasabahID, startDate, endDate string) ([]models.PenerimaBagiHasil, error) {
	args := m.Called(nasabahID, startDate, endDate)
	return args.Get(0).([]models.PenerimaBagiHasil), args.Error(1)
}

func (m *MockBagiHasilRepo) GetPenerimaBagiHasilNasabah(penerimaID string) (*models.PenerimaBagiHasil, error) {
	args := m.Called(penerimaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PenerimaBagiHasil), args.Error(1)
}

func (m *MockBagiHasilRepo) GetDetailBagiHasilItems(penerimaID string) ([]models.DetailBagiHasil, error) {
	args := m.Called(penerimaID)
	return args.Get(0).([]models.DetailBagiHasil), args.Error(1)
}

func (m *MockBagiHasilRepo) GetBagiHasilByBankWithFilter(bankID string, startTime, endTime time.Time) ([]models.BagiHasil, error) {
	args := m.Called(bankID, startTime, endTime)
	return args.Get(0).([]models.BagiHasil), args.Error(1)
}

func (m *MockBagiHasilRepo) GetNamaPetugas(adminID string) string {
	args := m.Called(adminID)
	return args.String(0)
}

func (m *MockBagiHasilRepo) FindDistribusiSisaID(bagiHasilID string) *string {
	args := m.Called(bagiHasilID)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*string)
}
