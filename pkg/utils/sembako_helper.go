package utils

import (
	"enviroo-be/internal/models"
	"errors"
	"regexp"

	"gorm.io/gorm"
)

// FindOrCreateMasterSembako mencari sembako di tabel master berdasarkan nama (regex case-insensitive).
// Kalau tidak ditemukan, otomatis buat record baru.
// Catatan: sembako tidak pakai satuan karena dihitung per item.
// Contoh: "beras 5 kg" dianggap 1 item.
func FindOrCreateMasterSembako(db *gorm.DB, namaSembako string) (*models.Sembako, error) {
	if namaSembako == "" {
		return nil, errors.New("nama sembako tidak boleh kosong")
	}

	var existing models.Sembako

	// Gunakan regex exact match, case-insensitive (PostgreSQL: ~*)
	// regexp.QuoteMeta mencegah karakter spesial di input dianggap regex operator
	pattern := "^" + regexp.QuoteMeta(namaSembako) + "$"
	err := db.Where("nama_barang ~* ?", pattern).First(&existing).Error

	if err == nil {
		// Sudah ada → return yang existing
		return &existing, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Error lain (koneksi DB, dll)
		return nil, err
	}

	// Belum ada → buat baru
	newSembako := models.Sembako{
		NamaBarang: namaSembako,
	}

	if err := db.Create(&newSembako).Error; err != nil {
		return nil, err
	}

	return &newSembako, nil
}