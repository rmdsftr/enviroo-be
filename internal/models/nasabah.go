package models

import "time"

type Nasabah struct {
	NasabahID     string     `gorm:"column:nasabah_id;type:varchar(15);primaryKey"`
	BankID        string     `gorm:"column:bank_id;type:varchar(9);uniqueIndex:idx_nasabah_bank_user"`
	UserID        string     `gorm:"column:user_id;type:varchar(25);uniqueIndex:idx_nasabah_bank_user;index"`
	JoinedAt      time.Time  `gorm:"column:joined_at;autoCreateTime"`
	NomorRekening string     `gorm:"column:nomor_rekening;type:varchar(25)"`
	StatusNasabah StatusAkun `gorm:"column:status_nasabah;type:status_akun_enum;not null"`

	Bank BankSampah `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:RESTRICT"`
	User User       `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:RESTRICT"`
}

func (Nasabah) TableName() string {
	return "nasabah"
}
