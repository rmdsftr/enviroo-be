package models

type DetailSetoranNasabah struct {
	SampahID      string `gorm:"column:sampah_id;type:varchar(100);primaryKey"`
	SetoranID     string `gorm:"column:setoran_id;type:varchar(100);primaryKey"`

	Qty           float64     `gorm:"column:qty;type:decimal(20,4);not null"`
	NilaiPoin     float64 `gorm:"column:nilai_poin;type:decimal(20,4);not null"`
	SubtotalPoin  float64 `gorm:"column:subtotal_poin;type:decimal(20,4);not null"`

	Sampah        KatalogSampah `gorm:"foreignKey:SampahID;references:SampahID"`
	Setoran       SetoranNasabah `gorm:"foreignKey:SetoranID;references:SetoranID"`
}

func (DetailSetoranNasabah) TableName() string {
	return "detail_setoran_nasabah"
}