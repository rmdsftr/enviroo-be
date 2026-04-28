package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type HistoryAkunBank struct {
	HistoryBankID uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BankID        string         `gorm:"type:varchar(100);index:idx_history_bank_id"`
	Action        string         `gorm:"type:varchar(50);check:action IN ('CREATE','UPDATE','DELETE')"`
	OldValue      datatypes.JSON `gorm:"type:jsonb"`
	NewValue      datatypes.JSON `gorm:"type:jsonb"`
	Informasi     string         `gorm:"type:text"`
	Keterangan    string         `gorm:"type:text"`
	CreatedAt     time.Time      `gorm:"type:timestamp;default:current_timestamp"`
	CreatedBy     string         `gorm:"type:varchar(100)"`

	Bank  *BankSampah `gorm:"foreignKey:BankID;references:BankID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

func (HistoryAkunBank) TableName() string {
	return "history_akun_bank"
}
