package models

import "time"

type HistoryNilaiReward struct {
	HistoryRewardID  int       `gorm:"column:history_reward_id;primaryKey;autoIncrement"`
	NilaiRewardID    string    `gorm:"column:nilai_reward_id;type:varchar(100);not null"`

	OldPersen        *float64  `gorm:"column:old_persen;type:decimal(20,4)"`
	NewPersen        *float64  `gorm:"column:new_persen;type:decimal(20,4)"`

	ChangedAt        time.Time `gorm:"column:changed_at;autoCreateTime"`
	ChangedBy        *string   `gorm:"column:changed_by;type:varchar(100)"`

	// Relasi
	NilaiRewardBank NilaiRewardBank `gorm:"foreignKey:NilaiRewardID;references:NilaiRewardID;constraint:OnDelete:CASCADE"`
}

func (HistoryNilaiReward) TableName() string {
	return "history_nilai_reward"
}
