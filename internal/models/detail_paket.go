package models

type DetailPaket struct {
	PaketID string `gorm:"column:paket_id;primaryKey;size:100"`

	SampahID string `gorm:"column:sampah_id;primaryKey;size:100"`

	Qty float64 `gorm:"column:qty;type:decimal(20,4)"`
}

func (DetailPaket) TableName() string {
	return "detail_paket"
} 