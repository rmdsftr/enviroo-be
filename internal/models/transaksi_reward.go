package models

import (
	"time"
)

type JenisTransaksiReward string

const (
	TransferBSIToBSU JenisTransaksiReward = "transfer_bsi_to_bsu"
	PenarikanNasabah JenisTransaksiReward = "penarikan_nasabah"
)

type StatusTransaksiReward string

const (
	StatusTransaksiWaiting  StatusTransaksiReward = "waiting"
	StatusTransaksiApproved StatusTransaksiReward = "approved"
	StatusTransaksiRejected StatusTransaksiReward = "rejected"
	StatusTransaksiCanceled StatusTransaksiReward = "canceled"
	StatusTransaksiSuccess  StatusTransaksiReward = "success"
	StatusTransaksiFailed   StatusTransaksiReward = "failed"
)

type TransaksiReward struct {
	TransaksiID string `gorm:"column:transaksi_id;primaryKey;size:100"`

	JenisTransaksi JenisTransaksiReward `gorm:"column:jenis_transaksi;type:jenis_transaksi_reward_enum;not null"`

	// Nullable untuk jenis transaksi tertentu
	NasabahID *string `gorm:"column:nasabah_id;size:100"`
	Nasabah   *Nasabah `gorm:"foreignKey:NasabahID;references:NasabahID"`

	BankAsalID *string     `gorm:"column:bank_asal_id;size:100"`
	BankAsal   *BankSampah `gorm:"foreignKey:BankAsalID;references:BankID"`

	BankTujuanID *string     `gorm:"column:bank_tujuan_id;size:100"`
	BankTujuan   *BankSampah `gorm:"foreignKey:BankTujuanID;references:BankID"`

	RewardID *int   `gorm:"column:reward_id"`
	Reward   *Reward `gorm:"foreignKey:RewardID;references:RewardID"`

	Poin    float64 `gorm:"column:poin;type:decimal(20,4);not null"`
	Nominal float64 `gorm:"column:nominal;type:decimal(20,4);not null"`

	StatusTransaksi StatusTransaksiReward `gorm:"column:status_transaksi;type:status_transaksi_enum;default:'waiting'"`

	Catatan *string `gorm:"column:catatan;size:255"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`

	UpdatedBy *string `gorm:"column:updated_by;size:100"`

	Details []TransaksiRewardDetail `gorm:"foreignKey:TransaksiID"`
}

func (TransaksiReward) TableName() string {
	return "transaksi_reward"
}