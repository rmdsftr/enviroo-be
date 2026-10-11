package mocks

import (
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockRewardRepo struct {
	mock.Mock
}

func (m *MockRewardRepo) ListRewards() ([]models.Reward, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Reward), args.Error(1)
}

func (m *MockRewardRepo) FindReward(tx *gorm.DB, rewardID int) (*models.Reward, error) {
	args := m.Called(tx, rewardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reward), args.Error(1)
}

func (m *MockRewardRepo) FindRewardByJenis(nama models.RewardEnum, satuan models.SatuanRewardEnum, exceptID int) (*models.Reward, error) {
	args := m.Called(nama, satuan, exceptID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reward), args.Error(1)
}

func (m *MockRewardRepo) CreateReward(tx *gorm.DB, reward *models.Reward) error {
	return m.Called(tx, reward).Error(0)
}

func (m *MockRewardRepo) SaveReward(tx *gorm.DB, reward *models.Reward) error {
	return m.Called(tx, reward).Error(0)
}

func (m *MockRewardRepo) DeleteReward(tx *gorm.DB, reward *models.Reward) error {
	return m.Called(tx, reward).Error(0)
}

func (m *MockRewardRepo) HitungPemakaianReward(rewardID int) ([]repositories.PemakaianReward, error) {
	args := m.Called(rewardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.PemakaianReward), args.Error(1)
}

func (m *MockRewardRepo) FindBank(bankID string) (*models.BankSampah, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BankSampah), args.Error(1)
}

func (m *MockRewardRepo) ListNilaiRewardByBank(bankID string) ([]models.NilaiRewardBank, error) {
	args := m.Called(bankID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.NilaiRewardBank), args.Error(1)
}

func (m *MockRewardRepo) ListNilaiRewardByBankReward(bankID string, rewardID int) ([]models.NilaiRewardBank, error) {
	args := m.Called(bankID, rewardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.NilaiRewardBank), args.Error(1)
}

func (m *MockRewardRepo) FindNilaiReward(tx *gorm.DB, bankID string, rewardID int, level models.LevelUser) (*models.NilaiRewardBank, error) {
	args := m.Called(tx, bankID, rewardID, level)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.NilaiRewardBank), args.Error(1)
}

func (m *MockRewardRepo) FindNilaiRewardByID(nilaiRewardID string) (*models.NilaiRewardBank, error) {
	args := m.Called(nilaiRewardID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.NilaiRewardBank), args.Error(1)
}

func (m *MockRewardRepo) CreateNilaiReward(tx *gorm.DB, row *models.NilaiRewardBank) error {
	return m.Called(tx, row).Error(0)
}

func (m *MockRewardRepo) SaveNilaiReward(tx *gorm.DB, row *models.NilaiRewardBank) error {
	return m.Called(tx, row).Error(0)
}

func (m *MockRewardRepo) NonaktifkanNilaiRewardPasangan(tx *gorm.DB, bankID string, rewardID int, adminID string) (int64, error) {
	args := m.Called(tx, bankID, rewardID, adminID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRewardRepo) ListHistoryByNilaiReward(nilaiRewardIDs []string) ([]repositories.HistoryNilaiRewardRow, error) {
	args := m.Called(nilaiRewardIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repositories.HistoryNilaiRewardRow), args.Error(1)
}

func (m *MockRewardRepo) CreateHistory(tx *gorm.DB, history *models.HistoryNilaiReward) error {
	return m.Called(tx, history).Error(0)
}

func (m *MockRewardRepo) FindAdminIDByUserBank(userID, bankID string) (string, error) {
	args := m.Called(userID, bankID)
	return args.String(0), args.Error(1)
}
