package models

import (
	"time"
)

type HariEnum string
type JadwalEnum string

const (
	Senin  HariEnum = "senin"
	Selasa HariEnum = "selasa"
	Rabu   HariEnum = "rabu"
	Kamis  HariEnum = "kamis"
	Jumat  HariEnum = "jumat"
	Sabtu  HariEnum = "sabtu"
	Minggu HariEnum = "minggu"
)

const (
	JadwalPenimbangan  JadwalEnum = "penimbangan"
	JadwalPengangkutan JadwalEnum = "pengangkutan"
)

type Jadwal struct {
	JadwalID string `gorm:"type:varchar(15);primaryKey" json:"jadwal_id"`
	BankID   string `gorm:"type:varchar(9);not null;index" json:"bank_id"`

	Hari     *HariEnum  `gorm:"type:hari_enum" json:"hari"`
	MingguKe *int       `gorm:"type:smallint;check:minggu_ke BETWEEN 1 AND 5" json:"minggu_ke"`
	Tanggal  *time.Time `gorm:"type:date" json:"tanggal"`

	JamMulai    string     `gorm:"type:time;not null" json:"jam_mulai"`
	JamSelesai  string     `gorm:"type:time;not null" json:"jam_selesai"`
	JenisJadwal JadwalEnum `gorm:"type:jadwal_enum;not null" json:"jenis_jadwal"`

	TargetBankID *string `gorm:"type:varchar(9);index" json:"target_bank_id"`

	IsActive          *bool     `gorm:"default:true;not null" json:"is_active"`
	IsRutin           *bool     `gorm:"default:true;not null" json:"is_rutin"`
	NamaJadwalSpesial string    `gorm:"type:varchar(50)" json:"nama_jadwal_spesial"`
	CreatedAt         time.Time `gorm:"autoCreateTime;not null" json:"created_at"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime;not null" json:"updated_at"`
	CreatedBy         string    `gorm:"type:varchar(25);not null" json:"created_by"`

	Bank       BankSampah  `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:RESTRICT"`
	TargetBank *BankSampah `gorm:"foreignKey:TargetBankID;references:BankID;constraint:OnDelete:RESTRICT"`
}

func (Jadwal) TableName() string {
	return "jadwal"
}
