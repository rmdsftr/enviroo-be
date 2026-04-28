package models

import "time"

type HistoryPoinSembako struct {
	HistorySembakoID int    `gorm:"column:history_sembako_id;primaryKey;autoIncrement"`
	SembakoID        string `gorm:"column:sembako_id;size:100"`

	LevelUser LevelUser `gorm:"column:level_user;type:level_user_enum;not null;"`
	PoinLama  float64   `gorm:"column:poin_lama;type:decimal(20,4)"`
	PoinBaru  float64   `gorm:"column:poin_baru;type:decimal(20,4)"`

	ChangedBy string    `gorm:"column:changed_by;size:100"`
	ChangedAt time.Time `gorm:"column:changed_at;autoCreateTime"`

	// Relasi balik
	KatalogSembako KatalogSembako `gorm:"foreignKey:SembakoID;references:SembakoID"`
}

func (HistoryPoinSembako) TableName() string {
	return "history_poin_sembako"
}
