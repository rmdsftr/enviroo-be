package models

import "time"

type KatalogSembako struct {
	SembakoID string `gorm:"column:sembako_id;type:varchar(100);primaryKey" json:"sembako_id"`

	BankID *string `gorm:"column:bank_id;type:varchar(100)" json:"bank_id"`

	BarangID int `gorm:"column:barang_id" json:"barang_id"`

	PhotoURL *string `gorm:"column:photo_url;type:varchar(255)" json:"photo_url"`

	NilaiPoin float64 `gorm:"column:nilai_poin;type:decimal(20,4)" json:"nilai_poin"`

	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy *string   `gorm:"column:created_by;type:varchar(100)" json:"created_by"`

	UpdatedAt time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
	UpdatedBy *string   `gorm:"column:updated_by;type:varchar(100)" json:"updated_by"`

	Bank *BankSampah `gorm:"foreignKey:BankID;references:BankID"`
	MasterSembako Sembako `gorm:"foreignKey:BarangID;references:BarangID"`
}

func (KatalogSembako) TableName() string {
	return "katalog_sembako"
}