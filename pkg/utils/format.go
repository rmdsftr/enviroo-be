package utils

import (
	"enviroo-be/internal/models"
	"fmt"
	"strings"
)

func FormatFloat(f float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", f), "0"), ".")
}

func FormatBagiHasilNilai(nilai float64, satuan models.SatuanRewardEnum) string {
	switch satuan {
	case models.SatuanRewardEnumPoin:
		return fmt.Sprintf("%.0f poin", nilai)
	default:
		return fmt.Sprintf("Rp%s", formatAngka(int64(nilai)))
	}
}

func formatAngka(n int64) string {
	s := fmt.Sprintf("%d", n)
	ln := len(s)
	if ln <= 3 {
		return s
	}
	result := ""
	for i, c := range s {
		if i > 0 && (ln-i)%3 == 0 {
			result += "."
		}
		result += string(c)
	}
	return result
}
