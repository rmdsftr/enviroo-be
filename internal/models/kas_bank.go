package models

import (
	"time"

	"github.com/google/uuid"
)

type KasBank struct {
	KasID         uuid.UUID `gorm:"column:kas_id;type:uuid;primaryKey"`
	BankID        string    `gorm:"column:bank_id;type:varchar(100);not null;uniqueIndex:idx_bank_reward"`
	RewardID      int       `gorm:"column:reward_id;not null;uniqueIndex:idx_bank_reward"`
	Nominal       float64   `gorm:"column:nominal;type:decimal(20,4);not null;default:0"`
	LastUpdatedAt time.Time `gorm:"column:last_updated_at;default:CURRENT_TIMESTAMP"`
	LastUpdatedBy string    `gorm:"column:last_updated_by;type:varchar(100)"`

	BankSampah BankSampah `gorm:"foreignKey:BankID;references:BankID"`
	Reward     Reward     `gorm:"foreignKey:RewardID;references:RewardID"`
}

func (KasBank) TableName() string {
	return "kas_bank"
}