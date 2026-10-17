package models

type Sampah struct{
	SarokID int `gorm:"column:sarok_id;autoIncrement;primaryKey"`
	NamaSampah string `gorm:"column:nama_sampah;type:varchar(100);not null"`
	SatuanID int `gorm:"column:satuan_id;not null"`

	Satuan Satuan `gorm:"foreignKey:SatuanID;references:SatuanID;constraint:OnDelete:RESTRICT"`
}

func (Sampah) TableName() string{
	return "sampah"
}