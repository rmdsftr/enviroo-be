package models

import "time"

type JenisTransaksi string

const (
	Setoran   JenisTransaksi = "setoran"
	Penarikan JenisTransaksi = "penarikan"
)

type TransaksiSaldoNasabah struct {
	TransaksiID     string         `gorm:"column:transaksi_id;type:varchar(100);primaryKey"`
	SaldoID         string         `gorm:"column:saldo_id;type:varchar(100);index;not null"`

	JenisTransaksi  JenisTransaksi `gorm:"column:jenis_transaksi;type:jenis_transaksi_enum;not null"`

	Jumlah          float64        `gorm:"column:jumlah;type:decimal(20,4);default:0"`
	SaldoSebelum    float64        `gorm:"column:saldo_sebelum;type:decimal(20,4);not null"`
	SaldoSesudah    float64        `gorm:"column:saldo_sesudah;type:decimal(20,4);not null"`

	CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime"`
	CreatedBy       string         `gorm:"column:created_by;type:varchar(100)"`

	Saldo           SaldoNasabah   `gorm:"foreignKey:SaldoID;references:SaldoID"`
}

func (TransaksiSaldoNasabah) TableName() string {
	return "transaksi_saldo_nasabah"
}