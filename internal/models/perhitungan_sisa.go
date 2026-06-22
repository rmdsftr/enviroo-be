package models

type PerhitunganSisa struct {
	BagiHasilID string `json:"bagi_hasil_id" gorm:"column:bagi_hasil_id;primaryKey;size:100"`

	TabunganID string `json:"tabungan_id" gorm:"column:tabungan_id;primaryKey;size:100"`

	QtyDipakai float64 `json:"qty_dipakai" gorm:"column:qty_dipakai;type:decimal(20,4)"`
	NilaiDipakai float64 `gorm:"column:nilai_dipakai;type:decimal(20,4)"`

	PenerimaSisaID *string `json:"penerima_sisa_id" gorm:"column:penerima_sisa_id;size:100"`

	BagiHasil BagiHasil `json:"bagi_hasil" gorm:"foreignKey:BagiHasilID;references:BagiHasilID"`

	PenerimaDistribusiSisa PenerimaDistribusiSisa `json:"penerima_distribusi_sisa,omitempty" gorm:"foreignKey:PenerimaSisaID;references:PenerimaSisaID"`

	TabunganSampah TabunganSampah `json:"tabungan_sampah,omitempty" gorm:"foreignKey:TabunganID;references:TabunganID"`
}

func (PerhitunganSisa) TableName() string {
	return "perhitungan_sisa"
}