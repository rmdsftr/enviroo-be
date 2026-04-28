package models

type TransaksiRewardDetail struct {
	TransaksiID string `gorm:"column:transaksi_id;primaryKey;size:100"`
	SembakoID   string `gorm:"column:sembako_id;primaryKey;size:100"`

	TransaksiReward TransaksiReward `gorm:"foreignKey:TransaksiID;references:TransaksiID"`
	Sembako         KatalogSembako  `gorm:"foreignKey:SembakoID;references:SembakoID"`

	Qty          float64 `gorm:"column:qty;type:decimal(20,4)"`
	NilaiPoin    float64 `gorm:"column:nilai_poin;type:decimal(20,4)"`
	SubtotalPoin float64 `gorm:"column:subtotal_poin;type:decimal(20,4)"`
}

func (TransaksiRewardDetail) TableName() string {
	return "transaksi_reward_detail"
}