package models

import "time"

type SaldoRekening struct {
	RekeningID         string      `gorm:"column:rekening_id;type:varchar(100);primaryKey" json:"rekening_id"`
	NasabahID          *string     `gorm:"column:nasabah_id;type:varchar(100)" json:"nasabah_id"`
	BankID             *string     `gorm:"column:bank_id;type:varchar(100)" json:"bank_id"`
	RewardID           *int        `gorm:"column:reward_id" json:"reward_id"`
	Entitas            EntitasEnum `gorm:"column:entitas;type:entitas_enum" json:"entitas"`
	NominalSaldo       float64     `gorm:"column:nominal_saldo;type:decimal(20,4)" json:"nominal_saldo"`
	SatuanNominalSaldo SatuanRewardEnum  `gorm:"column:satuan_nominal_saldo;type:satuan_reward_enum" json:"satuan_nominal_saldo"`
	LastUpdatedAt      time.Time   `gorm:"column:last_updated_at;default:CURRENT_TIMESTAMP" json:"last_updated_at"`
	LastUpdatedBy      *string     `gorm:"column:last_updated_by;type:varchar(100)" json:"last_updated_by"`

	Nasabah *Nasabah    `gorm:"foreignKey:NasabahID;references:NasabahID"`
	Bank    *BankSampah `gorm:"foreignKey:BankID;references:BankID"`
	Reward  *Reward     `gorm:"foreignKey:RewardID;references:RewardID"`
}

func (SaldoRekening) TableName() string {
	return "saldo_rekening"
}
