package models

import "time"

type PenerimaBagiHasil struct {
	PenerimaID string `gorm:"column:penerima_id;type:varchar(22);primaryKey" json:"penerima_id"`

	BagiHasilID *string `gorm:"column:bagi_hasil_id;type:varchar(19);not null;index;uniqueIndex:idx_penerima_bagi_hasil_nasabah_bagihasil" json:"bagi_hasil_id"`
	RewardID    *int    `gorm:"column:reward_id;not null" json:"reward_id"`
	NasabahID   *string `gorm:"column:nasabah_id;type:varchar(15);not null;uniqueIndex:idx_penerima_bagi_hasil_nasabah_bagihasil" json:"nasabah_id"`

	TotalItem int `gorm:"column:total_item;not null" json:"total_item"`

	TotalDiterima float64 `gorm:"column:total_diterima;type:decimal(20,4);not null" json:"total_diterima"`

	SatuanDiterima SatuanRewardEnum `gorm:"column:satuan_diterima;type:satuan_reward_enum;not null" json:"satuan_diterima"`

	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP;not null" json:"created_at"`
	CreatedBy *string   `gorm:"column:created_by;type:varchar(25);not null" json:"created_by"`

	BagiHasil *BagiHasil `gorm:"foreignKey:BagiHasilID;references:BagiHasilID;constraint:OnDelete:RESTRICT"`
	Reward    *Reward    `gorm:"foreignKey:RewardID;references:RewardID;constraint:OnDelete:RESTRICT"`
	Nasabah   *Nasabah   `gorm:"foreignKey:NasabahID;references:NasabahID;constraint:OnDelete:RESTRICT"`

	PencairanTabungan []PencairanTabungan `gorm:"foreignKey:PenerimaID;references:PenerimaID"`
}

func (PenerimaBagiHasil) TableName() string {
	return "penerima_bagi_hasil"
}