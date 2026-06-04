package models

type DetailPenarikanSembako struct {
	PenarikanID string `gorm:"column:penarikan_id;type:varchar(100);primaryKey" json:"penarikan_id"`

	SembakoID string `gorm:"column:sembako_id;type:varchar(100);primaryKey" json:"sembako_id"`

	Qty float64 `gorm:"column:qty;type:decimal(20,4)" json:"qty"`
	NilaiPoin float64 `gorm:"column:nilai_poin;type:decimal(20,4)" json:"nilai_poin"`
	SubtotalPoin float64 `gorm:"column:subtotal_poin;type:decimal(20,4)" json:"subtotal_poin"`

	Penarikan *Penarikan `gorm:"foreignKey:PenarikanID;references:PenarikanID"`

	KatalogSembako *KatalogSembako `gorm:"foreignKey:SembakoID;references:SembakoID"`
}

func (DetailPenarikanSembako) TableName() string {
	return "detail_penarikan_sembako"
}