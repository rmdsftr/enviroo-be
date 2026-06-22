package mocks

import (
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockPengangkutanRepo struct {
	mock.Mock
}

func (m *MockPengangkutanRepo) GetJadwalHariIni(bsiID string, todayHari models.HariEnum, mingguKe int, todayDate string) ([]repositories.ListJadwalRow, error) {
	args := m.Called(bsiID, todayHari, mingguKe, todayDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.ListJadwalRow), args.Error(1)
}

func (m *MockPengangkutanRepo) FindBank(bankID string) (*models.BankSampah, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockPengangkutanRepo) FindBankAktif(bankID string) (*models.BankSampah, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockPengangkutanRepo) FindBSUAktifByParent(bsuID, bsiID string) (*models.BankSampah, error) {
	args := m.Called(bsuID, bsiID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockPengangkutanRepo) FindJadwalStart(bsiID, bsuID string, todayHari models.HariEnum, mingguKe int, todayDate string) (*models.Jadwal, error) {
	args := m.Called(bsiID, bsuID, todayHari, mingguKe, todayDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Jadwal), args.Error(1)
}

func (m *MockPengangkutanRepo) GetListByBSI(bsiID, startDate, endDate string) ([]repositories.PengangkutanListRow, error) {
	args := m.Called(bsiID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.PengangkutanListRow), args.Error(1)
}

func (m *MockPengangkutanRepo) GetListByBSU(bsuID, startDate, endDate string) ([]repositories.PengangkutanListRow, error) {
	args := m.Called(bsuID, startDate, endDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.PengangkutanListRow), args.Error(1)
}

func (m *MockPengangkutanRepo) FindPengangkutan(pengangkutanID string) (*models.PengangkutanSampah, error) {
	args := m.Called(pengangkutanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PengangkutanSampah), args.Error(1)
}

func (m *MockPengangkutanRepo) FindAdminByBank(adminID, bankID string) (*models.Admin, error) {
	args := m.Called(adminID, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Admin), args.Error(1)
}

func (m *MockPengangkutanRepo) FindLastRiwayat(db *gorm.DB, pengangkutanID string) (*models.RiwayatPengangkutan, error) {
	args := m.Called(db, pengangkutanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RiwayatPengangkutan), args.Error(1)
}

func (m *MockPengangkutanRepo) SetAdminBSI(db *gorm.DB, pgk *models.PengangkutanSampah, adminID string) error {
	args := m.Called(db, pgk, adminID)
	return args.Error(0)
}

func (m *MockPengangkutanRepo) CheckJadwalConflict(bsiID, bsuID string, reqHari models.HariEnum, mingguKe int, reqDateStr, jamMulai, jamSelesai string) (bool, error) {
	args := m.Called(bsiID, bsuID, reqHari, mingguKe, reqDateStr, jamMulai, jamSelesai)
	return args.Bool(0), args.Error(1)
}

func (m *MockPengangkutanRepo) FindLatestPengangkutan(bsuID, bsiID string) (*models.PengangkutanSampah, error) {
	args := m.Called(bsuID, bsiID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PengangkutanSampah), args.Error(1)
}

func (m *MockPengangkutanRepo) FindBSUList(bsiID string) ([]models.BankSampah, error) {
	args := m.Called(bsiID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.BankSampah), args.Error(1)
}

func (m *MockPengangkutanRepo) FindJadwal(jadwalID uuid.UUID) (*models.Jadwal, error) {
	args := m.Called(jadwalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Jadwal), args.Error(1)
}

func (m *MockPengangkutanRepo) GetDetailHeader(pengangkutanID string) (*repositories.PengangkutanDetailHeader, error) {
	args := m.Called(pengangkutanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.PengangkutanDetailHeader), args.Error(1)
}

func (m *MockPengangkutanRepo) GetDetailItems(pengangkutanID string) ([]repositories.PengangkutanDetailItem, error) {
	args := m.Called(pengangkutanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.PengangkutanDetailItem), args.Error(1)
}

func (m *MockPengangkutanRepo) GetSampahList(bsiID, bsuID string) ([]repositories.SampahPengangkutanRow, error) {
	args := m.Called(bsiID, bsuID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.SampahPengangkutanRow), args.Error(1)
}

func (m *MockPengangkutanRepo) GetRiwayatDetail(pengangkutanID string) ([]repositories.RiwayatDetailRow, error) {
	args := m.Called(pengangkutanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.RiwayatDetailRow), args.Error(1)
}

func (m *MockPengangkutanRepo) FindAdminsByBank(bankID string) ([]repositories.PengangkutanAdminUser, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.PengangkutanAdminUser), args.Error(1)
}

func (m *MockPengangkutanRepo) FindAdminsBankAktif(bankID string) ([]repositories.PengangkutanAdminUser, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.PengangkutanAdminUser), args.Error(1)
}

func (m *MockPengangkutanRepo) FindStok(bankID, sampahID string) (float64, error) {
	args := m.Called(bankID, sampahID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockPengangkutanRepo) FindKatalogWithAll(sampahID string) (*models.KatalogSampah, error) {
	args := m.Called(sampahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KatalogSampah), args.Error(1)
}

func (m *MockPengangkutanRepo) CreatePengangkutan(tx *gorm.DB, p *models.PengangkutanSampah) error {
	args := m.Called(tx, p)
	return args.Error(0)
}

func (m *MockPengangkutanRepo) CreateRiwayat(tx *gorm.DB, rw *models.RiwayatPengangkutan) error {
	args := m.Called(tx, rw)
	return args.Error(0)
}

func (m *MockPengangkutanRepo) CreateJadwal(tx *gorm.DB, j *models.Jadwal) error {
	args := m.Called(tx, j)
	return args.Error(0)
}

func (m *MockPengangkutanRepo) UpdatePengangkutanMeta(tx *gorm.DB, pengangkutanID string, updates map[string]interface{}) error {
	args := m.Called(tx, pengangkutanID, updates)
	return args.Error(0)
}

func (m *MockPengangkutanRepo) CreateDetailPengangkutan(tx *gorm.DB, d *models.DetailPengangkutan) error {
	args := m.Called(tx, d)
	return args.Error(0)
}

func (m *MockPengangkutanRepo) DecrStokBSU(tx *gorm.DB, bankID, sampahID string, qty float64) error {
	args := m.Called(tx, bankID, sampahID, qty)
	return args.Error(0)
}

func (m *MockPengangkutanRepo) UpsertStokBSI(tx *gorm.DB, bankID, sampahID string, qty float64) error {
	args := m.Called(tx, bankID, sampahID, qty)
	return args.Error(0)
}

func (m *MockPengangkutanRepo) FindKatalogTx(tx *gorm.DB, sampahID string) (*models.KatalogSampah, error) {
	args := m.Called(tx, sampahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.KatalogSampah), args.Error(1)
}

func (m *MockPengangkutanRepo) CreateTabunganSampah(tx *gorm.DB, t *models.TabunganSampah) error {
	args := m.Called(tx, t)
	return args.Error(0)
}
