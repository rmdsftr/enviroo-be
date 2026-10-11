package models

import "time"
 
type Notifikasi struct {
	NotifikasiID string `gorm:"column:notifikasi_id;type:varchar(36);primaryKey" json:"notifikasi_id"`
	UserID       string `gorm:"column:user_id;type:varchar(25);not null;index"    json:"user_id"`

	RoleTarget string `gorm:"column:role_target;type:varchar(10);not null;default:'admin'" json:"role_target"`

	Judul string `gorm:"column:judul;type:varchar(255);not null" json:"judul"`
	Pesan string `gorm:"column:pesan;type:text;not null"         json:"pesan"`

	RefID   *string `gorm:"column:ref_id;type:varchar(50)"  json:"ref_id"`
	RefType *string `gorm:"column:ref_type;type:varchar(25)" json:"ref_type"`

	IsRead bool `gorm:"column:is_read;default:false;not null" json:"is_read"`

	CreatedAt time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP;not null" json:"created_at"`

	User User `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE" json:"-"`
}

func (Notifikasi) TableName() string {
	return "notifikasi"
}
