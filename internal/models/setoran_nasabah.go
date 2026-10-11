package models

import "time"

type SetoranNasabah struct {
	SetoranID     string `gorm:"column:setoran_id;type:varchar(21);primaryKey"`
	AdminID       string `gorm:"column:admin_id;type:varchar(13);not null"`
	NasabahID     string `gorm:"column:nasabah_id;type:varchar(15);not null;index"`
	PenimbanganID string `gorm:"column:penimbangan_id;type:varchar(17);not null;index"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;not null"`

	TotalItem int `gorm:"column:total_item;default:0;not null"`

	BuktiViaManual string `gorm:"column:bukti_via_manual;type:varchar(255)"`

	Admin       Admin       `gorm:"foreignKey:AdminID;references:AdminID;constraint:OnDelete:RESTRICT"`
	Nasabah     Nasabah     `gorm:"foreignKey:NasabahID;references:NasabahID;constraint:OnDelete:RESTRICT"`
	Penimbangan Penimbangan `gorm:"foreignKey:PenimbanganID;references:PenimbanganID;constraint:OnDelete:RESTRICT"`
}

func (SetoranNasabah) TableName() string {
	return "setoran_nasabah"
}
