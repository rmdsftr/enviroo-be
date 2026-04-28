package utils

import (
	crand "crypto/rand"
	"fmt"
	"math/big"
	mrand "math/rand"
	"strings"
	"time"

	"enviroo-be/internal/models"
)

// GenerateID membuat ID generik dengan format PREFIX-YYYYMMDDHHMMSS-RANDOM4
func GenerateID(prefix string) string {
	now := time.Now()
	randomPart := mrand.Intn(9000) + 1000 // 1000–9999
	if prefix == "" {
		return fmt.Sprintf("%s-%04d", now.Format("20060102150405"), randomPart)
	}
	return fmt.Sprintf("%s-%s-%04d", prefix, now.Format("20060102150405"), randomPart)
}

// GenerateAktivasiID membuat ID Aktivasi dengan format USERID-YYYYMMDDHHMMSS-RANDOM4
func GenerateAktivasiID(userID string) string {
	now := time.Now()
	randomPart := mrand.Intn(9000) + 1000
	return fmt.Sprintf("%s-%s-%04d", userID, now.Format("20060102150405"), randomPart)
}

// GenerateAdminID membuat ID Admin dengan format YYYYMMDD-RANDOM4
func GenerateAdminID() string {
	now := time.Now()
	randomPart := mrand.Intn(9000) + 1000
	return fmt.Sprintf("%s%04d", now.Format("20060102"), randomPart)
}

// GenerateBankRelatedID membuat ID yang terikat dengan Bank Sampah dengan format BANKID-YYYYMMDDHHMMSS-RANDOM4
func GenerateBankRelatedID(bankID string) string {
	now := time.Now()
	randomPart := mrand.Intn(9000) + 1000
	return fmt.Sprintf("%s-%s-%04d", bankID, now.Format("20060102150405"), randomPart)
}

// GenerateNasabahID membuat nomor rekening nasabah berdasarkan jenis bank dan ID bank.
// Jika jenisBank = BSU, direkomendasikan mengirimkan ID dari parent_bank_id (BSI) untuk memenuhi format (0103 + BSI_ID + RANDOM).
// Jika jenisBank = BSM, akan menggunakan format (0200 + BSM_ID + RANDOM).
func GenerateNasabahID(jenisBank models.JenisBank, referenceBankID string) string {
	var prefix string

	switch jenisBank {
	case models.BSU:
		prefix = "0103"
	case models.BSM:
		prefix = "0200"
	case models.BSI:
		// Asumsi bila ada nasabah yang direct daftar ke BSI
		prefix = "0100"
	default:
		prefix = "0000"
	}

	// Memastikan ID Bank persis 5 digit (diambil ujung belakang jika lebih panjang, dan di-padding jika kurang)
	formattedBankID := strings.TrimSpace(referenceBankID)
	if len(formattedBankID) > 5 {
		formattedBankID = formattedBankID[len(formattedBankID)-5:]
	} else {
		formattedBankID = fmt.Sprintf("%05s", formattedBankID)
	}

	// Generate angka random 4 digit
	n, err := crand.Int(crand.Reader, big.NewInt(10000))
	randomNum := "1234" // Fallback dasar bila error (sangat jarang)
	if err == nil {
		randomNum = fmt.Sprintf("%04d", n.Int64())
	}

	// Gabungkan tanpa space seperti nomor rekening aslinya
	return fmt.Sprintf("%s%s%s", prefix, formattedBankID, randomNum)
}
