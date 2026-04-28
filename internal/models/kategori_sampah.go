package models

import "time"

type KategoriSampah struct {
	KategoriID int       `gorm:"column:kategori_id;primaryKey;autoIncrement"`
	Kategori   string    `gorm:"column:kategori;size:255"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relasi (optional, biar bisa preload)
	KatalogSampah []KatalogSampah `gorm:"foreignKey:KategoriID"`
}

func (KategoriSampah) TableName() string {
	return "kategori_sampah"
}