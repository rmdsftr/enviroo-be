package models

import "time"

type StatusBagiHasilEnum string

const (
	BagiHasilPending   StatusBagiHasilEnum = "pending"
	BagiHasilBerhasil  StatusBagiHasilEnum = "berhasil"
)

type Penjualan struct {
	PenjualanID      string    `gorm:"column:penjualan_id;type:varchar(100);primaryKey"`
	BankID           string    `gorm:"column:bank_id;type:varchar(100);not null"`
	RewardID         int       `gorm:"column:reward_id;not null"`
	IdentitasPembeli string    `gorm:"column:identitas_pembeli;type:varchar(255)"`

	TotalItem        int       `gorm:"column:total_item;not null;default:0"`
	TotalPenjualan   float64   `gorm:"column:total_penjualan;type:decimal(20,4);not null;default:0"`
	SatuanReward     SatuanRewardEnum `gorm:"column:satuan_reward;type:satuan_reward_enum;not null;default:'poin'"`
 
	CreatedAt        time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	SoldBy           string    `gorm:"column:sold_by;type:varchar(100)"`
	BuktiFoto        string    `gorm:"column:bukti_foto;type:varchar(255)"`
	StatusBagiHasil StatusBagiHasilEnum `gorm:"column:status_bagi_hasil;type:status_bagi_hasil_enum;not null;default:'pending'"`
  
	BankSampah BankSampah `gorm:"foreignKey:BankID;references:BankID"`
	Reward     Reward     `gorm:"foreignKey:RewardID;references:RewardID"`
}

func (Penjualan) TableName() string {
	return "penjualan"
}