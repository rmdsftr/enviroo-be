package models

import "time"

type SatuanWaktuEnum string

const (
	SatuanWaktuMinggu SatuanWaktuEnum = "minggu"
	SatuanWaktuBulan  SatuanWaktuEnum = "bulan"
)

type SettingPenarikan struct {
	SettingID   string           `gorm:"column:setting_id;type:varchar(11);primaryKey" json:"setting_id"`
	BankID      string           `gorm:"column:bank_id;type:varchar(9);not null;uniqueIndex" json:"bank_id"`
	MinWaktu    *int             `gorm:"column:min_waktu" json:"min_waktu"`
	SatuanWaktu *SatuanWaktuEnum `gorm:"column:satuan_waktu;type:satuan_waktu_enum" json:"satuan_waktu"`
	MinRupiah   *float64         `gorm:"column:min_rupiah;type:decimal(20,4)" json:"min_rupiah"`
	MinPoin     *float64         `gorm:"column:min_poin;type:decimal(20,4)" json:"min_poin"`
	CreatedAt   time.Time        `gorm:"column:created_at;autoCreateTime;not null" json:"created_at"`
	CreatedBy   string           `gorm:"column:created_by;type:varchar(25);not null" json:"created_by"`
	UpdatedAt   time.Time        `gorm:"column:updated_at;autoUpdateTime;not null" json:"updated_at"`
	UpdatedBy   string           `gorm:"column:updated_by;type:varchar(25);not null" json:"updated_by"`
}

func (SettingPenarikan) TableName() string {
	return "setting_penarikan"
}
