package models

import "time"

type RiwayatPenarikan struct {
	RiwayatPenarikanID string `gorm:"column:riwayat_penarikan_id;type:varchar(25);primaryKey" json:"riwayat_penarikan_id"`

	PenarikanID string `gorm:"column:penarikan_id;type:varchar(24);not null;uniqueIndex:idx_riwayat_penarikan_status" json:"penarikan_id"`

	StatusPenarikan StatusPenarikanEnum `gorm:"column:status_penarikan;type:status_penarikan_enum;not null;uniqueIndex:idx_riwayat_penarikan_status" json:"status_penarikan"`

	Catatan *string `gorm:"column:catatan;type:varchar(255)" json:"catatan"`

	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP;not null" json:"created_at"`
	CreatedBy *string   `gorm:"column:created_by;type:varchar(25);not null" json:"created_by"`

	Penarikan *Penarikan `gorm:"foreignKey:PenarikanID;references:PenarikanID;constraint:OnDelete:CASCADE"`
}

func (RiwayatPenarikan) TableName() string {
	return "riwayat_penarikan"
}
