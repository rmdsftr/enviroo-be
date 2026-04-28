package models

import "time"

type Nasabah struct {
	NasabahID     string     `gorm:"column:nasabah_id;type:varchar(100);primaryKey"`
	BankID        string     `gorm:"column:bank_id;type:varchar(100)"`
	UserID        string     `gorm:"column:user_id;type:varchar(100)"`
	JoinedAt      time.Time  `gorm:"column:joined_at;autoCreateTime"`
	NomorRekening string     `gorm:"column:nomor_rekening;type:varchar(100)"`
	StatusNasabah StatusAkun `gorm:"column:status_nasabah;type:status_akun_enum"`

	Bank BankSampah `gorm:"foreignKey:BankID;constraint:OnDelete:CASCADE"`
	User User       `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

func (Nasabah) TableName() string {
	return "nasabah"
}
