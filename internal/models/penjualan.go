package models

import "time"

type StatusBagiHasilEnum string

const (
	BagiHasilPending  StatusBagiHasilEnum = "pending"
	BagiHasilBerhasil StatusBagiHasilEnum = "berhasil"
)

type Penjualan struct {
	PenjualanID string `gorm:"column:penjualan_id;type:varchar(18);primaryKey"`
	BankID      string `gorm:"column:bank_id;type:varchar(9);not null;index"`
	RewardID    int    `gorm:"column:reward_id;not null"`
	MitraID     string `gorm:"column:mitra_id;type:varchar(13);not null;index"`

	TotalItem      int              `gorm:"column:total_item;not null;default:0;check:total_item >= 0"`
	TotalPenjualan float64          `gorm:"column:total_penjualan;type:decimal(20,4);not null;default:0;check:total_penjualan >= 0"`
	SatuanReward   SatuanRewardEnum `gorm:"column:satuan_reward;type:satuan_reward_enum;not null"`

	CreatedAt       time.Time           `gorm:"column:created_at;default:CURRENT_TIMESTAMP;not null"`
	SoldBy          string              `gorm:"column:sold_by;type:varchar(25);not null"`
	BuktiFoto       string              `gorm:"column:bukti_foto;type:varchar(255)"`
	StatusBagiHasil StatusBagiHasilEnum `gorm:"column:status_bagi_hasil;type:status_bagi_hasil_enum;not null;default:'pending'"`

	BankSampah    BankSampah    `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:RESTRICT"`
	Reward        Reward        `gorm:"foreignKey:RewardID;references:RewardID;constraint:OnDelete:RESTRICT"`
	MitraPengepul MitraPengepul `gorm:"foreignKey:MitraID;references:MitraID;constraint:OnDelete:RESTRICT"`
	Admin         Admin         `gorm:"foreignKey:SoldBy;references:AdminID;constraint:OnDelete:RESTRICT"`
}

func (Penjualan) TableName() string {
	return "penjualan"
}
