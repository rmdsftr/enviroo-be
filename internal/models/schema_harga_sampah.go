package models

type LevelUser string

const (
	LevelNasabah   LevelUser = "nasabah"
	LevelEksternal LevelUser = "eksternal"
)

type SchemaHargaSampah struct {
	SchemaID  string    `gorm:"column:schema_id;type:varchar(14);primaryKey"`
	SampahID  string    `gorm:"column:sampah_id;type:varchar(13);not null;uniqueIndex:idx_schema_harga_sampah_level"`
	LevelUser LevelUser `gorm:"column:level_user;type:level_user_enum;not null;uniqueIndex:idx_schema_harga_sampah_level"`
	Harga     float64   `gorm:"column:harga;type:decimal(20,4);not null"`
	SatuanReward SatuanRewardEnum `gorm:"column:satuan_reward;type:satuan_reward_enum;not null"`

	KatalogSampah KatalogSampah `gorm:"foreignKey:SampahID;references:SampahID;constraint:OnDelete:CASCADE"`
}

func (SchemaHargaSampah) TableName() string {
	return "schema_harga_sampah"
}
