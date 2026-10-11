package models

type DetailPenarikanBarang struct {
	PenarikanID string `gorm:"column:penarikan_id;type:varchar(24);primaryKey" json:"penarikan_id"`

	ProdukID string `gorm:"column:produk_id;type:varchar(12);primaryKey" json:"produk_id"`

	Qty float64 `gorm:"column:qty;type:decimal(20,4);not null" json:"qty"`
	NilaiPoin float64 `gorm:"column:nilai_poin;type:decimal(20,4);not null" json:"nilai_poin"`
	SubtotalPoin float64 `gorm:"column:subtotal_poin;type:decimal(20,4);not null" json:"subtotal_poin"`

	Penarikan *Penarikan `gorm:"foreignKey:PenarikanID;references:PenarikanID;constraint:OnDelete:CASCADE"`

	KatalogBarang *KatalogBarang `gorm:"foreignKey:ProdukID;references:ProdukID;constraint:OnDelete:RESTRICT"`
}

func (DetailPenarikanBarang) TableName() string {
	return "detail_penarikan_barang"
}
