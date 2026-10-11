package mocks

import (
	"enviroo-be/internal/models"

	"github.com/stretchr/testify/mock"
)

type MockNotifikasiRepo struct {
	mock.Mock
}

func (m *MockNotifikasiRepo) Create(notif *models.Notifikasi) error {
	args := m.Called(notif)
	return args.Error(0)
}

func (m *MockNotifikasiRepo) FindByID(notifikasiID string) (*models.Notifikasi, error) {
	args := m.Called(notifikasiID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Notifikasi), args.Error(1)
}

func (m *MockNotifikasiRepo) FindByUserID(userID, roleTarget string, limit, offset int) ([]models.Notifikasi, int64, error) {
	args := m.Called(userID, roleTarget, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.Notifikasi), args.Get(1).(int64), args.Error(2)
}

func (m *MockNotifikasiRepo) MarkAsRead(notifikasiID, userID string) error {
	args := m.Called(notifikasiID, userID)
	return args.Error(0)
}

func (m *MockNotifikasiRepo) MarkAllAsRead(userID, roleTarget string) error {
	args := m.Called(userID, roleTarget)
	return args.Error(0)
}

func (m *MockNotifikasiRepo) CountUnread(userID, roleTarget string) (int64, error) {
	args := m.Called(userID, roleTarget)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNotifikasiRepo) Delete(notifikasiID, userID string) error {
	args := m.Called(notifikasiID, userID)
	return args.Error(0)
}
