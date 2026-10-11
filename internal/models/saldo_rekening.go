package models

import "time"

type SaldoRekening struct {
	RekeningID         string      `gorm:"column:rekening_id;type:varchar(16);primaryKey" json:"rekening_id"`
	NasabahID          *string     `gorm:"column:nasabah_id;type:varchar(15);uniqueIndex:idx_saldo_nasabah_reward" json:"nasabah_id"`
	BankID             *string     `gorm:"column:bank_id;type:varchar(9);uniqueIndex:idx_saldo_bank_reward" json:"bank_id"`
	RewardID           *int        `gorm:"column:reward_id;not null;uniqueIndex:idx_saldo_nasabah_reward;uniqueIndex:idx_saldo_bank_reward" json:"reward_id"`
	Entitas            EntitasEnum `gorm:"column:entitas;type:entitas_enum;not null" json:"entitas"`
	NominalSaldo       float64     `gorm:"column:nominal_saldo;type:decimal(20,4);not null" json:"nominal_saldo"`
	SatuanNominalSaldo SatuanRewardEnum  `gorm:"column:satuan_nominal_saldo;type:satuan_reward_enum;not null" json:"satuan_nominal_saldo"`
	LastUpdatedAt      time.Time   `gorm:"column:last_updated_at;default:CURRENT_TIMESTAMP;not null" json:"last_updated_at"`
	LastUpdatedBy      *string     `gorm:"column:last_updated_by;type:varchar(25)" json:"last_updated_by"`

	Nasabah *Nasabah    `gorm:"foreignKey:NasabahID;references:NasabahID;constraint:OnDelete:RESTRICT"`
	Bank    *BankSampah `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:RESTRICT"`
	Reward  *Reward     `gorm:"foreignKey:RewardID;references:RewardID;constraint:OnDelete:RESTRICT"`
}

func (SaldoRekening) TableName() string {
	return "saldo_rekening"
}
