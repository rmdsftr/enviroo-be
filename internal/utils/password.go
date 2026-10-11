package utils

import (
	"errors"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const passwordCost = 12

const passwordMinLength = 8

func HashPassword(plain string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), passwordCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func CheckPassword(hashed, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}

func ValidasiKekuatanPassword(plain string) error {
	var adaKapital, adaAngka bool
	for _, r := range plain {
		switch {
		case r >= 'A' && r <= 'Z':
			adaKapital = true
		case r >= '0' && r <= '9':
			adaAngka = true
		}
	}

	if utf8.RuneCountInString(plain) < passwordMinLength || !adaKapital || !adaAngka {
		return errors.New("Password tidak memenuhi persyaratan")
	}
	return nil
}
