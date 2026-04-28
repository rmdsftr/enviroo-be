package models

type Media struct {
	MediaID  string `gorm:"column:media_id;primaryKey;size:100"`
	MediaURL string `gorm:"column:media_url;size:255"`
	FileName string `gorm:"column:file_name;size:255"`
	FileType string `gorm:"column:file_type;size:50"` // e.g., image/png, image/jpeg
	FileSize int64  `gorm:"column:file_size"`        // In bytes
	AdminID  string `gorm:"column:admin_id;size:100"` // Uploader
	KontenID string `gorm:"column:konten_id;size:100"` // FK to Konten (optional/independent link)

	// Relasi balik
	Konten Konten `gorm:"foreignKey:KontenID;references:KontenID"`
}

func (Media) TableName() string {
	return "media"
}