package models

import "time"

type EntitasEnum string

const (
	EntitasNasabah    EntitasEnum = "nasabah"
	EntitasBankSampah EntitasEnum = "bank_sampah"
) 

type TabunganSampah struct {
	TabunganID string      `gorm:"column:tabungan_id;type:varchar(24);primaryKey" json:"tabungan_id"`
	NasabahID  *string     `gorm:"column:nasabah_id;type:varchar(15);index" json:"nasabah_id"`
	BankID     *string     `gorm:"column:bank_id;type:varchar(9);index" json:"bank_id"`
	SampahID   string      `gorm:"column:sampah_id;type:varchar(13);not null;index" json:"sampah_id"`
	Entitas    EntitasEnum `gorm:"column:entitas;type:entitas_enum;not null" json:"entitas"`
	Qty        float64     `gorm:"column:qty;type:decimal(20,4);not null" json:"qty"`
	SisaQty    float64     `gorm:"column:sisa_qty;type:decimal(20,4);not null" json:"sisa_qty"`
	CreatedAt  time.Time   `gorm:"column:created_at;default:CURRENT_TIMESTAMP;not null" json:"created_at"`
	SourceID   *string     `gorm:"column:source_id;type:varchar(21);not null;index" json:"source_id"`

	Nasabah       *Nasabah       `gorm:"foreignKey:NasabahID;references:NasabahID;constraint:OnDelete:RESTRICT"`
	BankSampah    *BankSampah    `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:RESTRICT"`
	KatalogSampah *KatalogSampah `gorm:"foreignKey:SampahID;references:SampahID;constraint:OnDelete:RESTRICT"`
}

func (TabunganSampah) TableName() string {
	return "tabungan_sampah"
}
