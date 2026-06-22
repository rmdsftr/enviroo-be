package models

type DetailPengangkutan struct {
	PengangkutanID string `gorm:"column:pengangkutan_id;primaryKey;size:100"`

	SampahID string `gorm:"column:sampah_id;primaryKey;size:100"`

	Qty float64 `gorm:"column:qty;type:decimal(20,4)"`

	Pengangkutan PengangkutanSampah `gorm:"foreignKey:PengangkutanID;references:PengangkutanID"`
	Sampah KatalogSampah `gorm:"foreignKey:SampahID;references:SampahID"`
}

func (DetailPengangkutan) TableName() string {
	return "detail_pengangkutan"
} 