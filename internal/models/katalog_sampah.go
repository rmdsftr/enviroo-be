package models

import "time"

type KatalogSampah struct {
	SampahID        string    `gorm:"column:sampah_id;primaryKey;size:13"`
	SarokID         int       `gorm:"column:sarok_id;not null;uniqueIndex:idx_katalog_sampah_sarok_reward_bank"`
	PhotoURL        string    `gorm:"column:photo_url;size:255"`
	SyaratPemilahan string    `gorm:"column:syarat_pemilahan;type:text"`
	BankID          string    `gorm:"column:bank_id;size:9;not null;index;uniqueIndex:idx_katalog_sampah_sarok_reward_bank"`
	KategoriID      int       `gorm:"column:kategori_id;not null"`
	RewardID        int       `gorm:"column:reward_id;not null;uniqueIndex:idx_katalog_sampah_sarok_reward_bank"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime;not null"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime;not null"`

	Kategori KategoriSampah `gorm:"foreignKey:KategoriID;references:KategoriID;constraint:OnDelete:RESTRICT"`
	Bank     BankSampah     `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:RESTRICT"`
	Reward   Reward         `gorm:"foreignKey:RewardID;references:RewardID;constraint:OnDelete:RESTRICT"`
	Sarok    Sampah         `gorm:"foreignKey:SarokID;references:SarokID"`
}

func (KatalogSampah) TableName() string {
	return "katalog_sampah"
}
