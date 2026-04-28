package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

const otpChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const otpLength = 6

// GenerateOTP menghasilkan kode OTP 6 karakter (huruf besar + angka).
// Menggunakan crypto/rand agar aman secara kriptografis.
//
// Contoh output: "A3TF9R"
func GenerateOTP() (string, error) {
	result := make([]byte, otpLength)
	for i := range result {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(otpChars))))
		if err != nil {
			return "", fmt.Errorf("gagal generate OTP: %w", err)
		}
		result[i] = otpChars[idx.Int64()]
	}
	return string(result), nil
}

// HashOTP meng-hash kode OTP plaintext menggunakan bcrypt sebelum disimpan ke database.
// Cost default bcrypt (10) digunakan. Simpan nilai yang dikembalikan ke kolom token.
//
// Contoh:
//
//	hashed, err := utils.HashOTP("A3TF9R")
//	// simpan hashed ke database
func HashOTP(otp string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("gagal hash OTP: %w", err)
	}
	return string(bytes), nil
}

// VerifyOTP memverifikasi apakah kode OTP yang dimasukkan user cocok dengan hash yang tersimpan di database.
// Kembalikan true jika cocok, false jika tidak.
//
// Contoh:
//
//	match := utils.VerifyOTP("A3TF9R", hashedTokenDariDB)
func VerifyOTP(plainOTP, hashedOTP string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedOTP), []byte(plainOTP))
	return err == nil
}
