package models

import "time"

// ENUM
type RoleUserEnum string

const (
	RoleUserNasabah RoleUserEnum = "nasabah"
	RoleUserAdmin   RoleUserEnum = "admin"
	RoleUserGlobal  RoleUserEnum = "global"
)

type TujuanEnum string

const (
	TujuanAktivasi      TujuanEnum = "aktivasi"
	TujuanResetPassword TujuanEnum = "reset_password"
)

// MODEL
type AktivasiAkun struct {
	AktivasiID string        `gorm:"column:aktivasi_id;primaryKey;size:100"`
	UserID     string        `gorm:"column:user_id;size:100;not null;index"`
	AsRole     RoleUserEnum  `gorm:"column:as_role;type:role_user_enum"`

	Token      string        `gorm:"column:token;size:255;not null;index"`

	IsUsed     bool          `gorm:"column:is_used;default:false"`
	UsedAt     *time.Time    `gorm:"column:used_at"`

	ExpiredAt  time.Time     `gorm:"column:expired_at;not null"`
	CreatedAt  time.Time     `gorm:"column:created_at;autoCreateTime"`

	GeneratedBy string       `gorm:"column:generated_by;size:100"`
	Tujuan      TujuanEnum   `gorm:"column:tujuan;type:tujuan_enum"`

	// Relasi (optional)
	User        *User        `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

func (AktivasiAkun) TableName() string {
	return "aktivasi_akun"
}