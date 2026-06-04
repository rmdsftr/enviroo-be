package models

type JenisTerimaEnum string

const (
	JenisTerimaBagiHasilBSU JenisTerimaEnum = "bagi_hasil_bsu"
	JenisTerimaBagiHasilBSI JenisTerimaEnum = "bagi_hasil_bsi"
)

type PenerimaDistribusiSisa struct {
	PenerimaSisaID string `json:"penerima_sisa_id" gorm:"column:penerima_sisa_id;primaryKey;size:100"`

	DistribusiID string `json:"distribusi_id" gorm:"column:distribusi_id;size:100"`

	BankID string `json:"bank_id" gorm:"column:bank_id;size:100"`

	JenisPenerimaan JenisTerimaEnum `json:"jenis_penerimaan" gorm:"column:jenis_penerimaan;type:jenis_terima_enum"`

	NominalDiterima float64 `json:"nominal_diterima" gorm:"column:nominal_diterima;type:decimal(20,4)"`
	Transportasi float64 `json:"transportasi" gorm:"column:transportasi;type:decimal(20,4)"`
	Porsi float64 `json:"porsi" gorm:"column:porsi;type:decimal(20,4)"`

	SatuanNominal string `json:"satuan_nominal" gorm:"column:satuan_nominal;type:satuan_reward_enum"`

	DiantarOleh *AntarEnum `json:"diantar_oleh,omitempty" gorm:"column:diantar_oleh;type:antar_enum"`

	DistribusiSisa DistribusiSisa `json:"distribusi_sisa,omitempty" gorm:"foreignKey:DistribusiID;references:DistribusiID"`

	BankSampah BankSampah `json:"bank_sampah,omitempty" gorm:"foreignKey:BankID;references:BankID"`
}

func (PenerimaDistribusiSisa) TableName() string {
	return "penerima_distribusi_sisa"
}