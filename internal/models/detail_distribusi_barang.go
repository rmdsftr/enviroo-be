package models

type DetailDistribusiBarang struct {
	DisbaID   string  `gorm:"column:disba_id;type:varchar(18);primaryKey" json:"disba_id"`
	ProdukID  string  `gorm:"column:produk_id;type:varchar(12);primaryKey" json:"produk_id"`

	Item         float64 `gorm:"column:item;type:decimal(20,4);not null" json:"item"`
	Poin         float64 `gorm:"column:poin;type:decimal(20,4);not null" json:"poin"`
	SubtotalPoin float64 `gorm:"column:subtotal_poin;type:decimal(20,4);not null" json:"subtotal_poin"`

	StokBsuSebelum float64 `gorm:"column:stok_bsu_sebelum;type:decimal(20,4);not null" json:"stok_bsu_sebelum"`
	StokBsuSesudah float64 `gorm:"column:stok_bsu_sesudah;type:decimal(20,4);not null" json:"stok_bsu_sesudah"`
	StokBsiSebelum float64 `gorm:"column:stok_bsi_sebelum;type:decimal(20,4);not null" json:"stok_bsi_sebelum"`
	StokBsiSesudah float64 `gorm:"column:stok_bsi_sesudah;type:decimal(20,4);not null" json:"stok_bsi_sesudah"`

	DistribusiBarangBsu DistribusiBarangBsu `gorm:"foreignKey:DisbaID;references:DisbaID;constraint:OnDelete:CASCADE"`
	KatalogBarang       KatalogBarang       `gorm:"foreignKey:ProdukID;references:ProdukID;constraint:OnDelete:RESTRICT"`
}

func (DetailDistribusiBarang) TableName() string {
	return "detail_distribusi_barang"
}
