package models

type Sembako struct{
	BarangID int `gorm:"column:barang_id;primaryKey;autoIncrement"`
	NamaBarang string `gorm:"column:nama_barang;type:varchar(255)"`
}

func (Sembako) TableName() string{
	return "sembako"
}