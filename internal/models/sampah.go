package models

type SatuanEnum string

const (
	SatuanKG  SatuanEnum = "kg"
	SatuanPCS SatuanEnum = "pcs"
	SatuanLITER SatuanEnum = "liter"
)

type Sampah struct{
	SarokID int `gorm:"column:sarok_id;autoIncrement;primaryKey"`
	NamaSampah string `gorm:"column:nama_sampah;type:varchar(255)"`
	Satuan SatuanEnum `gorm:"column:satuan;type:satuan_enum"`
}

func (Sampah) TableName() string{
	return "sampah"
}