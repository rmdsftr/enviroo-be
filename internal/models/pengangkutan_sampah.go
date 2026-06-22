package models

import "github.com/google/uuid"


type PengangkutanSampah struct {
	PengangkutanID string `gorm:"column:pengangkutan_id;primaryKey;size:100"`

	BSIID string `gorm:"column:bsi_id;size:100"`
	BSUId string `gorm:"column:bsu_id;size:100"`

	AdminBSIID *string `gorm:"column:admin_bsi_id;size:100"`
	AdminBSUId *string `gorm:"column:admin_bsu_id;size:100"`

	BuktiFoto *string `gorm:"column:bukti_foto;type:varchar(255)"`

	JadwalID uuid.UUID `gorm:"column:jadwal_id;type:uuid"`

	TotalItem  float64 `gorm:"column:total_item;type:decimal(20,4)"`
	IsMandiri  bool    `gorm:"column:is_mandiri;default:false"`

	RiwayatPengangkutan []RiwayatPengangkutan `gorm:"foreignKey:PengangkutanID"`
}

func (PengangkutanSampah) TableName() string {
	return "pengangkutan_sampah"
}
