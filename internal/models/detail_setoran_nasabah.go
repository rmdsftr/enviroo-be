package models

type DetailSetoranNasabah struct {
	SampahID      string `gorm:"column:sampah_id;type:varchar(13);primaryKey"`
	SetoranID     string `gorm:"column:setoran_id;type:varchar(21);primaryKey;index"`

	Qty           float64     `gorm:"column:qty;type:decimal(20,4);not null"`

	Sampah        KatalogSampah `gorm:"foreignKey:SampahID;references:SampahID;constraint:OnDelete:RESTRICT"`
	Setoran       SetoranNasabah `gorm:"foreignKey:SetoranID;references:SetoranID;constraint:OnDelete:CASCADE"`
}

func (DetailSetoranNasabah) TableName() string {
	return "detail_setoran_nasabah"
}