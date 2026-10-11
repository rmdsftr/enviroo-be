package models

import (
	"time"

	"gorm.io/datatypes"
)

type HistoryAkunBank struct {
	HistoryBankID string         `gorm:"type:varchar(14);primaryKey"`
	BankID        string         `gorm:"type:varchar(9);not null;index:idx_history_bank_id"`
	Action        string         `gorm:"type:varchar(10);not null;check:action IN ('CREATE','UPDATE','DELETE')"`
	OldValue      datatypes.JSON `gorm:"type:jsonb"`
	NewValue      datatypes.JSON `gorm:"type:jsonb"`
	Informasi     string         `gorm:"type:text;not null"`
	Keterangan    string         `gorm:"type:text;not null"`
	CreatedAt     time.Time      `gorm:"type:timestamp;not null;default:current_timestamp"`
	CreatedBy     string         `gorm:"type:varchar(25);not null"`

	Bank  *BankSampah `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:CASCADE"`
}

func (HistoryAkunBank) TableName() string {
	return "history_akun_bank"
}
