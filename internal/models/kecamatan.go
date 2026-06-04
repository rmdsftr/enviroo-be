package models

import "time"

type Kecamatan struct {
	IDKecamatan int       `gorm:"column:id_kecamatan;primaryKey;autoIncrement" json:"id_kecamatan"`
	Kecamatan   string    `gorm:"column:kecamatan;type:varchar(255);not null" json:"kecamatan"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Kecamatan) TableName() string {
	return "kecamatan"
}