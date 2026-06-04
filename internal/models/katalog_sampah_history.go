package models

import "time"

type KatalogSampahHistory struct {
	HistorySampahID int       `gorm:"column:history_sampah_id;primaryKey;autoIncrement"`
	SchemaID        int       `gorm:"column:schema_id;size:100;not null;index"`

	HargaLama   float64   `gorm:"column:harga_lama;type:decimal(20,4)"`
	HargaBaru   float64   `gorm:"column:harga_baru;type:decimal(20,4)"`

	ChangedAt       time.Time `gorm:"column:changed_at;autoCreateTime"`
	ChangedBy       string    `gorm:"column:changed_by;size:100"`

	SchemaHargaSampah SchemaHargaSampah `gorm:"foreignKey:SchemaID;references:SchemaID;constraint:OnDelete:CASCADE"`
}

func (KatalogSampahHistory) TableName() string {
	return "katalog_sampah_history"
}