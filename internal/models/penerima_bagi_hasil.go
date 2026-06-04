package models

import "time"

type PenerimaBagiHasil struct {
	PenerimaID string `gorm:"column:penerima_id;type:varchar(100);primaryKey" json:"penerima_id"`

	BagiHasilID *string `gorm:"column:bagi_hasil_id;type:varchar(100)" json:"bagi_hasil_id"`
	RewardID    *int    `gorm:"column:reward_id" json:"reward_id"`
	NasabahID   *string `gorm:"column:nasabah_id;type:varchar(100)" json:"nasabah_id"`

	TotalItem int `gorm:"column:total_item" json:"total_item"`

	TotalDiterima float64 `gorm:"column:total_diterima;type:decimal(20,4)" json:"total_diterima"`

	SatuanDiterima SatuanRewardEnum `gorm:"column:satuan_diterima;type:satuan_reward_enum" json:"satuan_diterima"`

	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy *string   `gorm:"column:created_by;type:varchar(100)" json:"created_by"`

	BagiHasil *BagiHasil `gorm:"foreignKey:BagiHasilID;references:BagiHasilID"`
	Reward    *Reward    `gorm:"foreignKey:RewardID;references:RewardID"`
	Nasabah   *Nasabah   `gorm:"foreignKey:NasabahID;references:NasabahID"`
}

func (PenerimaBagiHasil) TableName() string {
	return "penerima_bagi_hasil"
}