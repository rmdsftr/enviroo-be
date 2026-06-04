package utils

import (
	"fmt"
	mrand "math/rand"
	"strconv"
	"time"

	"enviroo-be/internal/models"

	"gorm.io/gorm"
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

// GenerateBankRelatedID membuat ID untuk entitas yang berelasi dengan bank (katalog, sembako, penjualan, dst)
// Format: BankID-YYYYMMDDHHMMSS-RANDOM4
func GenerateBankRelatedID(bankID string) string {
	now := time.Now()
	randomPart := mrand.Intn(9000) + 1000
	return fmt.Sprintf("%s-%s-%04d", bankID, now.Format("20060102150405"), randomPart)
}

// GenerateAktivasiID membuat ID Aktivasi dengan format USERID-YYYYMMDDHHMMSS-RANDOM4
func GenerateAktivasiID(userID string) string {
	now := time.Now()
	randomPart := mrand.Intn(9000) + 1000
	return fmt.Sprintf("%s-%s-%04d", userID, now.Format("20060102150405"), randomPart)
}

// GenerateAdminID membuat ID Admin dengan format Kode Role (2 digit) + BankID (opsional) + Sequence (2 digit)
func GenerateAdminID(tx *gorm.DB, role models.RoleAdmin, bankID string) (string, error) {
	var kodeRole string
	switch role {
	case models.SuperAdmin:
		kodeRole = "10"
	case models.AdminBSI:
		kodeRole = "21"
	case models.AdminBSM:
		kodeRole = "22"
	case models.AdminBSU:
		kodeRole = "23"
	case models.PetugasBSI:
		kodeRole = "31"
	case models.PetugasBSM:
		kodeRole = "32"
	case models.PetugasBSU:
		kodeRole = "33"
	default:
		return "", fmt.Errorf("role tidak valid untuk admin")
	}

	var prefix string
	if role == models.SuperAdmin {
		prefix = kodeRole
	} else {
		prefix = kodeRole + bankID
	}

	var lastID string
	err := tx.Model(&models.Admin{}).
		Where("admin_id LIKE ?", prefix+"%").
		Order("admin_id DESC").
		Limit(1).
		Pluck("admin_id", &lastID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}

	sequence := 1
	if lastID != "" && len(lastID) >= len(prefix)+2 {
		seqStr := lastID[len(lastID)-2:]
		if lastSeq, err := strconv.Atoi(seqStr); err == nil {
			sequence = lastSeq + 1
		}
	}

	if sequence > 99 {
		return "", fmt.Errorf("maksimal sequence admin/petugas tercapai")
	}

	return fmt.Sprintf("%s%02d", prefix, sequence), nil
}

// GenerateNasabahID membuat ID Nasabah dengan format Kode Role (2 digit) + BankID + Sequence (4 digit)
func GenerateNasabahID(tx *gorm.DB, bankID string) (string, error) {
	if len(bankID) == 0 {
		return "", fmt.Errorf("bank ID tidak boleh kosong untuk nasabah")
	}

	// 1 = BSI, 2 = BSM, 3 = BSU
	bankType := bankID[0]
	var kodeRole string
	switch bankType {
	case '1':
		kodeRole = "41"
	case '2':
		kodeRole = "42"
	case '3':
		kodeRole = "43"
	default:
		return "", fmt.Errorf("tipe bank tidak valid pada bankID")
	}

	prefix := kodeRole + bankID

	var lastID string
	err := tx.Model(&models.Nasabah{}).
		Where("nasabah_id LIKE ?", prefix+"%").
		Order("nasabah_id DESC").
		Limit(1).
		Pluck("nasabah_id", &lastID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}

	sequence := 1
	if lastID != "" && len(lastID) >= len(prefix)+4 {
		seqStr := lastID[len(lastID)-4:]
		if lastSeq, err := strconv.Atoi(seqStr); err == nil {
			sequence = lastSeq + 1
		}
	}

	if sequence > 9999 {
		return "", fmt.Errorf("maksimal sequence nasabah tercapai")
	}

	return fmt.Sprintf("%s%04d", prefix, sequence), nil
}

// GenerateRekeningID membuat ID rekening dengan format {rewardID}-{entityID}-{random3digit}
func GenerateRekeningID(rewardID int, entityID string) string {
	randomPart := mrand.Intn(900) + 100 // 100–999
	return fmt.Sprintf("%d-%s-%03d", rewardID, entityID, randomPart)
}

// GenerateBankID membuat Bank ID dengan format:
// [kode_bank 1 digit] + [kode_kecamatan 2 digit] + [kode_kelurahan 3 digit] + [sequence 3 digit]
// Total: 9 digit
//
// Kode bank: BSI = 1, BSM = 2, BSU = 3
//
// Contoh: BSI di kecamatan 3, kelurahan 4, bank ke-15 => "103004015"
func GenerateBankID(tx *gorm.DB, jenisBank models.JenisBank, idKecamatan int, idKelurahan int) (string, error) {
	// Tentukan kode bank
	var kodeBank string
	switch jenisBank {
	case models.BSI:
		kodeBank = "1"
	case models.BSM:
		kodeBank = "2"
	case models.BSU:
		kodeBank = "3"
	default:
		kodeBank = "0"
	}

	// Format kode kecamatan (2 digit) dan kelurahan (3 digit)
	kodeKecamatan := fmt.Sprintf("%02d", idKecamatan)
	kodeKelurahan := fmt.Sprintf("%03d", idKelurahan)

	// Prefix = kode_bank + kode_kecamatan + kode_kelurahan
	prefix := kodeBank + kodeKecamatan + kodeKelurahan

	// Cari bank_id terbesar yang berawalan prefix ini (dalam transaction)
	var lastBankID string
	err := tx.Model(&models.BankSampah{}).
		Where("bank_id LIKE ?", prefix+"%").
		Order("bank_id DESC").
		Limit(1).
		Pluck("bank_id", &lastBankID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", fmt.Errorf("gagal query bank_id terakhir: %w", err)
	}

	// Hitung sequence berikutnya
	sequence := 1
	if lastBankID != "" && len(lastBankID) == 9 {
		// Ambil 3 digit terakhir sebagai sequence saat ini
		lastSeqStr := lastBankID[6:]
		lastSeq, err := strconv.Atoi(lastSeqStr)
		if err == nil {
			sequence = lastSeq + 1
		}
	}

	if sequence > 999 {
		return "", fmt.Errorf("sequence bank di wilayah ini sudah mencapai batas maksimal (999)")
	}

	bankID := fmt.Sprintf("%s%03d", prefix, sequence)
	return bankID, nil
}
