package models

type SettingSisaBagiHasil struct {
	SettingID      string  `json:"setting_id" gorm:"column:setting_id;type:varchar(12);primaryKey"`
	BankID        string    `gorm:"column:bank_id;type:varchar(9);not null;uniqueIndex"`
	PorsiBSU       float64 `json:"porsi_bsu" gorm:"column:porsi_bsu;type:decimal(20,4);not null"`
	PorsiBSI       float64 `json:"porsi_bsi" gorm:"column:porsi_bsi;type:decimal(20,4);not null"`
	PorsiTransport float64 `json:"porsi_transport" gorm:"column:porsi_transport;type:decimal(20,4);not null"`

	Bank   BankSampah `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:CASCADE"`
}

func (SettingSisaBagiHasil) TableName() string {
	return "setting_sisa_bagi_hasil"
}