package models

type StokBarang struct {
	BankID    string  `gorm:"column:bank_id;type:varchar(9);primaryKey" json:"bank_id"`
	ProdukID  string  `gorm:"column:produk_id;type:varchar(12);primaryKey" json:"produk_id"`
	Stok      float64 `gorm:"column:stok;type:decimal(20,4)" json:"stok"`
}

func (StokBarang) TableName() string {
	return "stok_barang"
}
