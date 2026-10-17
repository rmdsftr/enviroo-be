package utils_test

import (
	"crypto/rand"
	"errors"
	"strings"
	"testing"

	"enviroo-be/internal/utils"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

type readerGagal struct{ err error }

func (r readerGagal) Read([]byte) (int, error) { return 0, r.err }

func TestGenerateOTP_EnamDigitAngka(t *testing.T) {
	for i := 0; i < 200; i++ {
		otp, err := utils.GenerateOTP()

		if !assert.NoError(t, err) {
			return
		}
		assert.Len(t, otp, 6)
		assert.Regexp(t, `^[0-9]{6}$`, otp)
	}
}

func TestGenerateOTP_Acak(t *testing.T) {
	unik := map[string]bool{}
	for i := 0; i < 100; i++ {
		otp, err := utils.GenerateOTP()
		if !assert.NoError(t, err) {
			return
		}
		unik[otp] = true
	}

	assert.GreaterOrEqual(t, len(unik), 95, "OTP harus acak, bukan nilai yang berulang")
}

func TestGenerateOTP_SumberAcakGagal(t *testing.T) {
	errAcak := errors.New("entropy habis")
	asli := rand.Reader
	rand.Reader = readerGagal{err: errAcak}
	t.Cleanup(func() { rand.Reader = asli })

	otp, err := utils.GenerateOTP()

	assert.Empty(t, otp)
	assert.ErrorIs(t, err, errAcak)
	assert.ErrorContains(t, err, "gagal generate OTP")
}

func TestHashOTP_BisaDiverifikasi(t *testing.T) {
	hash, err := utils.HashOTP("123456")

	assert.NoError(t, err)
	assert.NotEqual(t, "123456", hash, "OTP tidak boleh disimpan polos")
	cost, err := bcrypt.Cost([]byte(hash))
	if assert.NoError(t, err) {
		assert.Equal(t, bcrypt.DefaultCost, cost)
	}
	assert.True(t, utils.VerifyOTP("123456", hash))
}

func TestHashOTP_SaltSelaluBeda(t *testing.T) {
	hash1, err1 := utils.HashOTP("123456")
	hash2, err2 := utils.HashOTP("123456")

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotEqual(t, hash1, hash2, "OTP sama tetap menghasilkan hash berbeda")
	assert.True(t, utils.VerifyOTP("123456", hash1))
	assert.True(t, utils.VerifyOTP("123456", hash2))
}

func TestHashOTP_LebihDari72Byte(t *testing.T) {
	hash, err := utils.HashOTP(strings.Repeat("1", 73))

	assert.Empty(t, hash)
	assert.ErrorIs(t, err, bcrypt.ErrPasswordTooLong)
	assert.ErrorContains(t, err, "gagal hash OTP")
}

func TestVerifyOTP_Salah(t *testing.T) {
	hash, err := utils.HashOTP("123456")
	if !assert.NoError(t, err) {
		return
	}

	for _, otp := range []string{"654321", "12345", ""} {
		assert.False(t, utils.VerifyOTP(otp, hash), "OTP %q", otp)
	}
	for _, hashRusak := range []string{"bukan-hash", ""} {
		assert.NotPanics(t, func() {
			assert.False(t, utils.VerifyOTP("123456", hashRusak), "hash %q", hashRusak)
		})
	}
}
