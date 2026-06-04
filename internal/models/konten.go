package models

import "time"

type Konten struct {
	KontenID   string `gorm:"column:konten_id;primaryKey;size:100"`
	Judul      string `gorm:"column:judul;size:255"`
	Deskripsi  string `gorm:"column:deskripsi;type:text"` // Untuk ringkasan/lead berita
	Body       string `gorm:"column:body;type:text"`      // Tempat JSON (struktur blok-blok konten)
	Thumbnail  string `gorm:"column:thumbnail;size:255"`  // URL foto cover utama
	IsUploaded bool   `gorm:"column:is_uploaded"`
	BankID     *string `gorm:"column:bank_id;size:100"`
	AdminID    string `gorm:"column:admin_id;size:100"` // Pembuat konten

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relasi
	Media []Media `gorm:"foreignKey:KontenID"`
}

func (Konten) TableName() string {
	return "konten"
}
