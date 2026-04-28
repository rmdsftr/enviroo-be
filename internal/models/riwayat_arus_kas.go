package models

import "time"

type RiwayatArusKas struct {
	ArusKasID      string    `gorm:"column:arus_kas_id;type:varchar(100);primaryKey"`
	KasID          string    `gorm:"column:kas_id;type:uuid;not null"`
	Jumlah         float64   `gorm:"column:jumlah;type:decimal(20,4);not null"`
	NominalSebelum float64   `gorm:"column:nominal_sebelum;type:decimal(20,4);not null"`
	NominalSesudah float64   `gorm:"column:nominal_sesudah;type:decimal(20,4);not null"`
	CreatedAt      time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP"`

	KasBank KasBank `gorm:"foreignKey:KasID;references:KasID"`
}

func (RiwayatArusKas) TableName() string {
	return "riwayat_arus_kas"
}