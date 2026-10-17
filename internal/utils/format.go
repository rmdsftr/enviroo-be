package utils

import (
	"enviroo-be/internal/models"
	"errors"
	"fmt"
	"math"
	"strings"
)

func FormatFloat(f float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", f), "0"), ".")
}

func FormatBagiHasilNilai(nilai float64, satuan models.SatuanRewardEnum) (string, error) {
	if nilai < 0 {
		return "", errors.New("nilai bagi hasil tidak boleh negatif")
	}
	switch satuan {
	case models.SatuanRewardEnumPoin:
		return fmt.Sprintf("%.0f poin", math.Round(nilai)), nil
	default:
		return fmt.Sprintf("Rp%s", formatAngka(int64(math.Round(nilai)))), nil
	}
}

func FormatReward(satuan string) string {
	if strings.EqualFold(satuan, "poin") {
		return `#,##0.00" poin"`
	}
	return `"Rp"#,##0.00`
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
