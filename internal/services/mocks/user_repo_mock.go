package mocks

import (
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockUserRepo struct {
	mock.Mock
}

func (m *MockUserRepo) GetProfilUser(userID string) (*repositories.ProfilUserRow, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.ProfilUserRow), args.Error(1)
}

func (m *MockUserRepo) GetDetailPetugas(petugasID string) (*repositories.DetailPetugasRow, error) {
	args := m.Called(petugasID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.DetailPetugasRow), args.Error(1)
}

func (m *MockUserRepo) FindUser(userID string) (*models.User, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepo) FindUserRow(userID string) (*repositories.UserRow, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.UserRow), args.Error(1)
}

func (m *MockUserRepo) FindUserRingkas(userID string) (*repositories.UserRingkasRow, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.UserRingkasRow), args.Error(1)
}

func (m *MockUserRepo) CountUserByID(userID string) (int64, error) {
	args := m.Called(userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserRepo) CountUserByEmail(email string) (int64, error) {
	args := m.Called(email)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserRepo) CountUser() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserRepo) ListUserBukanAdmin() ([]repositories.UserRow, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.UserRow), args.Error(1)
}

func (m *MockUserRepo) ListUserBukanNasabah() ([]repositories.UserRow, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.UserRow), args.Error(1)
}

func (m *MockUserRepo) ListUserWithAkun(limit, offset int) ([]models.User, error) {
	args := m.Called(limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockUserRepo) FindUserWithAkun(userID string) (*models.User, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepo) ListNasabahWithBank(userID string) ([]models.Nasabah, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Nasabah), args.Error(1)
}

func (m *MockUserRepo) ListAdminWithBank(userID string) ([]models.Admin, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Admin), args.Error(1)
}

func (m *MockUserRepo) CountNasabahByUser(userID string) (int64, error) {
	args := m.Called(userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserRepo) CountAdminByUser(userID string) (int64, error) {
	args := m.Called(userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserRepo) CreateUser(tx *gorm.DB, user *models.User) error {
	args := m.Called(tx, user)
	return args.Error(0)
}

func (m *MockUserRepo) UpdateUser(userID string, fields map[string]interface{}) error {
	args := m.Called(userID, fields)
	return args.Error(0)
}

func (m *MockUserRepo) DeleteUser(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepo) UpdateFCMToken(userID, token string) error {
	args := m.Called(userID, token)
	return args.Error(0)
}

func (m *MockUserRepo) CreateAdmin(tx *gorm.DB, admin *models.Admin) error {
	args := m.Called(tx, admin)
	return args.Error(0)
}

func (m *MockUserRepo) FindActiveAdmin(adminID string) (*repositories.ActiveAdminRow, error) {
	args := m.Called(adminID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.ActiveAdminRow), args.Error(1)
}

func (m *MockUserRepo) FindActivePetugas(adminID string) (*repositories.ActivePetugasRow, error) {
	args := m.Called(adminID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.ActivePetugasRow), args.Error(1)
}

func (m *MockUserRepo) ListSuperadmin() ([]repositories.SuperadminRow, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.SuperadminRow), args.Error(1)
}
