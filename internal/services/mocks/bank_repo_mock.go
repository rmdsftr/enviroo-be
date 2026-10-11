package mocks

import (
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockBankRepo struct {
	mock.Mock
}

func (m *MockBankRepo) FindBank(bankID string) (*models.BankSampah, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockBankRepo) ListBankAktifWithWilayah() ([]models.BankSampah, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.BankSampah), args.Error(1)
}

func (m *MockBankRepo) ListSemuaBank() ([]models.BankSampah, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.BankSampah), args.Error(1)
}

func (m *MockBankRepo) ListNasabahByBank(bankID string, limit, offset int) ([]repositories.NasabahBankRow, error) {
	args := m.Called(bankID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.NasabahBankRow), args.Error(1)
}

func (m *MockBankRepo) ListBSI(limit, offset int) ([]repositories.BSIRow, error) {
	args := m.Called(limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.BSIRow), args.Error(1)
}

func (m *MockBankRepo) ListBSU(limit, offset int) ([]repositories.BSURow, error) {
	args := m.Called(limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.BSURow), args.Error(1)
}

func (m *MockBankRepo) ListBSM(limit, offset int) ([]repositories.BSMRow, error) {
	args := m.Called(limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.BSMRow), args.Error(1)
}

func (m *MockBankRepo) ListUnitByParent(parentBankID string, limit, offset int) ([]repositories.UnitBSURow, error) {
	args := m.Called(parentBankID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.UnitBSURow), args.Error(1)
}

func (m *MockBankRepo) ListBSUByParent(parentBankID string) ([]repositories.BSUSimpleRow, error) {
	args := m.Called(parentBankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.BSUSimpleRow), args.Error(1)
}

func (m *MockBankRepo) FindRewardByNama(tx *gorm.DB, nama models.RewardEnum) (*models.Reward, error) {
	args := m.Called(tx, nama)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reward), args.Error(1)
}

func (m *MockBankRepo) FindUserTx(tx *gorm.DB, userID string) (*models.User, error) {
	args := m.Called(tx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockBankRepo) CountAdminAktifPending(tx *gorm.DB, userID string) (int64, error) {
	args := m.Called(tx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockBankRepo) CreateBank(tx *gorm.DB, b *models.BankSampah) error {
	args := m.Called(tx, b)
	return args.Error(0)
}

func (m *MockBankRepo) SaveBank(tx *gorm.DB, b *models.BankSampah) error {
	args := m.Called(tx, b)
	return args.Error(0)
}

func (m *MockBankRepo) CreateSaldoRekening(tx *gorm.DB, s *models.SaldoRekening) error {
	args := m.Called(tx, s)
	return args.Error(0)
}

func (m *MockBankRepo) CreateAdmin(tx *gorm.DB, a *models.Admin) error {
	args := m.Called(tx, a)
	return args.Error(0)
}

func (m *MockBankRepo) CreateHistoryAkunBank(tx *gorm.DB, h *models.HistoryAkunBank) error {
	args := m.Called(tx, h)
	return args.Error(0)
}

func (m *MockBankRepo) GetProfilBank(bankID string) (*repositories.BankProfilRow, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.BankProfilRow), args.Error(1)
}

func (m *MockBankRepo) GetHistoryAkunBank(bankID string) ([]repositories.HistoryBankRow, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.HistoryBankRow), args.Error(1)
}

func (m *MockBankRepo) FindBankWithRelasi(bankID string) (*models.BankSampah, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockBankRepo) ListAdminWithUserByBank(bankID string) ([]models.Admin, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Admin), args.Error(1)
}

func (m *MockBankRepo) ListAdminBank(bankID string) ([]repositories.AdminBankRow, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.AdminBankRow), args.Error(1)
}

func (m *MockBankRepo) DeleteBank(bank *models.BankSampah) error {
	args := m.Called(bank)
	return args.Error(0)
}

func (m *MockBankRepo) FindUser(userID string) (*models.User, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockBankRepo) FindAdminWithUserBank(adminID string) (*models.Admin, error) {
	args := m.Called(adminID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Admin), args.Error(1)
}

func (m *MockBankRepo) FindNamaAdmin(adminID string) string {
	args := m.Called(adminID)
	return args.String(0)
}

func (m *MockBankRepo) FindAdminByUserAndBankTx(tx *gorm.DB, userID, bankID string) (*models.Admin, error) {
	args := m.Called(tx, userID, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Admin), args.Error(1)
}

func (m *MockBankRepo) DeleteAktivasiAdminBelumDipakai(tx *gorm.DB, userID string, tujuan *models.TujuanEnum) error {
	args := m.Called(tx, userID, tujuan)
	return args.Error(0)
}

func (m *MockBankRepo) DeleteAdmin(tx *gorm.DB, adminID string) error {
	args := m.Called(tx, adminID)
	return args.Error(0)
}

func (m *MockBankRepo) UpdateStatusAdmin(tx *gorm.DB, adminID string, status models.StatusAkun) error {
	args := m.Called(tx, adminID, status)
	return args.Error(0)
}
