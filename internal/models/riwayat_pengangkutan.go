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
	RiwayatPengangkutanID uint `gorm:"column:riwayat_pengangkutan_id;primaryKey;autoIncrement"`

	PengangkutanID string `gorm:"column:pengangkutan_id;size:100"`

	StatusPengangkutan StatusPengangkutan `gorm:"column:status_pengangkutan;type:status_pengangkutan_enum"`

	ChangedAt time.Time `gorm:"column:changed_at"`

	ChangedBy string `gorm:"column:changed_by;size:100"`

	Notes string `gorm:"column:notes;size:255"`
}

func (RiwayatPengangkutan) TableName() string {
	return "riwayat_pengangkutan"
}