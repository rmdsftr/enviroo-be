package models

import "time"

type SatuanRewardEnum string

const (
	SatuanRewardEnumRp SatuanRewardEnum = "Rp"
	SatuanRewardEnumPoin SatuanRewardEnum = "poin"
)

type RewardEnum string

const (
	RewardEnumUang   RewardEnum = "Uang"
	RewardEnumSembako RewardEnum = "Sembako"
)

type Reward struct {
	RewardID   int       `gorm:"column:reward_id;primaryKey;autoIncrement"`
	NamaReward RewardEnum    `gorm:"column:nama_reward;type:varchar(100);not null"`
	Satuan     string    `gorm:"column:satuan;type:varchar(5);not null"`
	Deskripsi  *string   `gorm:"column:deskripsi;type:varchar(255)"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Reward) TableName() string {
	return "reward"
}
