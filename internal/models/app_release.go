package models

import "time"

type AppRelease struct {
	ReleaseID   int       `gorm:"column:release_id;primaryKey;autoIncrement"`
	VersionCode int       `gorm:"column:version_code;not null;unique"`
	VersionName string    `gorm:"column:version_name;size:20;not null"`
	ApkKey      string    `gorm:"column:apk_key;size:255;not null"`
	FileSize    int64     `gorm:"column:file_size;not null"`
	Changelog   string    `gorm:"column:changelog;type:text;not null"`
	IsMandatory bool      `gorm:"column:is_mandatory;not null"`
	CreatedBy   *string   `gorm:"column:created_by;size:25"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (AppRelease) TableName() string {
	return "app_release"
}
