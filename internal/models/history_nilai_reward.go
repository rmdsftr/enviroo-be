package models

import "time"

type HistoryNilaiReward struct {
	HistoryRewardID  string    `gorm:"column:history_reward_id;type:varchar(15);primaryKey"`
	NilaiRewardID    string    `gorm:"column:nilai_reward_id;type:varchar(11);not null;index"`

	OldPersen        *float64  `gorm:"column:old_persen;type:decimal(20,4);not null"`
	NewPersen        *float64  `gorm:"column:new_persen;type:decimal(20,4);not null"`

	ChangedAt        time.Time `gorm:"column:changed_at;autoCreateTime"`
	ChangedBy        *string   `gorm:"column:changed_by;type:varchar(25);not null"`

	NilaiRewardBank NilaiRewardBank `gorm:"foreignKey:NilaiRewardID;references:NilaiRewardID;constraint:OnDelete:CASCADE"`
}

func (HistoryNilaiReward) TableName() string {
	return "history_nilai_reward"
}
