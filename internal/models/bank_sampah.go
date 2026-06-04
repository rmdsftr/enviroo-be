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
	Longitude         float64           `gorm:"column:longitude;type:decimal(9,6)"`
	Latitude          float64           `gorm:"column:latitude;type:decimal(9,6)"`
	IDKecamatan       *int              `gorm:"column:id_kecamatan"`
	IDKelurahan       *int              `gorm:"column:id_kelurahan"`
	Deskripsi         string            `gorm:"column:deskripsi;type:text"`
	IsActive          bool              `gorm:"column:is_active"`
	CreatedAt         time.Time         `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time         `gorm:"column:updated_at;autoUpdateTime"`

	ParentBank *BankSampah  `gorm:"foreignKey:ParentBankID;constraint:OnDelete:SET NULL"`
	Children   []BankSampah `gorm:"foreignKey:ParentBankID"`

	Kecamatan *Kecamatan `gorm:"foreignKey:IDKecamatan;references:IDKecamatan"`
	Kelurahan *Kelurahan `gorm:"foreignKey:IDKelurahan;references:IDKelurahan"`

	Nasabahs []Nasabah `gorm:"foreignKey:BankID"`
	Admins   []Admin   `gorm:"foreignKey:BankID"`
	Jadwals  []Jadwal  `gorm:"foreignKey:BankID"`
}

func (BankSampah) TableName() string {
	return "bank_sampah"
}

// BeforeCreate membuat BankID otomatis menggunakan format smart ID:
// [kode_bank 1 digit] + [kode_kecamatan 2 digit] + [kode_kelurahan 3 digit] + [sequence 3 digit]
// IDKecamatan dan IDKelurahan harus sudah diisi sebelum Create dipanggil.
func (b *BankSampah) BeforeCreate(tx *gorm.DB) (err error) {
	if b.BankID != "" {
		return nil
	}

	if b.IDKecamatan == nil || b.IDKelurahan == nil {
		return fmt.Errorf("IDKecamatan dan IDKelurahan harus diisi untuk generate BankID")
	}

	// Import utils di sini tidak bisa (circular import), jadi logika generate dipanggil lewat callback
	// yang diset dari controller menggunakan field BankID secara eksplisit sebelum Create.
	// BeforeCreate ini hanya sebagai safety net.
	return fmt.Errorf("BankID belum di-generate: panggil utils.GenerateBankID sebelum Create")
}