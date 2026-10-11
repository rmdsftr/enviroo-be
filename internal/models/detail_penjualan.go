package models

type DetailPenjualan struct {
	PenjualanID       string  `gorm:"column:penjualan_id;type:varchar(18);primaryKey"`
	SampahID          string  `gorm:"column:sampah_id;type:varchar(13);primaryKey"`
	Qty               float64 `gorm:"column:qty;type:decimal(20,4);not null"`
	HargaJual         float64 `gorm:"column:harga_jual;type:decimal(20,4);not null"`
	SubtotalPenjualan float64 `gorm:"column:subtotal_penjualan;type:decimal(20,4);not null"`

	HargaNasabahSnapshot float64 `gorm:"column:harga_nasabah_snapshot;type:decimal(20,4);not null"`

	Penjualan Penjualan     `gorm:"foreignKey:PenjualanID;references:PenjualanID;constraint:OnDelete:CASCADE"`
	Sampah    KatalogSampah `gorm:"foreignKey:SampahID;references:SampahID;constraint:OnDelete:RESTRICT"`
}

func (DetailPenjualan) TableName() string {
	return "detail_penjualan"
}