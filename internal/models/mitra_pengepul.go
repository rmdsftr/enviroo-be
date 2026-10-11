package models

import "time"

type MitraPengepul struct {
	MitraID   string    `gorm:"column:mitra_id;type:varchar(13);primaryKey"`
	NamaMitra string    `gorm:"column:nama_mitra;type:varchar(100);not null"`
	BankID    string    `gorm:"column:bank_id;type:varchar(9);not null;index"`
	IsActive  bool      `gorm:"column:is_active;type:boolean;default:true;not null"`
	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP;not null"`

	BankSampah BankSampah `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:RESTRICT"`
}

func (MitraPengepul) TableName() string {
	return "mitra_pengepul"
}
