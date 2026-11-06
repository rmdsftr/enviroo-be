package utils_test

import (
	"testing"

	"enviroo-be/internal/utils"

	"github.com/stretchr/testify/assert"
)

func TestRound4(t *testing.T) {
	a, b := 0.1, 0.2 // variabel supaya dijumlah saat runtime (float64), bukan konstanta eksak
	kasus := []struct {
		nama    string
		masukan float64
		harapan float64
	}{
		{"noise positif jadi 0", 2.7755575615628914e-17, 0},
		{"noise negatif jadi 0", -5.551115123125783e-17, 0},
		{"0.1+0.2 jadi 0.3", a + b, 0.3},
		{"sudah 4 desimal tetap", 1234.5678, 1234.5678},
		{"dibulatkan ke atas", 1.23456, 1.2346},
		{"dibulatkan ke bawah", 1.23454, 1.2345},
		{"negatif dibulatkan", -1.23456, -1.2346},
		{"angka besar (ratusan juta) tetap presisi", 987654321.98765432, 987654321.9877},
		{"nol", 0, 0},
	}
	for _, k := range kasus {
		assert.Equal(t, k.harapan, utils.Round4(k.masukan), k.nama)
	}
}
