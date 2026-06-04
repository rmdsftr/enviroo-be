package models

type PencairanTabungan struct {
	PenerimaID string `gorm:"column:penerima_id;type:varchar(100);primaryKey" json:"penerima_id"`
	TabunganID string `gorm:"column:tabungan_id;type:varchar(100);primaryKey" json:"tabungan_id"`
	QtyDipakai float64 `json:"qty_dipakai" gorm:"column:qty_dipakai;type:decimal(20,4)"`

	PenerimaBagiHasil *PenerimaBagiHasil `gorm:"foreignKey:PenerimaID;references:PenerimaID"`
	TabunganSampah    *TabunganSampah    `gorm:"foreignKey:TabunganID;references:TabunganID"`
}

func (PencairanTabungan) TableName() string {
	return "pencairan_tabungan"
}