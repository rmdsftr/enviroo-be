package idgen

import (
	"fmt"
	mrand "math/rand"
	"strconv"
	"time"

	"enviroo-be/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GenerateID(prefix string) string {
	now := time.Now()
	randomPart := mrand.Intn(9000) + 1000
	if prefix == "" {
		return fmt.Sprintf("%s-%04d", now.Format("20060102150405"), randomPart)
	}
	return fmt.Sprintf("%s-%s-%04d", prefix, now.Format("20060102150405"), randomPart)
}

func GenerateBankRelatedID(bankID string) string {
	now := time.Now()
	randomPart := mrand.Intn(9000) + 1000
	return fmt.Sprintf("%s-%s-%04d", bankID, now.Format("20060102150405"), randomPart)
}

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

func GenerateNasabahID(tx *gorm.DB, bankID string) (string, error) {
	if len(bankID) == 0 {
		return "", fmt.Errorf("bank ID tidak boleh kosong untuk nasabah")
	}

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

func GenerateRekeningID(rewardID int, entityID string) (string, error) {
	if entityID == "" {
		return "", fmt.Errorf("entity ID tidak boleh kosong untuk rekening")
	}
	if rewardID < 0 || rewardID > 9 {
		return "", fmt.Errorf("reward ID di luar jangkauan yang didukung skema ID rekening (0-9)")
	}

	return fmt.Sprintf("%d%s", rewardID, entityID), nil
}

func GenerateRiwayatSaldoID(tx *gorm.DB, rekeningID string) (string, error) {
	if rekeningID == "" {
		return "", fmt.Errorf("rekening ID tidak boleh kosong untuk riwayat saldo")
	}

	prefix := rekeningID + time.Now().Format("060102")

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.RiwayatArusSaldo{}).
		Where("riwayat_saldo_id LIKE ?", prefix+"%").
		Order("riwayat_saldo_id DESC").
		Limit(1).
		Pluck("riwayat_saldo_id", &lastID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}

	sequence := 1
	if lastID != "" && len(lastID) >= len(prefix)+3 {
		seqStr := lastID[len(lastID)-3:]
		if lastSeq, err := strconv.Atoi(seqStr); err == nil {
			sequence = lastSeq + 1
		}
	}

	if sequence > 999 {
		return "", fmt.Errorf("maksimal sequence riwayat saldo hari ini untuk rekening ini tercapai")
	}

	return fmt.Sprintf("%s%03d", prefix, sequence), nil
}

func GenerateBankID(tx *gorm.DB, jenisBank models.JenisBank, idKecamatan int, idKelurahan int) (string, error) {

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

	kodeKecamatan := fmt.Sprintf("%02d", idKecamatan)
	kodeKelurahan := fmt.Sprintf("%03d", idKelurahan)

	prefix := kodeBank + kodeKecamatan + kodeKelurahan

	var lastBankID string
	err := tx.Model(&models.BankSampah{}).
		Where("bank_id LIKE ?", prefix+"%").
		Order("bank_id DESC").
		Limit(1).
		Pluck("bank_id", &lastBankID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", fmt.Errorf("gagal query bank_id terakhir: %w", err)
	}

	sequence := 1
	if lastBankID != "" && len(lastBankID) == 9 {

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

func GenerateJadwalID(tx *gorm.DB, isRutin bool, jenisJadwal models.JadwalEnum, bankID string) (string, error) {
	kodeRutin := "K"
	if isRutin {
		kodeRutin = "R"
	}

	var kodeJenis string
	switch jenisJadwal {
	case models.JadwalPenimbangan:
		kodeJenis = "T"
	case models.JadwalPengangkutan:
		kodeJenis = "A"
	default:
		return "", fmt.Errorf("jenis jadwal tidak valid")
	}

	if bankID == "" {
		return "", fmt.Errorf("bank ID tidak boleh kosong untuk jadwal")
	}

	prefix := kodeRutin + kodeJenis + bankID

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.Jadwal{}).
		Where("jadwal_id LIKE ?", prefix+"%").
		Order("jadwal_id DESC").
		Limit(1).
		Pluck("jadwal_id", &lastID).Error
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
		return "", fmt.Errorf("maksimal sequence jadwal untuk kombinasi ini tercapai")
	}

	return fmt.Sprintf("%s%04d", prefix, sequence), nil
}

