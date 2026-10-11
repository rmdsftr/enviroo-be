package models

type DetailPengangkutan struct {
	PengangkutanID string `gorm:"column:pengangkutan_id;primaryKey;size:18"`

	SampahID string `gorm:"column:sampah_id;primaryKey;size:13"`

	Qty float64 `gorm:"column:qty;type:decimal(20,4);not null"`

	Pengangkutan PengangkutanSampah `gorm:"foreignKey:PengangkutanID;references:PengangkutanID;constraint:OnDelete:CASCADE"`
	Sampah KatalogSampah `gorm:"foreignKey:SampahID;references:SampahID;constraint:OnDelete:RESTRICT"`
}

func (DetailPengangkutan) TableName() string {
	return "detail_pengangkutan"
} 