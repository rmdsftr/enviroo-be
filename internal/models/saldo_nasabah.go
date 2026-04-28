package models

import "time"

type SaldoNasabah struct {
    SaldoID        string    `gorm:"column:saldo_id;type:varchar(100);primaryKey"`
    NasabahID      string    `gorm:"column:nasabah_id;type:varchar(100)"`
    TotalPoin      float64   `gorm:"column:total_poin;type:decimal(20,4)"`
    LastUpdatedAt  time.Time `gorm:"column:last_updated_at;autoUpdateTime"`
    LastUpdatedBy  string    `gorm:"column:last_updated_by;type:varchar(100)"`

    Nasabah        Nasabah   `gorm:"foreignKey:NasabahID;references:NasabahID"`
}

func (SaldoNasabah) TableName() string {
	return "saldo_nasabah"
}