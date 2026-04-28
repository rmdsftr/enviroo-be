package models

import "time"

type SaldoBank struct {
	SaldoBankID string `gorm:"column:saldo_bank_id;primaryKey;size:100"`

	BankID string `gorm:"column:bank_id;unique;size:100"`

	TotalPoin float64 `gorm:"column:total_poin;type:decimal(20,4)"`

	LastUpdatedAt time.Time `gorm:"column:last_updated_at"`

	LastUpdatedBy string `gorm:"column:last_updated_by;size:100"`

	TransaksiSaldoBank []TransaksiSaldoBank `gorm:"foreignKey:SaldoBankID"`
}

func (SaldoBank) TableName() string {
	return "saldo_bank"
}