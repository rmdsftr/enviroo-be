package models

import "time"

type PaketSetoranBank struct {
	PaketID string `gorm:"column:paket_id;primaryKey;size:100"`

	PengangkutanID string `gorm:"column:pengangkutan_id;size:100"`

	BSIID string `gorm:"column:bsi_id;size:100"`
	BSUId string `gorm:"column:bsu_id;size:100"`

	AdminBSIID string `gorm:"column:admin_bsi_id;size:100"`
	AdminBSUId string `gorm:"column:admin_bsu_id;size:100"`

	TotalItem int `gorm:"column:total_item"`

	CreatedAt time.Time `gorm:"column:created_at"`

	StatusSetoran StatusSetoran `gorm:"column:status_setoran;type:status_setoran_enum"`

	DetailPaket []DetailPaket `gorm:"foreignKey:PaketID"`
	PengangkutanSampah PengangkutanSampah `gorm:"foreignKey:PengangkutanID;references:PengangkutanID;constraint:OnDelete:CASCADE"`
}

func (PaketSetoranBank) TableName() string {
	return "paket_setoran_bank"
}