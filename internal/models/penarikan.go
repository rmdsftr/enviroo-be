package models

import "time"

type StatusPenarikanEnum string

const (
	StatusPenarikanPending        StatusPenarikanEnum = "pending"
	StatusPenarikanCanceled       StatusPenarikanEnum = "canceled"
	StatusPenarikanRequestExpired StatusPenarikanEnum = "request_expired"
	StatusPenarikanRejected       StatusPenarikanEnum = "rejected"
	StatusPenarikanApproved       StatusPenarikanEnum = "approved"
	StatusPenarikanPickupExpired  StatusPenarikanEnum = "pickup_expired"
	StatusPenarikanCompleted      StatusPenarikanEnum = "completed"
)

type Penarikan struct {
	PenarikanID string `gorm:"column:penarikan_id;type:varchar(24);primaryKey" json:"penarikan_id"`

	NasabahID *string `gorm:"column:nasabah_id;type:varchar(15);not null;index" json:"nasabah_id"`

	BankID *string `gorm:"column:bank_id;type:varchar(9);not null;index" json:"bank_id"`

	RewardID *int `gorm:"column:reward_id;not null" json:"reward_id"`

	NominalPenarikan float64 `gorm:"column:nominal_penarikan;type:decimal(20,4);not null" json:"nominal_penarikan"`

	SatuanPenarikan SatuanRewardEnum `gorm:"column:satuan_penarikan;type:satuan_reward_enum;not null" json:"satuan_penarikan"`

	StatusPenarikan StatusPenarikanEnum `gorm:"column:status_penarikan;type:status_penarikan_enum;not null" json:"status_penarikan"`

	DeadlineKonfirmasi *time.Time `gorm:"column:deadline_konfirmasi" json:"deadline_konfirmasi"`

	DeadlineJemput *time.Time `gorm:"column:deadline_jemput" json:"deadline_jemput"`

	BuktiFoto *string `gorm:"column:bukti_foto;type:varchar(255)" json:"bukti_foto"`

	Nasabah                *Nasabah                 `gorm:"foreignKey:NasabahID;references:NasabahID;constraint:OnDelete:RESTRICT"`
	Bank                   *BankSampah              `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:RESTRICT"`
	Reward                 *Reward                  `gorm:"foreignKey:RewardID;references:RewardID;constraint:OnDelete:RESTRICT"`
	DetailPenarikanBarang  []DetailPenarikanBarang  `gorm:"foreignKey:PenarikanID;references:PenarikanID"`
	RiwayatPenarikan       []RiwayatPenarikan       `gorm:"foreignKey:PenarikanID;references:PenarikanID"`
}

func (Penarikan) TableName() string {
	return "penarikan"
}
