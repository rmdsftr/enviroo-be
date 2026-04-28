package models

import "time"

type SchemaHargaSembako struct {
	SchemaID   uint      `gorm:"column:schema_id;primaryKey;autoIncrement"`
	SembakoID  string    `gorm:"column:sembako_id;type:varchar(100);not null"`
	LevelUser  LevelUser `gorm:"column:level_user;type:level_user_enum;not null"`
	PoinHarga  float64   `gorm:"column:poin_harga;type:decimal(20,4);not null"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`

	// Relation
	Sembako KatalogSembako `gorm:"foreignKey:SembakoID;references:SembakoID"`
}

func (SchemaHargaSembako) TableName() string {
	return "schema_harga_sembako"
}