package mocks

import (
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockNasabahRepo struct {
	mock.Mock
}

func (m *MockNasabahRepo) CountUser(userID string) (int64, error) {
	args := m.Called(userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNasabahRepo) CountUserByEmail(email string) (int64, error) {
	args := m.Called(email)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNasabahRepo) FindUser(userID string) (*models.User, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockNasabahRepo) FindBank(bankID string) (*models.BankSampah, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockNasabahRepo) CountNasabahAktif(userID string) (int64, error) {
	args := m.Called(userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNasabahRepo) CountNasabahPendingDiBankLain(userID, bankID string) (int64, error) {
	args := m.Called(userID, bankID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNasabahRepo) FindNasabahByUserAndBank(userID, bankID string) (*models.Nasabah, error) {
	args := m.Called(userID, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Nasabah), args.Error(1)
}

func (m *MockNasabahRepo) CreateUser(tx *gorm.DB, user *models.User) error {
	args := m.Called(tx, user)
	return args.Error(0)
}

func (m *MockNasabahRepo) CreateNasabah(tx *gorm.DB, nasabah *models.Nasabah) error {
	args := m.Called(tx, nasabah)
	return args.Error(0)
}

func (m *MockNasabahRepo) FindRewardByNama(tx *gorm.DB, nama models.RewardEnum) (*models.Reward, error) {
	args := m.Called(tx, nama)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reward), args.Error(1)
}

func (m *MockNasabahRepo) CreateSaldoRekening(tx *gorm.DB, saldo *models.SaldoRekening) error {
	args := m.Called(tx, saldo)
	return args.Error(0)
}

func (m *MockNasabahRepo) CreateRiwayatSaldo(tx *gorm.DB, riwayat *models.RiwayatArusSaldo) error {
	args := m.Called(tx, riwayat)
	return args.Error(0)
}

func (m *MockNasabahRepo) DeleteAktivasiNasabahBelumDipakai(tx *gorm.DB, userID string) error {
	args := m.Called(tx, userID)
	return args.Error(0)
}

func (m *MockNasabahRepo) GetProfilNasabah(nasabahID string) (*repositories.ProfilNasabahRow, error) {
	args := m.Called(nasabahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.ProfilNasabahRow), args.Error(1)
}

func (m *MockNasabahRepo) GetDetailNasabah(nasabahID string) (*repositories.DetailNasabahRow, error) {
	args := m.Called(nasabahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.DetailNasabahRow), args.Error(1)
}

func (m *MockNasabahRepo) FindNasabah(nasabahID string) (*models.Nasabah, error) {
	args := m.Called(nasabahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Nasabah), args.Error(1)
}

func (m *MockNasabahRepo) CountTransaksiNasabah(nasabahID string) (int64, error) {
	args := m.Called(nasabahID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNasabahRepo) PluckRekeningIDs(tx *gorm.DB, nasabahID string) ([]string, error) {
	args := m.Called(tx, nasabahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockNasabahRepo) DeleteRiwayatByRekening(tx *gorm.DB, rekeningIDs []string) error {
	args := m.Called(tx, rekeningIDs)
	return args.Error(0)
}

func (m *MockNasabahRepo) DeleteSaldoRekening(tx *gorm.DB, nasabahID string) error {
	args := m.Called(tx, nasabahID)
	return args.Error(0)
}

func (m *MockNasabahRepo) DeleteNasabah(tx *gorm.DB, nasabah *models.Nasabah) error {
	args := m.Called(tx, nasabah)
	return args.Error(0)
}
