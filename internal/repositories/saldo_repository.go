package repositories

import (
	"enviroo-be/internal/models"
	"enviroo-be/pkg/utils"
	"fmt"
	"time"

	"gorm.io/gorm"
)

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

	nominalSebelum := saldo.NominalSaldo
	saldo.NominalSaldo += nilai
	saldo.LastUpdatedAt = now
	saldo.LastUpdatedBy = &adminID
	if err := tx.Save(&saldo).Error; err != nil {
		return err
	}

	return tx.Create(&models.RiwayatArusSaldo{
		RiwayatSaldoID: utils.GenerateID("RS"),
		RekeningID:     &saldo.RekeningID,
		NominalSebelum: nominalSebelum,
		NominalSesudah: saldo.NominalSaldo,
		CreatedAt:      now,
		CreatedBy:      &adminID,
	}).Error
}
