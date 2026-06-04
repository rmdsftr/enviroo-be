package models

import "time"

type Kelurahan struct {
	IDKelurahan int       `gorm:"column:id_kelurahan;primaryKey;autoIncrement" json:"id_kelurahan"`
	IDKecamatan int       `gorm:"column:id_kecamatan;not null" json:"id_kecamatan"`
	Kelurahan   string    `gorm:"column:kelurahan;type:varchar(255);not null" json:"kelurahan"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`

	Kecamatan *Kecamatan `gorm:"foreignKey:IDKecamatan;references:IDKecamatan" json:"kecamatan,omitempty"`
}

func (Kelurahan) TableName() string {
	return "kelurahan"
}