package models

import (
	"time"

	"github.com/google/uuid"
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
	JadwalID     uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"jadwal_id"`
	BankID       string     `gorm:"type:varchar(100);not null" json:"bank_id"`
	Hari         HariEnum   `gorm:"type:hari_enum;not null" json:"hari"`
	MingguKe     int        `gorm:"type:smallint;check:minggu_ke BETWEEN 1 AND 5" json:"minggu_ke"`
	JamMulai     string     `gorm:"type:time" json:"jam_mulai"`
	JamSelesai   string     `gorm:"type:time" json:"jam_selesai"`
	JenisJadwal  JadwalEnum `gorm:"type:jadwal_enum;not null" json:"jenis_jadwal"`
	TargetBankID string     `gorm:"type:varchar(100);index" json:"target_bank_id"`
	IsActive     *bool       `gorm:"default:true" json:"is_active"`
	IsRutin      *bool       `gorm:"default:true" json:"is_rutin"`
	Tanggal      time.Time  `gorm:"type:date" json:"tanggal"`
	NamaJadwalSpesial string `gorm:"type:varchar(255);" json:"nama_jadwal_spesial"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	CreatedBy    string     `gorm:"type:varchar(100);not null" json:"created_by"`
}

func (Jadwal) TableName() string {
	return "jadwal"
}
