package models

import "time"

type KatalogSampah struct {
	SampahID        string    `gorm:"column:sampah_id;primaryKey;size:100"`
	SarokID         int       `gorm:"column:sarok_id"`
	PhotoURL        string    `gorm:"column:photo_url;size:255"`
	SyaratPemilahan string    `gorm:"column:syarat_pemilahan;type:text"`
	BankID          string    `gorm:"column:bank_id;size:100"`
	KategoriID      int       `gorm:"column:kategori_id"`
	RewardID        int       `gorm:"column:reward_id"`
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relasi
	Kategori KategoriSampah `gorm:"foreignKey:KategoriID;references:KategoriID"`
	Bank     BankSampah     `gorm:"foreignKey:BankID"` // aktifkan kalau struct BankSampah ada
	Reward   Reward         `gorm:"foreignKey:RewardID;references:RewardID"`
	Sarok    Sampah         `gorm:"foreignKey:SarokID;references:SarokID"`
}

func (KatalogSampah) TableName() string {
	return "katalog_sampah"
}