func GeneratePenimbanganID(tx *gorm.DB, bankID string) (string, error) {
	if bankID == "" {
		return "", fmt.Errorf("bank ID tidak boleh kosong untuk penimbangan")
	}

	prefix := bankID + time.Now().Format("060102")

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.Penimbangan{}).
		Where("penimbangan_id LIKE ?", prefix+"%").
		Order("penimbangan_id DESC").
		Limit(1).
		Pluck("penimbangan_id", &lastID).Error
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
		return "", fmt.Errorf("maksimal sequence penimbangan hari ini untuk bank ini tercapai")
	}

	return fmt.Sprintf("%s%02d", prefix, sequence), nil
}

func GeneratePenarikanID(tx *gorm.DB, rewardID int, nasabahID string) (string, error) {
	if nasabahID == "" {
		return "", fmt.Errorf("nasabah ID tidak boleh kosong untuk penarikan")
	}
	if rewardID < 0 || rewardID > 9 {
		return "", fmt.Errorf("reward ID di luar jangkauan yang didukung skema ID penarikan (0-9)")
	}

	prefix := fmt.Sprintf("%d", rewardID) + nasabahID + time.Now().Format("060102")

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.Penarikan{}).
		Where("penarikan_id LIKE ?", prefix+"%").
		Order("penarikan_id DESC").
		Limit(1).
		Pluck("penarikan_id", &lastID).Error
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
		return "", fmt.Errorf("maksimal sequence penarikan hari ini untuk nasabah ini tercapai")
	}

	return fmt.Sprintf("%s%02d", prefix, sequence), nil
}

func GenerateRiwayatPenarikanID(tx *gorm.DB, penarikanID string) (string, error) {
	if penarikanID == "" {
		return "", fmt.Errorf("penarikan ID tidak boleh kosong untuk riwayat penarikan")
	}

	prefix := penarikanID

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.RiwayatPenarikan{}).
		Where("riwayat_penarikan_id LIKE ?", prefix+"%").
		Order("riwayat_penarikan_id DESC").
		Limit(1).
		Pluck("riwayat_penarikan_id", &lastID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}

	sequence := 1
	if lastID != "" && len(lastID) >= len(prefix)+1 {
		seqStr := lastID[len(lastID)-1:]
		if lastSeq, err := strconv.Atoi(seqStr); err == nil {
			sequence = lastSeq + 1
		}
	}

	if sequence > 9 {
		return "", fmt.Errorf("maksimal sequence riwayat untuk penarikan ini tercapai")
	}

	return fmt.Sprintf("%s%d", prefix, sequence), nil
}

func GenerateMitraID(tx *gorm.DB, bankID string) (string, error) {
	if bankID == "" {
		return "", fmt.Errorf("bank ID tidak boleh kosong untuk mitra")
	}

	prefix := "MP" + bankID

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.MitraPengepul{}).
		Where("mitra_id LIKE ?", prefix+"%").
		Order("mitra_id DESC").
		Limit(1).
		Pluck("mitra_id", &lastID).Error
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
		return "", fmt.Errorf("maksimal sequence mitra untuk bank ini tercapai")
	}

	return fmt.Sprintf("%s%02d", prefix, sequence), nil
}

func GenerateBagiHasilID(penjualanID string) string {
	return "B" + penjualanID
}

func GeneratePenerimaBagiHasilID(tx *gorm.DB, penjualanID string) (string, error) {
	if penjualanID == "" {
		return "", fmt.Errorf("penjualan ID tidak boleh kosong untuk penerima bagi hasil")
	}

	prefix := "P" + penjualanID

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.PenerimaBagiHasil{}).
		Where("penerima_id LIKE ?", prefix+"%").
		Order("penerima_id DESC").
		Limit(1).
		Pluck("penerima_id", &lastID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}

	sequence := 1
	if lastID != "" && len(lastID) >= len(prefix)+3 {
		seqStr := lastID[len(lastID)-3:]
		if lastSeq, err := strconv.Atoi(seqStr); err == nil {
			sequence = lastSeq + 1
		}
	}

	if sequence > 999 {
		return "", fmt.Errorf("maksimal sequence penerima bagi hasil untuk penjualan ini tercapai")
	}

	return fmt.Sprintf("%s%03d", prefix, sequence), nil
}

func GenerateDistribusiSisaID(penjualanID string) string {
	return "D" + penjualanID
}

