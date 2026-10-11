package models

import "time"

type KatalogBarang struct {
	ProdukID string `gorm:"column:produk_id;type:varchar(12);primaryKey" json:"produk_id"`

	BankID *string `gorm:"column:bank_id;type:varchar(9);not null;uniqueIndex:idx_katalog_barang_bank_barang" json:"bank_id"`

	BarangID int `gorm:"column:barang_id;not null;uniqueIndex:idx_katalog_barang_bank_barang" json:"barang_id"`

	PhotoURL *string `gorm:"column:photo_url;type:varchar(255)" json:"photo_url"`

	NilaiPoin float64 `gorm:"column:nilai_poin;type:decimal(20,4);not null" json:"nilai_poin"`

	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP;not null" json:"created_at"`
	CreatedBy *string   `gorm:"column:created_by;type:varchar(25);not null" json:"created_by"`

	UpdatedAt time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP;not null" json:"updated_at"`
	UpdatedBy *string   `gorm:"column:updated_by;type:varchar(25);not null" json:"updated_by"`

	Bank         *BankSampah `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:RESTRICT"`
	MasterBarang Barang      `gorm:"foreignKey:BarangID;references:BarangID;constraint:OnDelete:RESTRICT"`
}

func (KatalogBarang) TableName() string {
	return "katalog_barang"
}
