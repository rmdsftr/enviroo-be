package models

import "time"

type EntitasEnum string

const (
	EntitasNasabah    EntitasEnum = "nasabah"
	EntitasBankSampah EntitasEnum = "bank_sampah"
) 

type TabunganSampah struct {
	TabunganID string      `gorm:"column:tabungan_id;type:varchar(100);primaryKey" json:"tabungan_id"`
	NasabahID  *string     `gorm:"column:nasabah_id;type:varchar(100)" json:"nasabah_id"`
	BankID     *string     `gorm:"column:bank_id;type:varchar(100)" json:"bank_id"`
	SampahID   string      `gorm:"column:sampah_id;type:varchar(100)" json:"sampah_id"`
	Entitas    EntitasEnum `gorm:"column:entitas;type:entitas_enum" json:"entitas"`
	Qty        float64     `gorm:"column:qty;type:decimal(20,4)" json:"qty"`
	SisaQty    float64     `gorm:"column:sisa_qty;type:decimal(20,4)" json:"sisa_qty"`
	CreatedAt  time.Time   `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	SourceID   *string     `gorm:"column:source_id;type:varchar(100)" json:"source_id"`

	Nasabah       *Nasabah       `gorm:"foreignKey:NasabahID;references:NasabahID"`
	BankSampah    *BankSampah    `gorm:"foreignKey:BankID;references:BankID"`
	KatalogSampah *KatalogSampah `gorm:"foreignKey:SampahID;references:SampahID"`
}

func (TabunganSampah) TableName() string {
	return "tabungan_sampah"
}
