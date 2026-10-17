package models

type Satuan struct {
	SatuanID int    `gorm:"column:satuan_id;primaryKey;autoIncrement"`
	Satuan   string `gorm:"column:satuan;type:varchar(20);unique;not null"`
}

func (Satuan) TableName() string {
	return "satuan"
}
