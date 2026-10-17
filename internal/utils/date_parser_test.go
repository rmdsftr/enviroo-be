package utils_test

import (
	"testing"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/utils"

	"github.com/stretchr/testify/assert"
)

func tgl(tahun int, bulan time.Month, hari, jam, menit int) time.Time {
	return time.Date(tahun, bulan, hari, jam, menit, 0, 0, time.UTC)
}

func TestParseFlexibleDate_TanggalSaja(t *testing.T) {
	hasil, err := utils.ParseFlexibleDate("2026-03-10")

	assert.NoError(t, err)
	assert.Equal(t, tgl(2026, time.March, 10, 0, 0), hasil)
}

func TestParseFlexibleDate_RFC3339(t *testing.T) {
	denganZona, err1 := utils.ParseFlexibleDate("2026-03-10T09:30:00+07:00")
	utc, err2 := utils.ParseFlexibleDate("2026-03-10T02:30:00Z")

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.True(t, denganZona.Equal(utc), "09:30 WIB = 02:30 UTC")
	assert.True(t, utc.Equal(tgl(2026, time.March, 10, 2, 30)))
}

func TestParseFlexibleDate_FormatSalah(t *testing.T) {
	for _, masukan := range []string{"", "10-03-2026", "2026/03/10", "2026-13-01", "2026-03-10 09:30"} {
		_, err := utils.ParseFlexibleDate(masukan)

		assert.Error(t, err, "%q", masukan)
	}
}

func TestFormatTanggalIndo(t *testing.T) {
	kasus := []struct {
		waktu   time.Time
		harapan string
	}{
		{tgl(2026, time.March, 5, 9, 7), "5 Maret 2026 pukul 09.07"},
		{tgl(2026, time.January, 1, 0, 0), "1 Januari 2026 pukul 00.00"},
		{tgl(2026, time.December, 31, 23, 59), "31 Desember 2026 pukul 23.59"},
	}
	for _, k := range kasus {
		assert.Equal(t, k.harapan, utils.FormatTanggalIndo(k.waktu))
	}
}

func TestFormatTanggalSajaIndo(t *testing.T) {
	namaBulan := []string{
		"Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember",
	}
	for i, nama := range namaBulan {
		bulan := time.Month(i + 1)

		assert.Equal(t, "1 "+nama+" 2026", utils.FormatTanggalSajaIndo(tgl(2026, bulan, 1, 15, 30)), bulan.String())
	}
}

func TestFormatJamIndo(t *testing.T) {
	kasus := map[string]string{
		"08:00":    "08.00",
		"08:00:00": "08.00",
		"13:45:30": "13.45",
		"":         "",
		"0800":     "0800",
		"08":       "08",
	}
	for masukan, harapan := range kasus {
		assert.Equal(t, harapan, utils.FormatJamIndo(masukan), "%q", masukan)
	}
}

func TestWeekOfMonth(t *testing.T) {
	kasus := []struct {
		nama    string
		tanggal time.Time
		harapan int
	}{
		// Februari 2026 diawali hari Minggu.
		{"1 Feb (Minggu)", tgl(2026, time.February, 1, 0, 0), 1},
		{"7 Feb (Sabtu)", tgl(2026, time.February, 7, 0, 0), 1},
		{"8 Feb (Minggu)", tgl(2026, time.February, 8, 0, 0), 2},
		{"28 Feb (Sabtu)", tgl(2026, time.February, 28, 0, 0), 4},
		// Agustus 2026 diawali hari Sabtu, jadi bulannya punya 6 minggu.
		{"1 Agu (Sabtu)", tgl(2026, time.August, 1, 0, 0), 1},
		{"2 Agu (Minggu)", tgl(2026, time.August, 2, 0, 0), 2},
		{"31 Agu (Senin)", tgl(2026, time.August, 31, 0, 0), 6},
	}
	for _, k := range kasus {
		assert.Equal(t, k.harapan, utils.WeekOfMonth(k.tanggal), k.nama)
	}
}

func TestHariOf(t *testing.T) {
	urutan := []models.HariEnum{
		models.Minggu, models.Senin, models.Selasa, models.Rabu,
		models.Kamis, models.Jumat, models.Sabtu,
	}
	for i, harapan := range urutan {
		tanggal := tgl(2026, time.February, i+1, 10, 0)

		assert.Equal(t, harapan, utils.HariOf(tanggal), tanggal.Format("2006-01-02 Monday"))
	}
}
