package models

import "time"

type StatusPengangkutan string

const (
	StatusRequested StatusPengangkutan = "requested"
	StatusApproved  StatusPengangkutan = "approved"
	StatusRejected  StatusPengangkutan = "rejected"
	StatusOTW       StatusPengangkutan = "otw"
	StatusCanceled  StatusPengangkutan = "canceled"
	StatusCompleted StatusPengangkutan = "completed"
	StatusArrived   StatusPengangkutan = "arrived"
)

type RiwayatPengangkutan struct {
	RiwayatPengangkutanID string `gorm:"column:riwayat_pengangkutan_id;type:varchar(19);primaryKey"`

	PengangkutanID string `gorm:"column:pengangkutan_id;size:18;not null;uniqueIndex:idx_riwayat_pengangkutan_status"`

	StatusPengangkutan StatusPengangkutan `gorm:"column:status_pengangkutan;type:status_pengangkutan_enum;not null;uniqueIndex:idx_riwayat_pengangkutan_status"`

	ChangedAt time.Time `gorm:"column:changed_at;default:CURRENT_TIMESTAMP;not null"`

	ChangedBy string `gorm:"column:changed_by;size:13;not null"`

	Notes string `gorm:"column:notes;size:255"`

	Pengangkutan PengangkutanSampah `gorm:"foreignKey:PengangkutanID;references:PengangkutanID;constraint:OnDelete:CASCADE"`
	Admin        Admin              `gorm:"foreignKey:ChangedBy;references:AdminID"`
}

func (RiwayatPengangkutan) TableName() string {
	return "riwayat_pengangkutan"
}