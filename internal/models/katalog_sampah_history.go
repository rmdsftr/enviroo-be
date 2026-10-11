package models

import "time"

type KatalogSampahHistory struct {
	HistorySampahID string    `gorm:"column:history_sampah_id;type:varchar(18);primaryKey"`
	SchemaID        string    `gorm:"column:schema_id;type:varchar(14);not null;index"`

	HargaLama   float64   `gorm:"column:harga_lama;type:decimal(20,4);not null"`
	HargaBaru   float64   `gorm:"column:harga_baru;type:decimal(20,4);not null"`

	ChangedAt       time.Time `gorm:"column:changed_at;autoCreateTime;not null"`
	ChangedBy       string    `gorm:"column:changed_by;size:25;not null"`

	SchemaHargaSampah SchemaHargaSampah `gorm:"foreignKey:SchemaID;references:SchemaID;constraint:OnDelete:CASCADE"`
}

func (KatalogSampahHistory) TableName() string {
	return "katalog_sampah_history"
}