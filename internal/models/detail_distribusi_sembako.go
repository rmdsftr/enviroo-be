package models

type DetailDistribusiSembako struct {
	SembakoID  string  `gorm:"column:sembako_id;primaryKey" json:"sembako_id"`
	DisbakoID  string  `gorm:"column:disbako_id;primaryKey" json:"disbako_id"`

	Item         float64 `gorm:"column:item;type:decimal(20,4)" json:"item"`
	Poin         float64 `gorm:"column:poin;type:decimal(20,4)" json:"poin"`
	SubtotalPoin float64 `gorm:"column:subtotal_poin;type:decimal(20,4)" json:"subtotal_poin"`

	StokBsuSebelum float64 `gorm:"column:stok_bsu_sebelum;type:decimal(20,4)" json:"stok_bsu_sebelum"`
	StokBsuSesudah float64 `gorm:"column:stok_bsu_sesudah;type:decimal(20,4)" json:"stok_bsu_sesudah"`
	StokBsiSebelum float64 `gorm:"column:stok_bsi_sebelum;type:decimal(20,4)" json:"stok_bsi_sebelum"`
	StokBsiSesudah float64 `gorm:"column:stok_bsi_sesudah;type:decimal(20,4)" json:"stok_bsi_sesudah"`
}

func (DetailDistribusiSembako) TableName() string {
	return "detail_distribusi_sembako"
}