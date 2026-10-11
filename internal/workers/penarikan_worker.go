package workers

import (
	"context"
	"enviroo-be/internal/adapters/fcm"
	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PenarikanWorker struct {
	DB       *gorm.DB
	NotifSvc services.NotifikasiService
	Interval time.Duration
	stopCh   chan struct{}
}

func NewPenarikanWorker(db *gorm.DB, fcmClient *fcm.Client, interval time.Duration) *PenarikanWorker {
	notifRepo := repositories.NewNotifikasiRepository(db)
	notifSvc := services.NewNotifikasiService(notifRepo, fcmClient)
	return &PenarikanWorker{
		DB:       db,
		NotifSvc: notifSvc,
		Interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (w *PenarikanWorker) Start() {
	go func() {
		log.Println("[PenarikanWorker] Worker started")
		ticker := time.NewTicker(w.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.prosesRequestExpired()
				w.prosesPickupExpired()
			case <-w.stopCh:
				log.Println("[PenarikanWorker] Worker stopped")
				return
			}
		}
	}()
}

func (w *PenarikanWorker) Stop() {
	close(w.stopCh)
}

func (w *PenarikanWorker) prosesRequestExpired() {
	var list []models.Penarikan
	if err := w.DB.
		Preload("Reward").
		Where("status_penarikan = ? AND deadline_konfirmasi < ?", models.StatusPenarikanPending, time.Now()).
		Find(&list).Error; err != nil {
		log.Printf("[PenarikanWorker] Gagal query request_expired: %v", err)
		return
	}
	if len(list) == 0 {
		return
	}

	log.Printf("[PenarikanWorker] Memproses %d penarikan request_expired", len(list))
	for _, p := range list {
		if err := w.prosesPerPenarikan(p, models.StatusPenarikanPending, models.StatusPenarikanRequestExpired); err != nil {
			log.Printf("[PenarikanWorker] Gagal proses request_expired %s: %v", p.PenarikanID, err)
			continue
		}
		log.Printf("[PenarikanWorker] Berhasil proses request_expired: %s", p.PenarikanID)
	}
}

func (w *PenarikanWorker) prosesPickupExpired() {
	var list []models.Penarikan
	if err := w.DB.
		Preload("Reward").
		Where("status_penarikan = ? AND deadline_jemput < ?", models.StatusPenarikanApproved, time.Now()).
		Find(&list).Error; err != nil {
		log.Printf("[PenarikanWorker] Gagal query pickup_expired: %v", err)
		return
	}
	if len(list) == 0 {
		return
	}

	log.Printf("[PenarikanWorker] Memproses %d penarikan pickup_expired", len(list))
	for _, p := range list {
		if err := w.prosesPerPenarikan(p, models.StatusPenarikanApproved, models.StatusPenarikanPickupExpired); err != nil {
			log.Printf("[PenarikanWorker] Gagal proses pickup_expired %s: %v", p.PenarikanID, err)
			continue
		}
		log.Printf("[PenarikanWorker] Berhasil proses pickup_expired: %s", p.PenarikanID)
	}
}

func (w *PenarikanWorker) prosesPerPenarikan(penarikan models.Penarikan, fromStatus, toStatus models.StatusPenarikanEnum) error {
	const sistem = "sistem"

	err := w.DB.Transaction(func(tx *gorm.DB) error {
		var p models.Penarikan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("penarikan_id = ? AND status_penarikan = ?", penarikan.PenarikanID, fromStatus).
			First(&p).Error; err != nil {
			return nil
		}

		if err := tx.Model(&p).Updates(map[string]interface{}{
			"status_penarikan": toStatus,
		}).Error; err != nil {
			return err
		}

		sistemVal := sistem
		riwayatID, err := idgen.GenerateRiwayatPenarikanID(tx, p.PenarikanID)
		if err != nil {
			return err
		}
		if err := tx.Create(&models.RiwayatPenarikan{
			RiwayatPenarikanID: riwayatID,
			PenarikanID:        p.PenarikanID,
			StatusPenarikan:    toStatus,
			CreatedBy:          &sistemVal,
		}).Error; err != nil {
			return err
		}

		if p.NasabahID == nil || p.RewardID == nil {
			return fmt.Errorf("data nasabah_id/reward_id pada penarikan %s tidak valid", p.PenarikanID)
		}

		var saldo models.SaldoRekening
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("nasabah_id = ? AND reward_id = ?", *p.NasabahID, *p.RewardID).
			First(&saldo).Error; err != nil {
			return err
		}

		saldoSebelum := saldo.NominalSaldo
		saldoSesudah := saldoSebelum + p.NominalPenarikan

		if err := tx.Model(&saldo).Updates(map[string]interface{}{
			"nominal_saldo":   saldoSesudah,
			"last_updated_at": time.Now(),
			"last_updated_by": sistem,
		}).Error; err != nil {
			return err
		}

		riwayatSaldoID, err := idgen.GenerateRiwayatSaldoID(tx, saldo.RekeningID)
		if err != nil {
			return err
		}
		if err := tx.Create(&models.RiwayatArusSaldo{
			RiwayatSaldoID: riwayatSaldoID,
			RekeningID:     &saldo.RekeningID,
			NominalSebelum: saldoSebelum,
			NominalSesudah: saldoSesudah,
			CreatedBy:      &sistemVal,
			Keterangan:     "Pengembalian saldo",
		}).Error; err != nil {
			return err
		}

		if penarikan.Reward != nil && penarikan.Reward.NamaReward == models.RewardEnumBarang {
			if p.BankID == nil {
				return fmt.Errorf("data bank_id pada penarikan %s tidak valid", p.PenarikanID)
			}
			var details []models.DetailPenarikanBarang
			if err := tx.Where("penarikan_id = ?", p.PenarikanID).Find(&details).Error; err != nil {
				return err
			}
			for _, d := range details {
				if err := tx.Model(&models.StokBarang{}).
					Where("produk_id = ? AND bank_id = ?", d.ProdukID, *p.BankID).
					UpdateColumn("stok", gorm.Expr("stok + ?", d.Qty)).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	w.kirimNotifKadaluarsa(penarikan, toStatus)
	return nil
}

func (w *PenarikanWorker) kirimNotifKadaluarsa(penarikan models.Penarikan, toStatus models.StatusPenarikanEnum) {
	go func() {
		if penarikan.NasabahID == nil {
			return
		}

		var nas models.Nasabah
		if err := w.DB.Preload("User").Where("nasabah_id = ?", *penarikan.NasabahID).First(&nas).Error; err != nil {
			log.Printf("[PenarikanWorker] Gagal ambil data nasabah buat notif: %v", err)
			return
		}

		pesanNilai := services.FormatPenarikanNilai(penarikan.NominalPenarikan, penarikan.SatuanPenarikan)
		ctx := context.Background()

		var errNotif error
		if toStatus == models.StatusPenarikanRequestExpired {
			errNotif = w.NotifSvc.NotifPenarikanRequestExpired(ctx, nas.User.UserID, nas.User.FCMToken, pesanNilai, penarikan.PenarikanID)
		} else {
			errNotif = w.NotifSvc.NotifPenarikanPickupExpired(ctx, nas.User.UserID, nas.User.FCMToken, pesanNilai, penarikan.PenarikanID)
		}
		if errNotif != nil {
			log.Printf("[PenarikanWorker] Gagal kirim notif ke nasabah %s: %v", nas.User.UserID, errNotif)
		}

		if penarikan.BankID == nil {
			return
		}

		var admins []repositories.AdminInfo
		if err := w.DB.Table("admin").
			Select("users.user_id, users.fcm_token").
			Joins("JOIN users ON users.user_id = admin.user_id").
			Where("admin.bank_id = ? AND admin.status_admin = ?", *penarikan.BankID, models.Aktif).
			Scan(&admins).Error; err != nil {
			log.Printf("[PenarikanWorker] Gagal ambil daftar admin/petugas bank: %v", err)
			return
		}

		for _, au := range admins {
			var errAdmin error
			if toStatus == models.StatusPenarikanRequestExpired {
				errAdmin = w.NotifSvc.NotifPenarikanRequestExpiredAdmin(ctx, au.UserID, au.FCMToken, nas.User.Nama, pesanNilai, penarikan.PenarikanID)
			} else {
				errAdmin = w.NotifSvc.NotifPenarikanPickupExpiredAdmin(ctx, au.UserID, au.FCMToken, nas.User.Nama, pesanNilai, penarikan.PenarikanID)
			}
			if errAdmin != nil {
				log.Printf("[PenarikanWorker] Gagal kirim notif ke admin/petugas %s: %v", au.UserID, errAdmin)
			}
		}
	}()
}
