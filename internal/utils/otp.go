package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

const otpChars = "0123456789"
const otpLength = 6

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

func HashOTP(otp string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("gagal hash OTP: %w", err)
	}
	return string(bytes), nil
}

func VerifyOTP(plainOTP, hashedOTP string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedOTP), []byte(plainOTP))
	return err == nil
}
