package models

type PengangkutanSampah struct {
	PengangkutanID string `gorm:"column:pengangkutan_id;primaryKey;size:18"`

	BSIID string `gorm:"column:bsi_id;size:9;not null;index"`
	BSUId string `gorm:"column:bsu_id;size:9;not null;index"`

	AdminBSIID *string `gorm:"column:admin_bsi_id;size:13"`
	AdminBSUId *string `gorm:"column:admin_bsu_id;size:13"`

	BuktiFoto *string `gorm:"column:bukti_foto;type:varchar(255)"`

	JadwalID string `gorm:"column:jadwal_id;type:varchar(15);not null"`

	TotalItem  float64 `gorm:"column:total_item;type:decimal(20,4)"`
	IsMandiri  bool    `gorm:"column:is_mandiri;default:false;not null"`

	BSI                 BankSampah            `gorm:"foreignKey:BSIID;references:BankID;constraint:OnDelete:RESTRICT"`
	BSU                 BankSampah            `gorm:"foreignKey:BSUId;references:BankID;constraint:OnDelete:RESTRICT"`
	AdminBSI             *Admin                `gorm:"foreignKey:AdminBSIID;references:AdminID;constraint:OnDelete:RESTRICT"`
	AdminBSU             *Admin                `gorm:"foreignKey:AdminBSUId;references:AdminID;constraint:OnDelete:RESTRICT"`
	Jadwal               Jadwal                `gorm:"foreignKey:JadwalID;references:JadwalID"`
	RiwayatPengangkutan []RiwayatPengangkutan `gorm:"foreignKey:PengangkutanID"`
}

func (PengangkutanSampah) TableName() string {
	return "pengangkutan_sampah"
}
