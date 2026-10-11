package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type JenisTerimaEnum string

const (
	JenisTerimaBagiHasilBSU JenisTerimaEnum = "bagi_hasil_bsu"
	JenisTerimaBagiHasilBSI JenisTerimaEnum = "bagi_hasil_bsi"
)

type TransportDetailItem struct {
	BsuID     string  `json:"bsu_id"`
	NamaBsu   string  `json:"nama_bsu"`
	Transport float64 `json:"transport"`
}

type TransportDetail []TransportDetailItem

func (t TransportDetail) Value() (driver.Value, error) {
	if t == nil {
		return nil, nil
	}
	return json.Marshal(t)
}

func (t *TransportDetail) Scan(value interface{}) error {
	if value == nil {
		*t = nil
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case string:
		b = []byte(v)
	case []byte:
		b = v
	default:
		return fmt.Errorf("TransportDetail: cannot scan type %T", value)
	}
	return json.Unmarshal(b, t)
}

type PenerimaDistribusiSisa struct {
	PenerimaSisaID string `json:"penerima_sisa_id" gorm:"column:penerima_sisa_id;primaryKey;size:22"`

	DistribusiID string `json:"distribusi_id" gorm:"column:distribusi_id;size:19;not null;uniqueIndex:idx_penerima_distribusi_sisa_distribusi_bank"`

	BankID string `json:"bank_id" gorm:"column:bank_id;size:9;not null;index;uniqueIndex:idx_penerima_distribusi_sisa_distribusi_bank"`

	JenisPenerimaan JenisTerimaEnum `json:"jenis_penerimaan" gorm:"column:jenis_penerimaan;type:jenis_terima_enum;not null"`

	NominalDiterima float64 `json:"nominal_diterima" gorm:"column:nominal_diterima;type:decimal(20,4);not null"`
	Transportasi    float64 `json:"transportasi"     gorm:"column:transportasi;type:decimal(20,4);not null"`
	Porsi           float64 `json:"porsi"            gorm:"column:porsi;type:decimal(20,4);not null"`

	SatuanNominal SatuanRewardEnum `json:"satuan_nominal" gorm:"column:satuan_nominal;type:satuan_reward_enum;not null"`

	TransportDetail TransportDetail `json:"transport_detail,omitempty" gorm:"column:transport_detail;type:jsonb"`

	DistribusiSisa DistribusiSisa `json:"distribusi_sisa,omitempty" gorm:"foreignKey:DistribusiID;references:DistribusiID;constraint:OnDelete:CASCADE"`
	BankSampah     BankSampah     `json:"bank_sampah,omitempty"     gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:RESTRICT"`
}

func (PenerimaDistribusiSisa) TableName() string {
	return "penerima_distribusi_sisa"
}
