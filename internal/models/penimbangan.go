package models

import (
	"time"

	"github.com/google/uuid"
)

type StatusPenimbangan string

const (
	StatusAktif       StatusPenimbangan = "aktif"
	StatusSelesai     StatusPenimbangan = "selesai"
	StatusDibatalkan  StatusPenimbangan = "dibatalkan"
)

type Penimbangan struct {
	PenimbanganID     string             `gorm:"column:penimbangan_id;primaryKey" json:"penimbangan_id"`
	JadwalID          *uuid.UUID         `gorm:"column:jadwal_id;type:uuid" json:"jadwal_id"`
	BankID            *string            `gorm:"column:bank_id" json:"bank_id"`
	StartedBy         *string            `gorm:"column:started_by;index" json:"started_by"`
	StartedAt         *time.Time         `gorm:"column:started_at" json:"started_at"`
	EndedBy           *string            `gorm:"column:ended_by;index" json:"ended_by"`
	EndedAt           *time.Time         `gorm:"column:ended_at" json:"ended_at"`
	StatusPenimbangan StatusPenimbangan  `gorm:"column:status_penimbangan;type:status_penimbangan_enum;default:'aktif'" json:"status_penimbangan"`
}

// biar nama tabel fix
func (Penimbangan) TableName() string {
	return "penimbangan"
}