package models

import "time"

type StatusDisbaEnum string

const (
	DisbaPending StatusDisbaEnum = "pending"
	DisbaSelesai StatusDisbaEnum = "selesai"
	DisbaGagal   StatusDisbaEnum = "gagal"
)

type DistribusiBarangBsu struct {
	DisbaID    string    `gorm:"column:disba_id;type:varchar(18);primaryKey" json:"disba_id"`
	BsiID      string    `gorm:"column:bsi_id;type:varchar(9);not null;index" json:"bsi_id"`
	BsuID      string    `gorm:"column:bsu_id;type:varchar(9);not null;index" json:"bsu_id"`
	AdminBsiID string    `gorm:"column:admin_bsi_id;type:varchar(13);not null" json:"admin_bsi_id"`
	AdminBsuID *string   `gorm:"column:admin_bsu_id;type:varchar(13)" json:"admin_bsu_id"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime;not null" json:"created_at"`

	TotalItem float64 `gorm:"column:total_item;type:decimal(20,4);not null" json:"total_item"`
	TotalPoin float64 `gorm:"column:total_poin;type:decimal(20,4);not null" json:"total_poin"`
	StatusDistribusi StatusDisbaEnum `gorm:"column:status_distribusi;type:status_disba_enum;not null" json:"status_distribusi"`

	BSI      BankSampah `gorm:"foreignKey:BsiID;references:BankID;constraint:OnDelete:RESTRICT"`
	BSU      BankSampah `gorm:"foreignKey:BsuID;references:BankID;constraint:OnDelete:RESTRICT"`
	AdminBSI Admin      `gorm:"foreignKey:AdminBsiID;references:AdminID;constraint:OnDelete:RESTRICT"`
	AdminBSU *Admin     `gorm:"foreignKey:AdminBsuID;references:AdminID;constraint:OnDelete:RESTRICT"`
}

func (DistribusiBarangBsu) TableName() string {
	return "distribusi_barang_bsu"
}
