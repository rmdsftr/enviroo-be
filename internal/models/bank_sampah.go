package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type JenisBank string

const (
	BSI JenisBank = "bsi"
	BSM JenisBank = "bsm"
	BSU JenisBank = "bsu"
)

type BankSampah struct {
	BankID            string            `gorm:"column:bank_id;type:varchar(100);primaryKey"`
	NamaBank          string            `gorm:"column:nama_bank;type:varchar(255)"`
	ParentBankID      *string           `gorm:"column:parent_bank_id;type:varchar(100)"`
	PhotoURL          string            `gorm:"column:photo_url;type:varchar(255)"`
	JenisBank         JenisBank         `gorm:"column:jenis_bank;type:jenis_bank_enum"`
	Alamat            string            `gorm:"column:alamat;type:varchar(255)"`
	Provinsi          string            `gorm:"column:provinsi;type:varchar(100)"`
	KabupatenKota     string            `gorm:"column:kabupaten_kota;type:varchar(100)"`
	Kecamatan         string            `gorm:"column:kecamatan;type:varchar(100)"`
	Longitude         float64           `gorm:"column:longitude;type:decimal(9,6)"`
	Latitude          float64           `gorm:"column:latitude;type:decimal(9,6)"`
	Deskripsi         string            `gorm:"column:deskripsi;type:text"`
	IsActive          bool              `gorm:"column:is_active"`
	CreatedAt         time.Time         `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time         `gorm:"column:updated_at;autoUpdateTime"`

	ParentBank *BankSampah  `gorm:"foreignKey:ParentBankID;constraint:OnDelete:SET NULL"`
	Children   []BankSampah `gorm:"foreignKey:ParentBankID"`

	Nasabahs []Nasabah `gorm:"foreignKey:BankID"`
	Admins   []Admin   `gorm:"foreignKey:BankID"`
}

func (BankSampah) TableName() string {
	return "bank_sampah"
}

func (b *BankSampah) BeforeCreate(tx *gorm.DB) (err error) {
	
	if b.BankID != "" {
		return nil
	}

	var nextVal int64

	if err := tx.Raw(`SELECT nextval('bank_id_seq')`).Scan(&nextVal).Error; err != nil {
		return err
	}

	b.BankID = fmt.Sprintf("%05d", nextVal)

	return nil
}