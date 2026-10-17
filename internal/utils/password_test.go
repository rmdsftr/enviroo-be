package utils_test

import (
	"strings"
	"testing"

	"enviroo-be/internal/utils"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
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

func TestHashPassword_BisaDicocokkanKembali(t *testing.T) {
	hash, err := utils.HashPassword("Rahasia123")

	assert.NoError(t, err)
	assert.NotEqual(t, "Rahasia123", hash, "password tidak boleh disimpan polos")
	cost, err := bcrypt.Cost([]byte(hash))
	if assert.NoError(t, err) {
		assert.Equal(t, 12, cost)
	}
	assert.True(t, utils.CheckPassword(hash, "Rahasia123"))
}

func TestHashPassword_SaltSelaluBeda(t *testing.T) {
	hash1, err1 := utils.HashPassword("Rahasia123")
	hash2, err2 := utils.HashPassword("Rahasia123")

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotEqual(t, hash1, hash2, "password sama tetap menghasilkan hash berbeda")
	assert.True(t, utils.CheckPassword(hash1, "Rahasia123"))
	assert.True(t, utils.CheckPassword(hash2, "Rahasia123"))
}

func TestHashPassword_LebihDari72Byte(t *testing.T) {
	hash, err := utils.HashPassword("Rahasia1" + strings.Repeat("a", 65))

	assert.ErrorIs(t, err, bcrypt.ErrPasswordTooLong)
	assert.Empty(t, hash)
}

func TestCheckPassword_PasswordSalah(t *testing.T) {
	hash, err := utils.HashPassword("Rahasia123")
	if !assert.NoError(t, err) {
		return
	}

	for _, salah := range []string{"Rahasia124", "rahasia123", "RAHASIA123", ""} {
		assert.False(t, utils.CheckPassword(hash, salah), salah)
	}
}

func TestCheckPassword_HashTidakValid(t *testing.T) {
	for _, hash := range []string{"bukan-hash", "", "$2a$12$terpotong"} {
		assert.NotPanics(t, func() {
			assert.False(t, utils.CheckPassword(hash, "Rahasia123"), hash)
		})
	}
}
