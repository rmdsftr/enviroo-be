package models

import "time"

type BagiHasil struct {
	BagiHasilID string `gorm:"column:bagi_hasil_id;type:varchar(100);primaryKey" json:"bagi_hasil_id"`

	BankID      *string `gorm:"column:bank_id;type:varchar(100)" json:"bank_id"`
	PenjualanID *string `gorm:"column:penjualan_id;type:varchar(100)" json:"penjualan_id"`

	GrossBank float64 `gorm:"column:gross_bank;type:decimal(20,4)" json:"gross_bank"`
	TotalDistribusiNasabah float64 `gorm:"column:total_distribusi_nasabah;type:decimal(20,4)" json:"total_distribusi_nasabah"`
	SisaBagiHasil float64 `gorm:"column:sisa_bagi_hasil;type:decimal(20,4)" json:"sisa_bagi_hasil"`

	SatuanBagiHasil SatuanRewardEnum `gorm:"column:satuan_bagi_hasil;type:satuan_reward_enum" json:"satuan_bagi_hasil"`

	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy *string   `gorm:"column:created_by;type:varchar(100)" json:"created_by"`

	Bank      *BankSampah `gorm:"foreignKey:BankID;references:BankID"`
	Penjualan *Penjualan  `gorm:"foreignKey:PenjualanID;references:PenjualanID"`
}

func (BagiHasil) TableName() string {
	return "bagi_hasil"
}