func GeneratePenerimaSisaID(tx *gorm.DB, penjualanID string) (string, error) {
	if penjualanID == "" {
		return "", fmt.Errorf("penjualan ID tidak boleh kosong untuk penerima distribusi sisa")
	}

	prefix := "S" + penjualanID

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.PenerimaDistribusiSisa{}).
		Where("penerima_sisa_id LIKE ?", prefix+"%").
		Order("penerima_sisa_id DESC").
		Limit(1).
		Pluck("penerima_sisa_id", &lastID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}

	sequence := 1
	if lastID != "" && len(lastID) >= len(prefix)+3 {
		seqStr := lastID[len(lastID)-3:]
		if lastSeq, err := strconv.Atoi(seqStr); err == nil {
			sequence = lastSeq + 1
		}
	}

	if sequence > 999 {
		return "", fmt.Errorf("maksimal sequence penerima distribusi sisa untuk penjualan ini tercapai")
	}

	return fmt.Sprintf("%s%03d", prefix, sequence), nil
}

func GenerateSetoranID(tx *gorm.DB, penimbanganID string) (string, error) {
	if penimbanganID == "" {
		return "", fmt.Errorf("penimbangan ID tidak boleh kosong untuk setoran")
	}

	prefix := penimbanganID

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.SetoranNasabah{}).
		Where("setoran_id LIKE ?", prefix+"%").
		Order("setoran_id DESC").
		Limit(1).
		Pluck("setoran_id", &lastID).Error
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
		return "", fmt.Errorf("maksimal sequence setoran untuk sesi ini tercapai")
	}

	return fmt.Sprintf("%s%04d", prefix, sequence), nil
}

func GeneratePenjualanID(tx *gorm.DB, rewardID int, bankID string) (string, error) {
	if bankID == "" {
		return "", fmt.Errorf("bank ID tidak boleh kosong untuk penjualan")
	}
	if rewardID < 0 || rewardID > 9 {
		return "", fmt.Errorf("reward ID di luar jangkauan yang didukung skema ID penjualan (0-9)")
	}

	prefix := fmt.Sprintf("%d", rewardID) + bankID + time.Now().Format("060102")

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.Penjualan{}).
		Where("penjualan_id LIKE ?", prefix+"%").
		Order("penjualan_id DESC").
		Limit(1).
		Pluck("penjualan_id", &lastID).Error
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
		return "", fmt.Errorf("maksimal sequence penjualan hari ini untuk bank ini tercapai")
	}

	return fmt.Sprintf("%s%02d", prefix, sequence), nil
}

func GenerateSampahID(bankID string, sarokID int, rewardID int) (string, error) {
	if bankID == "" {
		return "", fmt.Errorf("bank ID tidak boleh kosong untuk sampah")
	}
	if sarokID < 0 || sarokID > 999 {
		return "", fmt.Errorf("sarok ID di luar jangkauan yang didukung skema ID sampah (0-999)")
	}
	if rewardID < 0 || rewardID > 9 {
		return "", fmt.Errorf("reward ID di luar jangkauan yang didukung skema ID sampah (0-9)")
	}

	return fmt.Sprintf("%s%03d%d", bankID, sarokID, rewardID), nil
}

func GenerateSchemaHargaID(sampahID string, levelUser models.LevelUser) (string, error) {
	if sampahID == "" {
		return "", fmt.Errorf("sampah ID tidak boleh kosong untuk schema harga")
	}

	var kodeLevel string
	switch levelUser {
	case models.LevelEksternal:
		kodeLevel = "1"
	case models.LevelNasabah:
		kodeLevel = "2"
	default:
		return "", fmt.Errorf("level user tidak valid untuk skema ID schema harga")
	}

	return kodeLevel + sampahID, nil
}

func GenerateHistorySampahID(tx *gorm.DB, schemaID string) (string, error) {
	if schemaID == "" {
		return "", fmt.Errorf("schema ID tidak boleh kosong untuk history sampah")
	}

	prefix := schemaID

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.KatalogSampahHistory{}).
		Where("history_sampah_id LIKE ?", prefix+"%").
		Order("history_sampah_id DESC").
		Limit(1).
		Pluck("history_sampah_id", &lastID).Error
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
		return "", fmt.Errorf("maksimal sequence history harga untuk schema ini tercapai")
	}

	return fmt.Sprintf("%s%04d", prefix, sequence), nil
}

