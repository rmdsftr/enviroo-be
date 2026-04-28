package models

import "time"

type KatalogSembako struct {
    SembakoID   string    `gorm:"column:sembako_id;primaryKey;size:100"`
    BankID      string    `gorm:"column:bank_id;size:100"`
    NamaSembako string    `gorm:"column:nama_sembako;size:255"`
    PhotoURL    string    `gorm:"column:photo_url;size:255"`

    CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
    UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`

    // Relasi ke history
    HistoryPoin []HistoryPoinSembako `gorm:"foreignKey:SembakoID"`
}

func (KatalogSembako) TableName() string {
	return "katalog_sembako"
}