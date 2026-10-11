package mocks

import (
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockBarangRepo struct {
	mock.Mock
}

func (m *MockBarangRepo) FindBank(bankID string) (*models.BankSampah, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockBarangRepo) ListMasterBarang(q string) ([]models.Barang, error) {
	args := m.Called(q)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Barang), args.Error(1)
}

func (m *MockBarangRepo) FindMasterBarang(barangID int) (*models.Barang, error) {
	args := m.Called(barangID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Barang), args.Error(1)
}

func (m *MockBarangRepo) FindOrCreateMasterBarang(tx *gorm.DB, namaBarang string) (*models.Barang, error) {
	args := m.Called(tx, namaBarang)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Barang), args.Error(1)
}

func (m *MockBarangRepo) AdaRewardBarangAktif(bankID string) (bool, error) {
	args := m.Called(bankID)
	return args.Bool(0), args.Error(1)
}

func (m *MockBarangRepo) FindKatalogBarang(produkID string) (*models.KatalogBarang, error) {
	args := m.Called(produkID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KatalogBarang), args.Error(1)
}

func (m *MockBarangRepo) FindKatalogBarangByBank(tx *gorm.DB, produkID, bankID string) (*models.KatalogBarang, error) {
	args := m.Called(tx, produkID, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KatalogBarang), args.Error(1)
}

func (m *MockBarangRepo) FindKatalogBarangWithMaster(produkID, bankID string) (*models.KatalogBarang, error) {
	args := m.Called(produkID, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KatalogBarang), args.Error(1)
}

func (m *MockBarangRepo) CreateKatalogBarang(tx *gorm.DB, katalog *models.KatalogBarang) error {
	args := m.Called(tx, katalog)
	return args.Error(0)
}

func (m *MockBarangRepo) UpdateKatalogBarang(tx *gorm.DB, produkID string, fields map[string]any) error {
	args := m.Called(tx, produkID, fields)
	return args.Error(0)
}

func (m *MockBarangRepo) DeleteKatalogBarang(tx *gorm.DB, produkID string) error {
	args := m.Called(tx, produkID)
	return args.Error(0)
}

func (m *MockBarangRepo) ListBarangBank(bankID, katalogBankID string, limit, offset int) ([]repositories.BarangBankRow, error) {
	args := m.Called(bankID, katalogBankID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.BarangBankRow), args.Error(1)
}

func (m *MockBarangRepo) CountPemakaianBarang(produkID string) (int64, error) {
	args := m.Called(produkID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockBarangRepo) FindStok(tx *gorm.DB, produkID, bankID string) (*models.StokBarang, error) {
	args := m.Called(tx, produkID, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StokBarang), args.Error(1)
}

func (m *MockBarangRepo) LockStok(tx *gorm.DB, produkID, bankID string) (*models.StokBarang, error) {
	args := m.Called(tx, produkID, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StokBarang), args.Error(1)
}

func (m *MockBarangRepo) CreateStok(tx *gorm.DB, stok *models.StokBarang) error {
	args := m.Called(tx, stok)
	return args.Error(0)
}

func (m *MockBarangRepo) SetStok(tx *gorm.DB, produkID, bankID string, nilai float64) error {
	args := m.Called(tx, produkID, bankID, nilai)
	return args.Error(0)
}

func (m *MockBarangRepo) TambahStok(tx *gorm.DB, produkID, bankID string, delta float64) error {
	args := m.Called(tx, produkID, bankID, delta)
	return args.Error(0)
}

func (m *MockBarangRepo) DeleteStokByProduk(tx *gorm.DB, produkID string) error {
	args := m.Called(tx, produkID)
	return args.Error(0)
}

func (m *MockBarangRepo) FindDistribusi(disbaID string) (*models.DistribusiBarangBsu, error) {
	args := m.Called(disbaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DistribusiBarangBsu), args.Error(1)
}

func (m *MockBarangRepo) CreateDistribusi(tx *gorm.DB, disba *models.DistribusiBarangBsu) error {
	args := m.Called(tx, disba)
	return args.Error(0)
}

func (m *MockBarangRepo) UpdateDistribusi(tx *gorm.DB, disbaID string, fields map[string]any) error {
	args := m.Called(tx, disbaID, fields)
	return args.Error(0)
}

func (m *MockBarangRepo) CreateDetailDistribusi(tx *gorm.DB, detail *models.DetailDistribusiBarang) error {
	args := m.Called(tx, detail)
	return args.Error(0)
}

func (m *MockBarangRepo) ListDetailDistribusi(disbaID string) ([]models.DetailDistribusiBarang, error) {
	args := m.Called(disbaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.DetailDistribusiBarang), args.Error(1)
}

func (m *MockBarangRepo) UpdateStokDetail(tx *gorm.DB, disbaID, produkID string, fields map[string]any) error {
	args := m.Called(tx, disbaID, produkID, fields)
	return args.Error(0)
}

func (m *MockBarangRepo) ListHeaderDistribusi(filterCol, bankID string, start, end *time.Time) ([]repositories.DistribusiHeaderRow, error) {
	args := m.Called(filterCol, bankID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.DistribusiHeaderRow), args.Error(1)
}

func (m *MockBarangRepo) FindHeaderDistribusi(disbaID string) (*repositories.DistribusiHeaderRow, error) {
	args := m.Called(disbaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.DistribusiHeaderRow), args.Error(1)
}

func (m *MockBarangRepo) ListItemDistribusi(disbaID string) ([]repositories.DistribusiItemRow, error) {
	args := m.Called(disbaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.DistribusiItemRow), args.Error(1)
}

func (m *MockBarangRepo) ListRiwayatDistribusiProduk(produkID, bankID string, sebagaiBSU bool) ([]repositories.RiwayatDistribusiRow, error) {
	args := m.Called(produkID, bankID, sebagaiBSU)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.RiwayatDistribusiRow), args.Error(1)
}

func (m *MockBarangRepo) FindAdminTargets(bankID string) ([]repositories.NotifTarget, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.NotifTarget), args.Error(1)
}