func GenerateProdukID(bankID string, barangID int) (string, error) {
	if bankID == "" {
		return "", fmt.Errorf("bank ID tidak boleh kosong untuk produk")
	}
	if barangID < 0 || barangID > 999 {
		return "", fmt.Errorf("barang ID di luar jangkauan yang didukung skema ID produk (0-999)")
	}

	return fmt.Sprintf("%s%03d", bankID, barangID), nil
}

func GeneratePengangkutanID(tx *gorm.DB, bsiID string) (string, error) {
	if bsiID == "" {
		return "", fmt.Errorf("BSI ID tidak boleh kosong untuk pengangkutan")
	}

	prefix := bsiID + time.Now().Format("060102")

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.PengangkutanSampah{}).
		Where("pengangkutan_id LIKE ?", prefix+"%").
		Order("pengangkutan_id DESC").
		Limit(1).
		Pluck("pengangkutan_id", &lastID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}

	sequence := 1
	if lastID != "" && len(lastID) >= len(prefix)+3 {
		seqStr := lastID[len(lastID)-3:]
		if lastSeq, err := strconv.Atoi(seqStr); err == nil {
			sequence = lastSeq + 1
		}
	}

	if sequence > 999 {
		return "", fmt.Errorf("maksimal sequence pengangkutan hari ini untuk BSI ini tercapai")
	}

	return fmt.Sprintf("%s%03d", prefix, sequence), nil
}

func GenerateTabunganID(tx *gorm.DB, sourceID string) (string, error) {
	if sourceID == "" {
		return "", fmt.Errorf("source ID tidak boleh kosong untuk tabungan sampah")
	}

	prefix := sourceID

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.TabunganSampah{}).
		Where("tabungan_id LIKE ?", prefix+"%").
		Order("tabungan_id DESC").
		Limit(1).
		Pluck("tabungan_id", &lastID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}

	sequence := 1
	if lastID != "" && len(lastID) >= len(prefix)+3 {
		seqStr := lastID[len(lastID)-3:]
		if lastSeq, err := strconv.Atoi(seqStr); err == nil {
			sequence = lastSeq + 1
		}
	}

	if sequence > 999 {
		return "", fmt.Errorf("maksimal sequence tabungan sampah untuk source ini tercapai")
	}

	return fmt.Sprintf("%s%03d", prefix, sequence), nil
}

func GenerateRiwayatPengangkutanID(tx *gorm.DB, pengangkutanID string) (string, error) {
	if pengangkutanID == "" {
		return "", fmt.Errorf("pengangkutan ID tidak boleh kosong untuk riwayat pengangkutan")
	}

	prefix := pengangkutanID

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.RiwayatPengangkutan{}).
		Where("riwayat_pengangkutan_id LIKE ?", prefix+"%").
		Order("riwayat_pengangkutan_id DESC").
		Limit(1).
		Pluck("riwayat_pengangkutan_id", &lastID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}

	sequence := 1
	if lastID != "" && len(lastID) >= len(prefix)+1 {
		seqStr := lastID[len(lastID)-1:]
		if lastSeq, err := strconv.Atoi(seqStr); err == nil {
			sequence = lastSeq + 1
		}
	}

	if sequence > 9 {
		return "", fmt.Errorf("maksimal sequence riwayat untuk pengangkutan ini tercapai")
	}

	return fmt.Sprintf("%s%d", prefix, sequence), nil
}

func GenerateNilaiRewardID(bankID string, rewardID int, levelUser models.LevelUser) (string, error) {
	if bankID == "" {
		return "", fmt.Errorf("bank ID tidak boleh kosong untuk nilai reward")
	}
	if rewardID < 0 || rewardID > 9 {
		return "", fmt.Errorf("reward ID di luar jangkauan yang didukung skema ID nilai reward (0-9)")
	}

	var kodeLevel string
	switch levelUser {
	case models.LevelEksternal:
		kodeLevel = "1"
	case models.LevelNasabah:
		kodeLevel = "2"
	default:
		return "", fmt.Errorf("level user tidak valid untuk skema ID nilai reward")
	}

	return fmt.Sprintf("%s%s%d", kodeLevel, bankID, rewardID), nil
}

func GenerateHistoryRewardID(tx *gorm.DB, nilaiRewardID string) (string, error) {
	if nilaiRewardID == "" {
		return "", fmt.Errorf("nilai reward ID tidak boleh kosong untuk history reward")
	}

	prefix := nilaiRewardID

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.HistoryNilaiReward{}).
		Where("history_reward_id LIKE ?", prefix+"%").
		Order("history_reward_id DESC").
		Limit(1).
		Pluck("history_reward_id", &lastID).Error
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
		return "", fmt.Errorf("maksimal sequence history untuk nilai reward ini tercapai")
	}

	return fmt.Sprintf("%s%04d", prefix, sequence), nil
}

