package utils_test

import (
	"testing"

	"enviroo-be/internal/utils"

	"github.com/stretchr/testify/assert"
)

const pesanPasswordLemah = "Password tidak memenuhi persyaratan"

func TestValidasiKekuatanPassword_Lolos(t *testing.T) {
	for _, password := range []string{"Rahasia123", "ABCDEFG1", "12345678A"} {
		assert.NoError(t, utils.ValidasiKekuatanPassword(password), password)
	}
}

func TestValidasiKekuatanPassword_Gagal(t *testing.T) {
	for _, password := range []string{
		"Rahas1a",
		"rahasia123",
		"RahasiaSaja",
		"rahasiasaja",
		"",
	} {
		err := utils.ValidasiKekuatanPassword(password)
		if assert.Error(t, err, password) {
			assert.Equal(t, pesanPasswordLemah, err.Error(), password)
		}
	}
}

func TestValidasiKekuatanPassword_PanjangPerKarakter(t *testing.T) {
	err := utils.ValidasiKekuatanPassword("Ré1éééé")

	if assert.Error(t, err) {
		assert.Equal(t, pesanPasswordLemah, err.Error())
	}
}
