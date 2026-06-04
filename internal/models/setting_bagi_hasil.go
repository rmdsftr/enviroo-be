package models

type SettingSisaBagiHasil struct {
	SettingID      int64   `json:"setting_id" gorm:"column:setting_id;primaryKey;autoIncrement"`
	BankID        string    `gorm:"column:bank_id;type:varchar(100);not null"`
	PorsiBSU       float64 `json:"porsi_bsu" gorm:"column:porsi_bsu;type:decimal(20,4)"`
	PorsiBSI       float64 `json:"porsi_bsi" gorm:"column:porsi_bsi;type:decimal(20,4)"`
	PorsiTransport float64 `json:"porsi_transport" gorm:"column:porsi_transport;type:decimal(20,4)"`

	Bank   BankSampah `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:CASCADE"`
}

func (SettingSisaBagiHasil) TableName() string {
	return "setting_sisa_bagi_hasil"
}