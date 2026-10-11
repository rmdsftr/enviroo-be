package models

import "time"

type NilaiRewardBank struct {
	NilaiRewardID string    `gorm:"column:nilai_reward_id;type:varchar(11);primaryKey"`
	BankID        string    `gorm:"column:bank_id;type:varchar(9);not null;index"`
	RewardID      int       `gorm:"column:reward_id;not null"`
	LevelUser     LevelUser `gorm:"column:level_user;type:level_user_enum;not null"`

	PersenBagiHasil float64   `gorm:"column:persen_bagi_hasil;type:decimal(20,4);not null"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`
	CreatedBy     *string   `gorm:"column:created_by;type:varchar(25);not null"`
	UpdatedBy     *string   `gorm:"column:updated_by;type:varchar(25);not null"`
	IsActive      bool      `gorm:"column:is_active;type:boolean;not null;default:true"`

	Bank   BankSampah `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:CASCADE"`
	Reward Reward     `gorm:"foreignKey:RewardID;references:RewardID;constraint:OnDelete:RESTRICT"`
}

func (NilaiRewardBank) TableName() string {
	return "nilai_reward_bank"
}
