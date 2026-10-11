package models

type Barang struct{
	BarangID int `gorm:"column:barang_id;primaryKey;autoIncrement"`
	NamaBarang string `gorm:"column:nama_barang;type:varchar(100);not null"`
}

func (Barang) TableName() string{
	return "barang"
}
