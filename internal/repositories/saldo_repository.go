package repositories

import (
	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"fmt"
	"time"

	"gorm.io/gorm"
)

func MutasiSaldo(tx *gorm.DB, saldo *models.SaldoRekening, delta float64, keterangan, adminID string, now time.Time) error {
	nominalSebelum := saldo.NominalSaldo
	saldo.NominalSaldo += delta
	saldo.LastUpdatedAt = now
	saldo.LastUpdatedBy = &adminID
	if err := tx.Save(saldo).Error; err != nil {
		return err
	}

	riwayatSaldoID, err := idgen.GenerateRiwayatSaldoID(tx, saldo.RekeningID)
	if err != nil {
		return err
	}

	return tx.Create(&models.RiwayatArusSaldo{
		RiwayatSaldoID: riwayatSaldoID,
		RekeningID:     &saldo.RekeningID,
		NominalSebelum: nominalSebelum,
		NominalSesudah: saldo.NominalSaldo,
		CreatedAt:      now,
		CreatedBy:      &adminID,
		Keterangan:     keterangan,
	}).Error
}

func UpdateSaldoRekening(
	tx *gorm.DB,
	bankID *string,
	nasabahID *string,
	rewardID int,
	reward models.Reward,
	nilai float64,
	entitas models.EntitasEnum,
	adminID string,
	now time.Time,
) error {
	var saldo models.SaldoRekening

	query := tx.Where("reward_id = ? AND entitas = ?", rewardID, entitas)
	if entitas == models.EntitasBankSampah {
		query = query.Where("bank_id = ?", *bankID)
	} else {
		query = query.Where("nasabah_id = ?", *nasabahID)
	}

	if err := query.First(&saldo).Error; err != nil {
		if entitas == models.EntitasNasabah {
			return fmt.Errorf("rekening nasabah %s untuk reward_id %d tidak ditemukan", *nasabahID, rewardID)
		}
		return fmt.Errorf("rekening bank %s untuk reward_id %d tidak ditemukan", *bankID, rewardID)
	}

	return MutasiSaldo(tx, &saldo, nilai, "Bagi hasil", adminID, now)
}
