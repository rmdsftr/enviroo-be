package models

import "time"

type BagiHasil struct {
	BagiHasilID string `gorm:"column:bagi_hasil_id;type:varchar(19);primaryKey" json:"bagi_hasil_id"`

	BankID      *string `gorm:"column:bank_id;type:varchar(9);not null;index" json:"bank_id"`
	PenjualanID *string `gorm:"column:penjualan_id;type:varchar(18);not null;uniqueIndex" json:"penjualan_id"`

	GrossBank float64 `gorm:"column:gross_bank;type:decimal(20,4);not null" json:"gross_bank"`
	TotalDistribusiNasabah float64 `gorm:"column:total_distribusi_nasabah;type:decimal(20,4);not null" json:"total_distribusi_nasabah"`
	SisaBagiHasil float64 `gorm:"column:sisa_bagi_hasil;type:decimal(20,4);not null" json:"sisa_bagi_hasil"`

	SatuanBagiHasil SatuanRewardEnum `gorm:"column:satuan_bagi_hasil;type:satuan_reward_enum;not null" json:"satuan_bagi_hasil"`

	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP;not null" json:"created_at"`
	CreatedBy *string   `gorm:"column:created_by;type:varchar(25);not null" json:"created_by"`

	Bank      *BankSampah `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:RESTRICT"`
	Penjualan *Penjualan  `gorm:"foreignKey:PenjualanID;references:PenjualanID;constraint:OnDelete:RESTRICT"`
}

func (BagiHasil) TableName() string {
	return "bagi_hasil"
}