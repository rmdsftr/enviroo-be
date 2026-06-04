package models

import "time"

type AntarEnum string

const (
	AntarBSI AntarEnum = "bsi"
	AntarBSU AntarEnum = "bsu"
)

type DistribusiSisa struct {
	DistribusiID string `json:"distribusi_id" gorm:"column:distribusi_id;primaryKey;size:100"`

	BagiHasilID string `json:"bagi_hasil_id" gorm:"column:bagi_hasil_id;size:100"`

	TotalSisa float64 `json:"total_sisa" gorm:"column:total_sisa;type:decimal(20,4)"`

	SatuanTotal string `json:"satuan_total" gorm:"column:satuan_total;type:satuan_reward_enum"`

	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;default:current_timestamp"`

	CreatedBy string `json:"created_by" gorm:"column:created_by;size:100"`

	BagiHasil BagiHasil `json:"bagi_hasil,omitempty" gorm:"foreignKey:BagiHasilID;references:BagiHasilID"`
}

func (DistribusiSisa) TableName() string {
	return "distribusi_sisa"
}