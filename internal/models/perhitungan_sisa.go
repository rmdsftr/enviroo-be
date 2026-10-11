package models

type PerhitunganSisa struct {
	BagiHasilID string `json:"bagi_hasil_id" gorm:"column:bagi_hasil_id;primaryKey;size:19"`

	TabunganID string `json:"tabungan_id" gorm:"column:tabungan_id;primaryKey;size:24"`

	QtyDipakai float64 `json:"qty_dipakai" gorm:"column:qty_dipakai;type:decimal(20,4)"`
	NilaiDipakai float64 `gorm:"column:nilai_dipakai;type:decimal(20,4)"`

	PenerimaSisaID *string `json:"penerima_sisa_id" gorm:"column:penerima_sisa_id;size:22"`

	BagiHasil BagiHasil `json:"bagi_hasil" gorm:"foreignKey:BagiHasilID;references:BagiHasilID;constraint:OnDelete:CASCADE"`

	PenerimaDistribusiSisa *PenerimaDistribusiSisa `json:"penerima_distribusi_sisa,omitempty" gorm:"foreignKey:PenerimaSisaID;references:PenerimaSisaID;constraint:OnDelete:SET NULL"`

	TabunganSampah TabunganSampah `json:"tabungan_sampah,omitempty" gorm:"foreignKey:TabunganID;references:TabunganID;constraint:OnDelete:CASCADE"`
}

func (PerhitunganSisa) TableName() string {
	return "perhitungan_sisa"
}