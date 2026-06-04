package models

import "time"

type DistribusiSembakoBSU struct {
	DistribusiID string `gorm:"column:distribusi_id;type:varchar(100);primaryKey" json:"distribusi_id"`

	BSUID *string `gorm:"column:bsu_id;type:varchar(100)" json:"bsu_id"`

	BSIID *string `gorm:"column:bsi_id;type:varchar(100)" json:"bsi_id"`

	SembakoID *string `gorm:"column:sembako_id;type:varchar(100)" json:"sembako_id"`

	AdminBSIID *string `gorm:"column:admin_bsi_id;type:varchar(100)" json:"admin_bsi_id"`

	AdminBSUID *string `gorm:"column:admin_bsu_id;type:varchar(100)" json:"admin_bsu_id"`

	StokTerdistribusi float64 `gorm:"column:stok_terdistribusi;type:decimal(20,4)" json:"stok_terdistribusi"`

	StokBSUSebelum float64 `gorm:"column:stok_bsu_sebelum;type:decimal(20,4)" json:"stok_bsu_sebelum"`

	StokBSUSesudah float64 `gorm:"column:stok_bsu_sesudah;type:decimal(20,4)" json:"stok_bsu_sesudah"`

	StokBSISebelum float64 `gorm:"column:stok_bsi_sebelum;type:decimal(20,4)" json:"stok_bsi_sebelum"`

	StokBSISesudah float64 `gorm:"column:stok_bsi_sesudah;type:decimal(20,4)" json:"stok_bsi_sesudah"`

	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`

	BSU *BankSampah `gorm:"foreignKey:BSUID;references:BankID"`

	BSI *BankSampah `gorm:"foreignKey:BSIID;references:BankID"`

	Sembako *KatalogSembako `gorm:"foreignKey:SembakoID;references:SembakoID"`

	AdminBSI *Admin `gorm:"foreignKey:AdminBSIID;references:AdminID"`

	AdminBSU *Admin `gorm:"foreignKey:AdminBSUID;references:AdminID"`
}

func (DistribusiSembakoBSU) TableName() string {
	return "distribusi_sembako_bsu"
}