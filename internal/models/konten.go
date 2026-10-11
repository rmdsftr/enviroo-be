package models

import "time"

type Konten struct {
	KontenID   string `gorm:"column:konten_id;primaryKey;size:21"`
	Judul      string `gorm:"column:judul;size:255;not null"`
	Deskripsi  string `gorm:"column:deskripsi;type:text;not null"`
	Body       string `gorm:"column:body;type:text;not null"`
	Thumbnail  string `gorm:"column:thumbnail;size:255;not null"`
	IsUploaded bool   `gorm:"column:is_uploaded;not null"`
	BankID     *string `gorm:"column:bank_id;size:9"`
	AdminID    string `gorm:"column:admin_id;size:13;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;not null"`

	Bank  *BankSampah `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:CASCADE"`
	Admin Admin       `gorm:"foreignKey:AdminID;references:AdminID;constraint:OnDelete:RESTRICT"`
	Media []Media `gorm:"foreignKey:KontenID"`
}

func (Konten) TableName() string {
	return "konten"
}
