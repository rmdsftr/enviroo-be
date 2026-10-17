package utils_test

import (
	"testing"

	"enviroo-be/internal/utils"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeString(t *testing.T) {
	kasus := map[string]string{
		"Plastik":            "plastik",
		"PLASTIK":            "plastik",
		"Botol Plastik":      "botolplastik",
		"Botol-Plastik":      "botolplastik",
		"  botol_plastik!! ": "botolplastik",
		"Kertas A4":          "kertasa4",
		"Kec. Tegalsari":     "kectegalsari",
		"":                   "",
		"!!! ---":            "",
		"Café":               "caf",
	}
	for masukan, harapan := range kasus {
		assert.Equal(t, harapan, utils.NormalizeString(masukan), "%q", masukan)
	}
}

func TestNormalizeString_PenulisanBedaDianggapSama(t *testing.T) {
	acuan := utils.NormalizeString("Botol Plastik")

	for _, varian := range []string{"botol-plastik", "BOTOL PLASTIK", "Botol.Plastik"} {
		assert.Equal(t, acuan, utils.NormalizeString(varian), "%q harus terdeteksi sama dengan \"Botol Plastik\"", varian)
	}
}
