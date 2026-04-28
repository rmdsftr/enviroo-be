package models

import "time"

type KatalogSampahHistory struct {
	HistorySampahID int       `gorm:"column:history_sampah_id;primaryKey;autoIncrement"`
	SampahID        string    `gorm:"column:sampah_id;size:100;not null;index"`

	LevelUser LevelUser `gorm:"column:level_user;type:level_user_enum;not null;"`

	OldPoin   float64   `gorm:"column:old_poin;type:decimal(20,4)"`
	NewPoin   float64   `gorm:"column:new_poin;type:decimal(20,4)"`

	ChangedAt       time.Time `gorm:"column:changed_at;autoCreateTime"`
	ChangedBy       string    `gorm:"column:changed_by;size:100"`

	// Relasi ke katalog_sampah
	KatalogSampah   KatalogSampah `gorm:"foreignKey:SampahID;references:SampahID;constraint:OnDelete:CASCADE"`
}

func (KatalogSampahHistory) TableName() string {
	return "katalog_sampah_history"
}