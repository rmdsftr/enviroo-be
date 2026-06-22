package models

type StokSembako struct {
	SembakoID string  `gorm:"column:sembako_id;primaryKey" json:"sembako_id"`
	BankID    string  `gorm:"column:bank_id;primaryKey" json:"bank_id"`
	Stok      float64 `gorm:"column:stok;type:decimal(20,4)" json:"stok"`
}

func (StokSembako) TableName() string {
	return "stok_sembako"
}