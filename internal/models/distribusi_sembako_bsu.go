package models

import "time"

type StatusDisbakoEnum string

const (
	DisbakoPending StatusDisbakoEnum = "pending"
	DisbakoSelesai StatusDisbakoEnum = "selesai"
	DisbakoGagal   StatusDisbakoEnum = "gagal"
)

type DistribusiSembakoBsu struct {
	DisbakoID  string    `gorm:"column:disbako_id;primaryKey" json:"disbako_id"`
	BsuID      string    `gorm:"column:bsu_id" json:"bsu_id"`
	BsiID      string    `gorm:"column:bsi_id" json:"bsi_id"`
	AdminBsiID string    `gorm:"column:admin_bsi_id" json:"admin_bsi_id"`
	AdminBsuID string    `gorm:"column:admin_bsu_id" json:"admin_bsu_id"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	TotalItem float64 `gorm:"column:total_item;type:decimal(20,4)" json:"total_item"`
	TotalPoin float64 `gorm:"column:total_poin;type:decimal(20,4)" json:"total_poin"`
	StatusDistribusi StatusDisbakoEnum `gorm:"column:status_distribusi;type:status_disbako_enum;" json:"status_distribusi"`
}

func (DistribusiSembakoBsu) TableName() string {
	return "distribusi_sembako_bsu"
}
