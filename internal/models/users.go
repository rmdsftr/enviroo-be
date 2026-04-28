package models

import "time"

type StatusAkun string

const (
	Aktif    StatusAkun = "aktif"
	Nonaktif StatusAkun = "nonaktif"
	Pending  StatusAkun = "pending"
)

type User struct {
	UserID     string    `gorm:"column:user_id;type:varchar(100);primaryKey"`
	Nama       string    `gorm:"column:nama;type:varchar(255)"`
	Email      string    `gorm:"column:email;type:varchar(255);unique"`
	NoWhatsapp string    `gorm:"column:no_whatsapp;type:varchar(25)"`
	Password   string    `gorm:"column:password;type:varchar(255)"`
	PhotoURL   string    `gorm:"column:photo_url;type:varchar(255)"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`

	Nasabahs []Nasabah `gorm:"foreignKey:UserID"`
	Admins   []Admin   `gorm:"foreignKey:UserID"`
}

func (User) TableName() string {
	return "users"
}
