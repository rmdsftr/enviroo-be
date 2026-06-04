package models

import "time"

type SatuanEnum string

const (
	SatuanKG  SatuanEnum = "kg"
	SatuanPCS SatuanEnum = "pcs"
	SatuanLITER SatuanEnum = "liter"
)

type KatalogSampah struct {
	SampahID     string           `gorm:"column:sampah_id;primaryKey;size:100"`
	NamaSampah   string           `gorm:"column:nama_sampah;size:255"`
	PhotoURL     string           `gorm:"column:photo_url;size:255"`
	Satuan       SatuanEnum       `gorm:"column:satuan;type:satuan_enum"`
	SyaratPemilahan string       `gorm:"column:syarat_pemilahan;type:text"`
	BankID       string           `gorm:"column:bank_id;size:100"`
	KategoriID   int              `gorm:"column:kategori_id"`
	RewardID     int              `gorm:"column:reward_id"`
	CreatedAt    time.Time        `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time        `gorm:"column:updated_at;autoUpdateTime"`
 
	// Relasi
	Kategori     KategoriSampah   `gorm:"foreignKey:KategoriID;references:KategoriID"`
	Bank        BankSampah    `gorm:"foreignKey:BankID"` // aktifkan kalau struct BankSampah ada
	Reward      Reward        `gorm:"foreignKey:RewardID;references:RewardID"`
}

func (KatalogSampah) TableName() string {
	return "katalog_sampah"
}