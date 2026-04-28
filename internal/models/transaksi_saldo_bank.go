package models

import "time"

type JenisTransaksiBank string

const (
	TransaksiPengangkutan JenisTransaksiBank = "pengangkutan"
	TransaksiPenjualan    JenisTransaksiBank = "penjualan"
	TransaksiKonversi     JenisTransaksiBank = "konversi"
)

type TransaksiSaldoBank struct {
	TransaksiBankID string `gorm:"column:transaksi_bank_id;primaryKey;size:100"`

	SaldoBankID string `gorm:"column:saldo_bank_id;size:100"`

	JenisTransaksi JenisTransaksiBank `gorm:"column:jenis_transaksi;type:jenis_transaksi_bank_enum"`

	Jumlah float64 `gorm:"column:jumlah;type:decimal(20,4)"`

	SaldoSebelum float64 `gorm:"column:saldo_sebelum;type:decimal(20,4)"`

	SaldoSesudah float64 `gorm:"column:saldo_sesudah;type:decimal(20,4)"`

	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (TransaksiSaldoBank) TableName() string {
	return "transaksi_saldo_bank"
}