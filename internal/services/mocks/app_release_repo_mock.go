package mocks

import (
	"io"

	"enviroo-be/internal/adapters/mailer"
	"enviroo-be/internal/models"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockAppReleaseRepo struct {
	mock.Mock
}

func (m *MockAppReleaseRepo) FindLatest() (*models.AppRelease, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AppRelease), args.Error(1)
}

func (m *MockAppReleaseRepo) FindByID(releaseID int) (*models.AppRelease, error) {
	args := m.Called(releaseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AppRelease), args.Error(1)
}

func (m *MockAppReleaseRepo) MaxVersionCode() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

func (m *MockAppReleaseRepo) AdaWajibDiAntara(current, latest int) (bool, error) {
	args := m.Called(current, latest)
	return args.Bool(0), args.Error(1)
}

func (m *MockAppReleaseRepo) List() ([]models.AppRelease, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.AppRelease), args.Error(1)
}

func (m *MockAppReleaseRepo) Create(tx *gorm.DB, rilis *models.AppRelease) error {
	return m.Called(tx, rilis).Error(0)
}

func (m *MockAppReleaseRepo) Save(rilis *models.AppRelease) error {
	return m.Called(rilis).Error(0)
}

func (m *MockAppReleaseRepo) Delete(rilis *models.AppRelease) error {
	return m.Called(rilis).Error(0)
}

type MockAppReleaseStorage struct {
	mock.Mock
}

func (m *MockAppReleaseStorage) UploadObject(body io.ReadSeeker, size int64, key, contentType string) error {
	return m.Called(body, size, key, contentType).Error(0)
}

func (m *MockAppReleaseStorage) DeleteObject(key string) error {
	return m.Called(key).Error(0)
}

func (m *MockAppReleaseStorage) PublicURL(key string) string {
	return "https://media.enviroo.tech/" + key
}

type MockLinkInstallProvider struct {
	mock.Mock
}

func (m *MockLinkInstallProvider) InfoInstallTerbaru() *mailer.InstallInfo {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*mailer.InstallInfo)
}
