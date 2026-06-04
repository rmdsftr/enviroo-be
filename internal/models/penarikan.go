package models

import "time"

type StatusPenarikanEnum string

const (
	StatusPenarikanPending    StatusPenarikanEnum = "pending"
	StatusPenarikanBerhasil   StatusPenarikanEnum = "berhasil"
	StatusPenarikanKadaluarsa StatusPenarikanEnum = "kadaluarsa"
	StatusPenarikanDibatalkan StatusPenarikanEnum = "dibatalkan"
)

type Penarikan struct {
	PenarikanID string `gorm:"column:penarikan_id;type:varchar(100);primaryKey" json:"penarikan_id"`

	NasabahID *string `gorm:"column:nasabah_id;type:varchar(100)" json:"nasabah_id"`

	BankID *string `gorm:"column:bank_id;type:varchar(100)" json:"bank_id"`

	RewardID *int `gorm:"column:reward_id" json:"reward_id"`

	NominalPenarikan float64 `gorm:"column:nominal_penarikan;type:decimal(20,4)" json:"nominal_penarikan"`

	SatuanPenarikan SatuanRewardEnum `gorm:"column:satuan_penarikan;type:satuan_reward_enum" json:"satuan_penarikan"`

	StatusPenarikan StatusPenarikanEnum `gorm:"column:status_penarikan;type:status_penarikan_enum" json:"status_penarikan"`

	KadaluarsaAt *time.Time `gorm:"column:kadaluarsa_at" json:"kadaluarsa_at"`

	BuktiFoto *string `gorm:"column:bukti_foto;type:varchar(255)" json:"bukti_foto"`

	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy *string   `gorm:"column:created_by;type:varchar(100)" json:"created_by"`

	UpdatedAt time.Time `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
	UpdatedBy *string   `gorm:"column:updated_by;type:varchar(100)" json:"updated_by"`

	Nasabah                *Nasabah                 `gorm:"foreignKey:NasabahID;references:NasabahID"`
	Bank                   *BankSampah              `gorm:"foreignKey:BankID;references:BankID"`
	Reward                 *Reward                  `gorm:"foreignKey:RewardID;references:RewardID"`
	DetailPenarikanSembako []DetailPenarikanSembako `gorm:"foreignKey:PenarikanID;references:PenarikanID"`
}

func (Penarikan) TableName() string {
	return "penarikan"
}
