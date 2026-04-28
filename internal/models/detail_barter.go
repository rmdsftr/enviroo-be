package models

type DetailBarter struct {
	PenjualanID   string  `gorm:"column:penjualan_id;type:varchar(100);primaryKey"`
	SembakoID     string  `gorm:"column:sembako_id;type:varchar(100);primaryKey"`
	Qty           float64 `gorm:"column:qty;type:decimal(20,4);not null"`
	HargaPoin     float64 `gorm:"column:harga_poin;type:decimal(20,4);not null"`
	SubtotalPoin  float64 `gorm:"column:subtotal_poin;type:decimal(20,4);not null"`

	Penjualan Penjualan     `gorm:"foreignKey:PenjualanID;references:PenjualanID"`
	Sembako   KatalogSembako `gorm:"foreignKey:SembakoID;references:SembakoID"`
}

func (DetailBarter) TableName() string {
	return "detail_barter"
}