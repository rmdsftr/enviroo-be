package mocks

import (
	"enviroo-be/internal/adapters/mailer"

	"github.com/stretchr/testify/mock"
)

type MockMailer struct {
	mock.Mock
}

func (m *MockMailer) SendEmail(params mailer.Params) error {
	args := m.Called(params)
	return args.Error(0)
}
