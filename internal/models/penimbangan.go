package models

import (
	"time"
)

type StatusPenimbangan string

const (
	StatusAktif          StatusPenimbangan = "aktif"
	StatusSelesai        StatusPenimbangan = "selesai"
	StatusDibatalkan     StatusPenimbangan = "dibatalkan"
	StatusTimbangPending StatusPenimbangan = "pending"
)

type Penimbangan struct {
	PenimbanganID     string            `gorm:"column:penimbangan_id;type:varchar(17);primaryKey" json:"penimbangan_id"`
	JadwalID          *string           `gorm:"column:jadwal_id;type:varchar(15);not null;uniqueIndex:idx_penimbangan_jadwal_tanggal" json:"jadwal_id"`
	BankID            *string           `gorm:"column:bank_id;type:varchar(9);not null;index" json:"bank_id"`
	StartedBy         *string           `gorm:"column:started_by;type:varchar(25)" json:"started_by"`
	StartedAt         *time.Time        `gorm:"column:started_at" json:"started_at"`
	EndedBy           *string           `gorm:"column:ended_by;type:varchar(25)" json:"ended_by"`
	EndedAt           *time.Time        `gorm:"column:ended_at" json:"ended_at"`
	StatusPenimbangan StatusPenimbangan `gorm:"column:status_penimbangan;type:status_penimbangan_enum;not null;default:'pending'" json:"status_penimbangan"`

	TanggalSesi time.Time `gorm:"column:tanggal_sesi;type:date;not null;uniqueIndex:idx_penimbangan_jadwal_tanggal" json:"tanggal_sesi"`

	AlasanPembatalan *string   `gorm:"column:alasan_pembatalan;type:varchar(255)" json:"alasan_pembatalan"`
	CreatedAt        time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP;not null"`

	Jadwal Jadwal     `gorm:"foreignKey:JadwalID;references:JadwalID;constraint:OnDelete:RESTRICT"`
	Bank   BankSampah `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:RESTRICT"`
}

func (Penimbangan) TableName() string {
	return "penimbangan"
}
