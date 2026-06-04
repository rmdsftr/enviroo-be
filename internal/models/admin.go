package models

import "time"

type RoleAdmin string

const (
	SuperAdmin RoleAdmin = "superadmin"
	AdminBSI   RoleAdmin = "admin_bsi"
	AdminBSM   RoleAdmin = "admin_bsm"
	AdminBSU   RoleAdmin = "admin_bsu"
	PetugasBSI RoleAdmin = "petugas_bsi"
	PetugasBSM RoleAdmin = "petugas_bsm"
	PetugasBSU RoleAdmin = "petugas_bsu"
	RoleNasabah RoleAdmin = "nasabah"
)

type Admin struct {
	AdminID     string     `gorm:"column:admin_id;type:varchar(100);primaryKey"`
	BankID      *string    `gorm:"column:bank_id;type:varchar(100);uniqueIndex:idx_admin_bank_user"`
	UserID      string     `gorm:"column:user_id;type:varchar(100);uniqueIndex:idx_admin_bank_user"`
	JoinedAt    time.Time  `gorm:"column:joined_at;autoCreateTime"`
	Role        RoleAdmin  `gorm:"column:role;type:role_admin_enum"`
	StatusAdmin StatusAkun `gorm:"column:status_admin;type:status_akun_enum"`

	Bank BankSampah `gorm:"foreignKey:BankID;references:BankID;constraint:OnDelete:CASCADE"`
	User User       `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

func (Admin) TableName() string {
	return "admin"
}
