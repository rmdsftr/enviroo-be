package models

import "time"

type AntarEnum string

const (
	AntarBSI AntarEnum = "bsi"
	AntarBSU AntarEnum = "bsu"
)

type DistribusiSisa struct {
	DistribusiID string `json:"distribusi_id" gorm:"column:distribusi_id;primaryKey;size:19"`

	BagiHasilID string `json:"bagi_hasil_id" gorm:"column:bagi_hasil_id;size:19;not null;uniqueIndex"`

	TotalSisa float64 `json:"total_sisa" gorm:"column:total_sisa;type:decimal(20,4);not null"`

	SatuanTotal SatuanRewardEnum `json:"satuan_total" gorm:"column:satuan_total;type:satuan_reward_enum;not null"`

	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;default:current_timestamp;not null"`

	CreatedBy string `json:"created_by" gorm:"column:created_by;size:25;not null"`

	BagiHasil BagiHasil `json:"bagi_hasil,omitempty" gorm:"foreignKey:BagiHasilID;references:BagiHasilID;constraint:OnDelete:RESTRICT"`
}

func (DistribusiSisa) TableName() string {
	return "distribusi_sisa"
}