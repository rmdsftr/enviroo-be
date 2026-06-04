package models

import "time"

type RiwayatArusSaldo struct {
	RiwayatSaldoID string    `gorm:"column:riwayat_saldo_id;type:varchar(100);primaryKey" json:"riwayat_saldo_id"`
	RekeningID     *string   `gorm:"column:rekening_id;type:varchar(100)" json:"rekening_id"`
	NominalSebelum float64   `gorm:"column:nominal_sebelum;type:decimal(20,4)" json:"nominal_sebelum"`
	NominalSesudah float64   `gorm:"column:nominal_sesudah;type:decimal(20,4)" json:"nominal_sesudah"`
	CreatedAt      time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy      *string   `gorm:"column:created_by;type:varchar(100)" json:"created_by"`
	Keterangan		string	 `gorm:"column:keterangan;type:varchar(255)" json:"keterangan"`

	SaldoRekening *SaldoRekening `gorm:"foreignKey:RekeningID;references:RekeningID"`
}

func (RiwayatArusSaldo) TableName() string {
	return "riwayat_arus_saldo"
}
