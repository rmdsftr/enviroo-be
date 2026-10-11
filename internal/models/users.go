package models

import "time"

type StatusAkun string

const (
	Aktif    StatusAkun = "aktif"
	Nonaktif StatusAkun = "nonaktif"
	Pending  StatusAkun = "pending"
)

type User struct {
	UserID     string    `gorm:"column:user_id;type:varchar(25);primaryKey"`
	Nama       string    `gorm:"column:nama;type:varchar(255);not null"`
	Email      string    `gorm:"column:email;type:varchar(255);unique;not null"`
	NoWhatsapp string    `gorm:"column:no_whatsapp;type:varchar(25)"`
	Password   string    `gorm:"column:password;type:varchar(255);not null"`
	PhotoURL   string    `gorm:"column:photo_url;type:varchar(255)"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
	FCMToken   string    `gorm:"column:fcm_token;type:text"`

	DeviceSessionID *string `gorm:"column:device_session_id;type:varchar(36)"`
	WebSessionID    *string `gorm:"column:web_session_id;type:varchar(36)"`

	Nasabahs []Nasabah `gorm:"foreignKey:UserID"`
	Admins   []Admin   `gorm:"foreignKey:UserID"`
}

func (User) TableName() string {
	return "users"
}

const (
	PlatformWeb    = "web"
	PlatformMobile = "mobile"
)

func SessionColumn(platform string) string {
	if platform == PlatformWeb {
		return "web_session_id"
	}
	return "device_session_id"
}

func (u User) SessionIDFor(platform string) *string {
	if platform == PlatformWeb {
		return u.WebSessionID
	}
	return u.DeviceSessionID
}
