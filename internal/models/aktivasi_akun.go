package models

import "time"

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

type AktivasiAkun struct {
	AktivasiID string       `gorm:"column:aktivasi_id;primaryKey;size:29"`
	UserID     string       `gorm:"column:user_id;size:25;not null;index"`
	AsRole     RoleUserEnum `gorm:"column:as_role;type:role_user_enum;not null"`

	NasabahID *string `gorm:"column:nasabah_id;type:varchar(15)"`
	AdminID   *string `gorm:"column:admin_id;type:varchar(13)"`

	Token string `gorm:"column:token;size:255;not null;index"`

	IsUsed bool       `gorm:"column:is_used;default:false"`
	UsedAt *time.Time `gorm:"column:used_at"`

	ExpiredAt time.Time `gorm:"column:expired_at;not null"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`

	GeneratedBy string     `gorm:"column:generated_by;type:varchar(25);not null"`
	Tujuan      TujuanEnum `gorm:"column:tujuan;type:tujuan_enum;not null"`

	User    *User    `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
	Nasabah *Nasabah `gorm:"foreignKey:NasabahID;references:NasabahID;constraint:OnDelete:CASCADE"`
	Admin   *Admin   `gorm:"foreignKey:AdminID;references:AdminID;constraint:OnDelete:CASCADE"`
}

func (AktivasiAkun) TableName() string {
	return "aktivasi_akun"
}
