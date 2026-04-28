package models

import "time"

type Penjualan struct {
	PenjualanID      string    `gorm:"column:penjualan_id;type:varchar(100);primaryKey"`
	BankID           string    `gorm:"column:bank_id;type:varchar(100);not null"`
	RewardID         int       `gorm:"column:reward_id;not null"`
	IdentitasPembeli string    `gorm:"column:identitas_pembeli;type:varchar(255)"`
	CreatedAt        time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	TotalItem        int       `gorm:"column:total_item;not null;default:0"`
	TotalPoin        float64   `gorm:"column:total_poin;type:decimal(20,4);not null;default:0"`
	SoldBy           string    `gorm:"column:sold_by;type:varchar(100)"`
	BuktiFoto        string    `gorm:"column:bukti_foto;type:varchar(255)"`

	BankSampah BankSampah `gorm:"foreignKey:BankID;references:BankID"`
	Reward     Reward     `gorm:"foreignKey:RewardID;references:RewardID"`
}

func (Penjualan) TableName() string {
	return "penjualan"
}