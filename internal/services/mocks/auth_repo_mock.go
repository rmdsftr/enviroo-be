package mocks

import (
	"enviroo-be/internal/models"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockAuthRepo struct {
	mock.Mock
}

func (m *MockAuthRepo) FindUserByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthRepo) FindUserByID(tx *gorm.DB, userID string) (*models.User, error) {
	args := m.Called(tx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthRepo) FindSessionUser(userID string) (*models.User, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthRepo) UpdateFCMToken(userID, token string) error {
	args := m.Called(userID, token)
	return args.Error(0)
}

func (m *MockAuthRepo) UpdatePassword(tx *gorm.DB, userID, hashedPassword string) error {
	args := m.Called(tx, userID, hashedPassword)
	return args.Error(0)
}

func (m *MockAuthRepo) SetSession(userID, platform, sessionID string) error {
	args := m.Called(userID, platform, sessionID)
	return args.Error(0)
}

func (m *MockAuthRepo) ClearSessionMatching(userID, platform, sessionID string) error {
	args := m.Called(userID, platform, sessionID)
	return args.Error(0)
}

func (m *MockAuthRepo) ClearAllSessions(tx *gorm.DB, userID string) error {
	args := m.Called(tx, userID)
	return args.Error(0)
}

func (m *MockAuthRepo) FindNasabahAktif(tx *gorm.DB, userID string) (*models.Nasabah, error) {
	args := m.Called(tx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Nasabah), args.Error(1)
}

func (m *MockAuthRepo) FindAdminAktif(tx *gorm.DB, userID string) (*models.Admin, error) {
	args := m.Called(tx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Admin), args.Error(1)
}

func (m *MockAuthRepo) FindNasabahPending(tx *gorm.DB, userID, nasabahID string) (*models.Nasabah, error) {
	args := m.Called(tx, userID, nasabahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Nasabah), args.Error(1)
}

func (m *MockAuthRepo) FindAdminPending(tx *gorm.DB, userID, adminID string) (*models.Admin, error) {
	args := m.Called(tx, userID, adminID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Admin), args.Error(1)
}

func (m *MockAuthRepo) FindNasabahByID(tx *gorm.DB, nasabahID string) (*models.Nasabah, error) {
	args := m.Called(tx, nasabahID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Nasabah), args.Error(1)
}

func (m *MockAuthRepo) FindNasabahByStatus(tx *gorm.DB, userID, nasabahID string, statuses []models.StatusAkun) (*models.Nasabah, error) {
	args := m.Called(tx, userID, nasabahID, statuses)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Nasabah), args.Error(1)
}

func (m *MockAuthRepo) FindAdminByStatus(tx *gorm.DB, userID, adminID string, statuses []models.StatusAkun) (*models.Admin, error) {
	args := m.Called(tx, userID, adminID, statuses)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Admin), args.Error(1)
}

func (m *MockAuthRepo) FindAdminByIDRole(tx *gorm.DB, adminID string, role models.RoleAdmin) (*models.Admin, error) {
	args := m.Called(tx, adminID, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Admin), args.Error(1)
}

func (m *MockAuthRepo) HasNasabahPending(userID string) bool {
	args := m.Called(userID)
	return args.Bool(0)
}

func (m *MockAuthRepo) HasAdminPending(userID string) bool {
	args := m.Called(userID)
	return args.Bool(0)
}

func (m *MockAuthRepo) CountNasabahAktif(tx *gorm.DB, userID string) (int64, error) {
	args := m.Called(tx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAuthRepo) CountAdminAktif(tx *gorm.DB, userID string) (int64, error) {
	args := m.Called(tx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAuthRepo) CountNasabahAktifSelain(tx *gorm.DB, userID, exceptNasabahID string) (int64, error) {
	args := m.Called(tx, userID, exceptNasabahID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAuthRepo) CountNasabahAktifAtauPendingSelain(tx *gorm.DB, userID, exceptNasabahID string) (int64, error) {
	args := m.Called(tx, userID, exceptNasabahID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAuthRepo) CountAdminAktifSelain(tx *gorm.DB, userID, exceptAdminID string) (int64, error) {
	args := m.Called(tx, userID, exceptAdminID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAuthRepo) UpdateStatusNasabah(tx *gorm.DB, nasabahID string, status models.StatusAkun) (int64, error) {
	args := m.Called(tx, nasabahID, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAuthRepo) UpdateStatusAdmin(tx *gorm.DB, adminID string, status models.StatusAkun, refreshJoinedAt bool) (int64, error) {
	args := m.Called(tx, adminID, status, refreshJoinedAt)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAuthRepo) IsBankAktif(bankID string) (bool, error) {
	args := m.Called(bankID)
	return args.Bool(0), args.Error(1)
}

func (m *MockAuthRepo) FindBank(tx *gorm.DB, bankID string) (*models.BankSampah, error) {
	args := m.Called(tx, bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockAuthRepo) FindAktivasiValidTerbaru(tx *gorm.DB, userID string, tujuan models.TujuanEnum) (*models.AktivasiAkun, error) {
	args := m.Called(tx, userID, tujuan)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AktivasiAkun), args.Error(1)
}

func (m *MockAuthRepo) ListAktivasiValid(tx *gorm.DB, userID string, tujuan models.TujuanEnum) ([]models.AktivasiAkun, error) {
	args := m.Called(tx, userID, tujuan)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.AktivasiAkun), args.Error(1)
}

func (m *MockAuthRepo) FindAktivasiValidByTarget(tx *gorm.DB, userID string, tujuan models.TujuanEnum, nasabahID, adminID *string) (*models.AktivasiAkun, error) {
	args := m.Called(tx, userID, tujuan, nasabahID, adminID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AktivasiAkun), args.Error(1)
}

func (m *MockAuthRepo) CreateAktivasi(tx *gorm.DB, aktivasi *models.AktivasiAkun) error {
	args := m.Called(tx, aktivasi)
	return args.Error(0)
}

func (m *MockAuthRepo) UpdateAktivasi(tx *gorm.DB, aktivasiID string, fields map[string]interface{}) error {
	args := m.Called(tx, aktivasiID, fields)
	return args.Error(0)
}

func (m *MockAuthRepo) MarkAktivasiUsed(tx *gorm.DB, aktivasiID string) error {
	args := m.Called(tx, aktivasiID)
	return args.Error(0)
}
