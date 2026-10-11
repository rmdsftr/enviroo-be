package models

import "time"

type KategoriSampah struct {
	KategoriID int       `gorm:"column:kategori_id;primaryKey;autoIncrement"`
	Kategori   string    `gorm:"column:kategori;size:100;not null"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime;not null"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime;not null"`

	KatalogSampah []KatalogSampah `gorm:"foreignKey:KategoriID"`
}

func (KategoriSampah) TableName() string {
	return "kategori_sampah"
}