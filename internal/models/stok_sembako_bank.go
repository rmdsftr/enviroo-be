package models

type StokSembakoBank struct {
	BankID    string  `gorm:"column:bank_id;type:varchar(100);primaryKey"`
	SembakoID string  `gorm:"column:sembako_id;type:varchar(100);primaryKey"`
	Stok      float64 `gorm:"column:stok;type:decimal(20,4);not null;default:0"`

	// Relations
	BankSampah BankSampah     `gorm:"foreignKey:BankID;references:BankID"`
	Sembako    KatalogSembako `gorm:"foreignKey:SembakoID;references:SembakoID"`
}

func (StokSembakoBank) TableName() string {
	return "stok_sembako_bank"
}