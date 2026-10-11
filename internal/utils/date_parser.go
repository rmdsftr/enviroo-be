package utils

import (
	"fmt"
	"strings"
	"time"

	"enviroo-be/internal/models"
)

func ParseFlexibleDate(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

var namaBulanIndo = [...]string{
	"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

func FormatTanggalIndo(t time.Time) string {
	return fmt.Sprintf("%d %s %d pukul %02d.%02d",
		t.Day(), namaBulanIndo[t.Month()-1], t.Year(), t.Hour(), t.Minute())
}

func FormatTanggalSajaIndo(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), namaBulanIndo[t.Month()-1], t.Year())
}

func FormatJamIndo(jam string) string {
	parts := strings.SplitN(jam, ":", 3)
	if len(parts) < 2 {
		return jam
	}
	return parts[0] + "." + parts[1]
}

func WeekOfMonth(date time.Time) int {
	firstOfMonth := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
	firstWeekday := int(firstOfMonth.Weekday())
	adjustedDay := date.Day() + firstWeekday - 1
	return (adjustedDay / 7) + 1
}

func HariOf(t time.Time) models.HariEnum {
	return []models.HariEnum{
		models.Minggu, models.Senin, models.Selasa, models.Rabu,
		models.Kamis, models.Jumat, models.Sabtu,
	}[t.Weekday()]
}