func GenerateHistoryBankID(tx *gorm.DB, action string, bankID string) (string, error) {
	if bankID == "" {
		return "", fmt.Errorf("bank ID tidak boleh kosong untuk history akun bank")
	}

	var kodeEvent string
	switch action {
	case "CREATE":
		kodeEvent = "1"
	case "UPDATE":
		kodeEvent = "2"
	case "DELETE":
		kodeEvent = "3"
	default:
		return "", fmt.Errorf("action tidak valid untuk skema ID history akun bank")
	}

	prefix := kodeEvent + bankID

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.HistoryAkunBank{}).
		Where("history_bank_id LIKE ?", prefix+"%").
		Order("history_bank_id DESC").
		Limit(1).
		Pluck("history_bank_id", &lastID).Error
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
		return "", fmt.Errorf("maksimal sequence history untuk bank dan kode event ini tercapai")
	}

	return fmt.Sprintf("%s%04d", prefix, sequence), nil
}

func GenerateAktivasiID(tx *gorm.DB, userID string, asRole models.RoleUserEnum, tujuan models.TujuanEnum) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("user ID tidak boleh kosong untuk aktivasi akun")
	}

	var kodeRole string
	switch asRole {
	case models.RoleUserAdmin:
		kodeRole = "1"
	case models.RoleUserNasabah:
		kodeRole = "2"
	case models.RoleUserGlobal:
		kodeRole = "3"
	default:
		return "", fmt.Errorf("as_role tidak valid untuk skema ID aktivasi akun")
	}

	var kodeTujuan string
	switch tujuan {
	case models.TujuanAktivasi:
		kodeTujuan = "1"
	case models.TujuanResetPassword:
		kodeTujuan = "2"
	default:
		return "", fmt.Errorf("tujuan tidak valid untuk skema ID aktivasi akun")
	}

	prefix := userID + kodeRole + kodeTujuan

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.AktivasiAkun{}).
		Where("aktivasi_id LIKE ?", prefix+"%").
		Order("aktivasi_id DESC").
		Limit(1).
		Pluck("aktivasi_id", &lastID).Error
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
		return "", fmt.Errorf("maksimal sequence aktivasi untuk kombinasi ini tercapai")
	}

	return fmt.Sprintf("%s%02d", prefix, sequence), nil
}

func GenerateSettingSisaBagiHasilID(bankID string) (string, error) {
	if bankID == "" {
		return "", fmt.Errorf("bank ID tidak boleh kosong untuk setting sisa bagi hasil")
	}

	return "SET" + bankID, nil
}

func GenerateKontenID(tx *gorm.DB, adminID string) (string, error) {
	if adminID == "" {
		return "", fmt.Errorf("admin ID tidak boleh kosong untuk konten")
	}

	prefix := adminID + time.Now().Format("060102")

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.Konten{}).
		Where("konten_id LIKE ?", prefix+"%").
		Order("konten_id DESC").
		Limit(1).
		Pluck("konten_id", &lastID).Error
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
		return "", fmt.Errorf("maksimal sequence konten hari ini untuk admin ini tercapai")
	}

	return fmt.Sprintf("%s%02d", prefix, sequence), nil
}

func GenerateDisbaID(tx *gorm.DB, bsiID string) (string, error) {
	if bsiID == "" {
		return "", fmt.Errorf("BSI ID tidak boleh kosong untuk distribusi barang")
	}

	prefix := bsiID + time.Now().Format("060102")

	var lastID string
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&models.DistribusiBarangBsu{}).
		Where("disba_id LIKE ?", prefix+"%").
		Order("disba_id DESC").
		Limit(1).
		Pluck("disba_id", &lastID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}

	sequence := 1
	if lastID != "" && len(lastID) >= len(prefix)+3 {
		seqStr := lastID[len(lastID)-3:]
		if lastSeq, err := strconv.Atoi(seqStr); err == nil {
			sequence = lastSeq + 1
		}
	}

	if sequence > 999 {
		return "", fmt.Errorf("maksimal sequence distribusi barang hari ini untuk BSI ini tercapai")
	}

	return fmt.Sprintf("%s%03d", prefix, sequence), nil
}
