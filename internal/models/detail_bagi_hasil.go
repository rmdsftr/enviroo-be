package models

type DetailBagiHasil struct {
	PenerimaID string `gorm:"column:penerima_id;type:varchar(100);primaryKey" json:"penerima_id"`
	SampahID   string `gorm:"column:sampah_id;type:varchar(100);primaryKey" json:"sampah_id"`

	Qty float64 `gorm:"column:qty;type:decimal(20,4)" json:"qty"`

	HargaItem float64 `gorm:"column:harga_item;type:decimal(20,4)" json:"harga_item"`

	SubtotalHarga float64 `gorm:"column:subtotal_harga;type:decimal(20,4)" json:"subtotal_harga"`

	PenerimaBagiHasil *PenerimaBagiHasil `gorm:"foreignKey:PenerimaID;references:PenerimaID"`
	KatalogSampah     *KatalogSampah     `gorm:"foreignKey:SampahID;references:SampahID"`
}

func (DetailBagiHasil) TableName() string {
	return "detail_bagi_hasil"
}