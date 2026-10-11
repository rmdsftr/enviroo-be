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
	RewardEnumBarang RewardEnum = "Barang"
)

type Reward struct {
	RewardID   int              `gorm:"column:reward_id;primaryKey;autoIncrement"`
	NamaReward RewardEnum       `gorm:"column:nama_reward;type:jenis_reward_enum;not null"`
	Satuan     SatuanRewardEnum `gorm:"column:satuan;type:satuan_reward_enum;not null"`
	Deskripsi  *string          `gorm:"column:deskripsi;type:varchar(255);not null"`
	CreatedAt  time.Time        `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time        `gorm:"column:updated_at;autoUpdateTime"`
}

func (Reward) TableName() string {
	return "reward"
}
