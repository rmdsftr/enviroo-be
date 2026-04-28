package models

type StokSampah struct {
	BankID   string `gorm:"column:bank_id;type:varchar(100);primaryKey"`
	SampahID string `gorm:"column:sampah_id;type:varchar(100);primaryKey"`
	Stok     float64    `gorm:"column:stok;type:decimal(20,4);not null;default:0"`

	BankSampah    BankSampah    `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:CASCADE"`
	KatalogSampah KatalogSampah `gorm:"foreignKey:SampahID;references:SampahID;constraint:OnDelete:CASCADE"`
}

func (StokSampah) TableName() string {
	return "stok_sampah"
}
