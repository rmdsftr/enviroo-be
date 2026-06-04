package models

import "time"

type StatusSetoran string

const (
	StatusBerhasil StatusSetoran = "berhasil"
	StatusPending  StatusSetoran = "pending"
	StatusGagal    StatusSetoran = "gagal"
)

type SetoranNasabah struct {
	SetoranID     string        `gorm:"column:setoran_id;type:varchar(100);primaryKey"`
	AdminID       string        `gorm:"column:admin_id;type:varchar(100);index"`
	NasabahID     string        `gorm:"column:nasabah_id;type:varchar(100);index"`
	PenimbanganID string        `gorm:"column:penimbangan_id;type:varchar(100);index"`

	CreatedAt     time.Time     `gorm:"column:created_at;autoCreateTime"`

	TotalItem     int           `gorm:"column:total_item;default:0"`
 
	StatusSetoran StatusSetoran `gorm:"column:status_setoran;type:status_setoran_enum;default:'pending'"`
	BuktiViaManual string        `gorm:"column:bukti_via_manual;type:varchar(255)"`
 
	Admin         Admin         `gorm:"foreignKey:AdminID;references:AdminID"`
	Nasabah       Nasabah       `gorm:"foreignKey:NasabahID;references:NasabahID"`
	Penimbangan   Penimbangan   `gorm:"foreignKey:PenimbanganID;references:PenimbanganID"`
}

func (SetoranNasabah) TableName() string {
	return "setoran_nasabah"
}