package models

type DetailPenjualan struct {
	PenjualanID       string  `gorm:"column:penjualan_id;type:varchar(100);primaryKey"`
	SampahID          string  `gorm:"column:sampah_id;type:varchar(100);primaryKey"`
	Qty               float64 `gorm:"column:qty;type:decimal(20,4);not null"`
	PoinJual          float64 `gorm:"column:poin_jual;type:decimal(20,4);not null"`
	NilaiPoin         float64 `gorm:"column:nilai_poin;type:decimal(20,4);not null"`
	NilaiKonversi     float64 `gorm:"column:nilai_konversi;type:decimal(20,4);not null"`
	SubtotalPoin      float64 `gorm:"column:subtotal_poin;type:decimal(20,4);not null"`
	SubtotalKonversi  float64 `gorm:"column:subtotal_konversi;type:decimal(20,4);not null"`

	Penjualan Penjualan     `gorm:"foreignKey:PenjualanID;references:PenjualanID"`
	Sampah    KatalogSampah `gorm:"foreignKey:SampahID;references:SampahID"`
}

func (DetailPenjualan) TableName() string {
	return "detail_penjualan"
}