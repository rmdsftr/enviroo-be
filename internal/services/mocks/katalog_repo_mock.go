package mocks

import (
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockKatalogRepo struct {
	mock.Mock
}

func (m *MockKatalogRepo) FindBank(bankID string) (*models.BankSampah, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockKatalogRepo) ListMasterSampah(q string) ([]models.Sampah, error) {
	args := m.Called(q)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Sampah), args.Error(1)
}

func (m *MockKatalogRepo) FindMasterSampah(sarokID int) (*models.Sampah, error) {
	args := m.Called(sarokID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Sampah), args.Error(1)
}

func (m *MockKatalogRepo) FindOrCreateMasterSampah(tx *gorm.DB, namaSampah string, satuanID int) (*models.Sampah, error) {
	args := m.Called(tx, namaSampah, satuanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Sampah), args.Error(1)
}

func (m *MockKatalogRepo) AdaNilaiRewardAktif(bankID string, rewardID int) (bool, error) {
	args := m.Called(bankID, rewardID)
	return args.Bool(0), args.Error(1)
}

func (m *MockKatalogRepo) FindKatalog(sampahID string) (*models.KatalogSampah, error) {
	args := m.Called(sampahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KatalogSampah), args.Error(1)
}

func (m *MockKatalogRepo) FindKatalogWithRelasi(sampahID string) (*models.KatalogSampah, error) {
	args := m.Called(sampahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KatalogSampah), args.Error(1)
}

func (m *MockKatalogRepo) FindKatalogSejenis(sarokID int, bankID string) (*models.KatalogSampah, error) {
	args := m.Called(sarokID, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KatalogSampah), args.Error(1)
}

func (m *MockKatalogRepo) ListKatalogByBank(bankID string, limit, offset int) ([]models.KatalogSampah, error) {
	args := m.Called(bankID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.KatalogSampah), args.Error(1)
}

func (m *MockKatalogRepo) CountKatalogByBank(bankID string) (int64, error) {
	args := m.Called(bankID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockKatalogRepo) CreateKatalog(tx *gorm.DB, katalog *models.KatalogSampah) error {
	args := m.Called(tx, katalog)
	return args.Error(0)
}

func (m *MockKatalogRepo) SaveKatalog(tx *gorm.DB, katalog *models.KatalogSampah) error {
	args := m.Called(tx, katalog)
	return args.Error(0)
}

func (m *MockKatalogRepo) UpdateKategoriSejenis(tx *gorm.DB, sarokID int, bankID string, kategoriID int) error {
	args := m.Called(tx, sarokID, bankID, kategoriID)
	return args.Error(0)
}

func (m *MockKatalogRepo) DeleteKatalog(katalog *models.KatalogSampah) error {
	args := m.Called(katalog)
	return args.Error(0)
}

func (m *MockKatalogRepo) CountPemakaianKatalog(sampahID string) (string, error) {
	args := m.Called(sampahID)
	return args.String(0), args.Error(1)
}

func (m *MockKatalogRepo) ListStokByBank(bankID string, sampahIDs []string) (map[string]float64, error) {
	args := m.Called(bankID, sampahIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]float64), args.Error(1)
}

func (m *MockKatalogRepo) FindStok(bankID, sampahID string) (float64, error) {
	args := m.Called(bankID, sampahID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockKatalogRepo) CreateStok(tx *gorm.DB, stok *models.StokSampah) error {
	args := m.Called(tx, stok)
	return args.Error(0)
}

func (m *MockKatalogRepo) ListSchemaHarga(sampahID string) ([]models.SchemaHargaSampah, error) {
	args := m.Called(sampahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.SchemaHargaSampah), args.Error(1)
}

func (m *MockKatalogRepo) ListHistoryHarga(sampahID string, start, end *time.Time) ([]repositories.HistoryHargaRow, error) {
	args := m.Called(sampahID, start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.HistoryHargaRow), args.Error(1)
}

func (m *MockKatalogRepo) ListKategori() ([]models.KategoriSampah, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.KategoriSampah), args.Error(1)
}

func (m *MockKatalogRepo) FindKategori(kategoriID string) (*models.KategoriSampah, error) {
	args := m.Called(kategoriID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KategoriSampah), args.Error(1)
}

func (m *MockKatalogRepo) CountKategoriDenganNama(namaNormal, exceptID string) (int64, error) {
	args := m.Called(namaNormal, exceptID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockKatalogRepo) CreateKategori(kategori *models.KategoriSampah) error {
	args := m.Called(kategori)
	return args.Error(0)
}

func (m *MockKatalogRepo) SaveKategori(kategori *models.KategoriSampah) error {
	args := m.Called(kategori)
	return args.Error(0)
}

func (m *MockKatalogRepo) DeleteKategori(kategori *models.KategoriSampah) error {
	args := m.Called(kategori)
	return args.Error(0)
}

func (m *MockKatalogRepo) CountKatalogByKategori(kategoriID string) (int64, error) {
	args := m.Called(kategoriID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockKatalogRepo) ListSatuan() ([]models.Satuan, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Satuan), args.Error(1)
}

func (m *MockKatalogRepo) FindSatuan(satuanID string) (*models.Satuan, error) {
	args := m.Called(satuanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Satuan), args.Error(1)
}

func (m *MockKatalogRepo) CountSatuanDenganNama(namaNormal, exceptID string) (int64, error) {
	args := m.Called(namaNormal, exceptID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockKatalogRepo) CreateSatuan(satuan *models.Satuan) error {
	args := m.Called(satuan)
	return args.Error(0)
}

func (m *MockKatalogRepo) SaveSatuan(satuan *models.Satuan) error {
	args := m.Called(satuan)
	return args.Error(0)
}

func (m *MockKatalogRepo) DeleteSatuan(satuan *models.Satuan) error {
	args := m.Called(satuan)
	return args.Error(0)
}

func (m *MockKatalogRepo) CountSampahBySatuan(satuanID string) (int64, error) {
	args := m.Called(satuanID)
	return args.Get(0).(int64), args.Error(1)
}
