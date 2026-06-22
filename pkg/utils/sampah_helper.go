package utils

import (
	"enviroo-be/internal/models"
	"errors"
	"regexp"

	"gorm.io/gorm"
)

// FindOrCreateMasterSampah mencari sampah di tabel master berdasarkan nama (regex case-insensitive).
// Kalau tidak ditemukan, otomatis buat record baru.
func FindOrCreateMasterSampah(db *gorm.DB, namaSampah string, satuan models.SatuanEnum) (*models.Sampah, error) {
	if namaSampah == "" {
		return nil, errors.New("nama sampah tidak boleh kosong")
	}

	var existing models.Sampah

	// Gunakan regex exact match, case-insensitive (PostgreSQL: ~*)
	// regexp.QuoteMeta mencegah karakter spesial di input dianggap regex operator
	pattern := "^" + regexp.QuoteMeta(namaSampah) + "$"
	err := db.Where("nama_sampah ~* ?", pattern).First(&existing).Error

	if err == nil {
		// Sudah ada → return yang existing
		return &existing, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Error lain (koneksi DB, dll)
		return nil, err
	}

	// Belum ada → buat baru
	newSampah := models.Sampah{
		NamaSampah: namaSampah,
		Satuan:     satuan,
	}

	if err := db.Create(&newSampah).Error; err != nil {
		return nil, err
	}

	return &newSampah, nil
}