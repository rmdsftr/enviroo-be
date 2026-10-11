package models

type Media struct {
	MediaID  string `gorm:"column:media_id;primaryKey;size:36"`
	MediaURL string `gorm:"column:media_url;size:255;not null"`
	FileName string `gorm:"column:file_name;size:255;not null"`
	FileType string `gorm:"column:file_type;size:50;not null"`
	FileSize int64  `gorm:"column:file_size;not null"`
	AdminID  string `gorm:"column:admin_id;size:13"`
	KontenID string `gorm:"column:konten_id;size:21"`

	Konten Konten `gorm:"foreignKey:KontenID;references:KontenID;constraint:OnDelete:CASCADE"`
}

func (Media) TableName() string {
	return "media"
}