package workers

import (
	"enviroo-be/internal/models"
	"enviroo-be/pkg/utils"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PenarikanWorker struct {
	DB       *gorm.DB
	Interval time.Duration
	stopCh   chan struct{}
}

func NewPenarikanWorker(db *gorm.DB, interval time.Duration) *PenarikanWorker {
	return &PenarikanWorker{
		DB:       db,
		Interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start menjalankan worker di goroutine terpisah
func (w *PenarikanWorker) Start() {
	go func() {
		log.Println("[PenarikanWorker] Worker started")
		ticker := time.NewTicker(w.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.prosesKadaluarsa()
			case <-w.stopCh:
				log.Println("[PenarikanWorker] Worker stopped")
				return
			}
		}
	}()
}

// Stop menghentikan worker secara graceful
func (w *PenarikanWorker) Stop() {
	close(w.stopCh)
}

// prosesKadaluarsa mencari dan memproses semua penarikan yang sudah melewati kadaluarsa_at
func (w *PenarikanWorker) prosesKadaluarsa() {
	// Ambil semua penarikan pending yang sudah kadaluarsa
	// Lock for update langsung di query awal untuk cegah race condition
	// kalau nanti di-scale ke multiple instance
	var penarikanList []models.Penarikan
	if err := w.DB.
		Preload("Reward").
		Where("status_penarikan = ? AND kadaluarsa_at < ?", models.StatusPenarikanPending, time.Now()).
		Find(&penarikanList).Error; err != nil {
		log.Printf("[PenarikanWorker] Gagal query penarikan kadaluarsa: %v", err)
		return
	}

	if len(penarikanList) == 0 {
		return
	}

	log.Printf("[PenarikanWorker] Memproses %d penarikan kadaluarsa", len(penarikanList))

	for _, penarikan := range penarikanList {
		if err := w.prosesPerPenarikan(penarikan); err != nil {
			log.Printf("[PenarikanWorker] Gagal proses penarikan %s: %v", penarikan.PenarikanID, err)
			// Lanjut ke penarikan berikutnya, jangan stop semua
			continue
		}
		log.Printf("[PenarikanWorker] Berhasil proses kadaluarsa: %s", penarikan.PenarikanID)
	}
}

func (w *PenarikanWorker) prosesPerPenarikan(penarikan models.Penarikan) error {
	return w.DB.Transaction(func(tx *gorm.DB) error {

		// ── 1. Lock & re-check status (cegah double-process) ─────────────
		var p models.Penarikan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("penarikan_id = ? AND status_penarikan = ?", penarikan.PenarikanID, models.StatusPenarikanPending).
			First(&p).Error; err != nil {
			// Sudah diproses oleh instance lain, skip
			return nil
		}

		// ── 2. Update status → kadaluarsa ────────────────────────────────
		now := time.Now()
		workerID := "system-worker"
		if err := tx.Model(&p).Updates(map[string]interface{}{
			"status_penarikan": models.StatusPenarikanKadaluarsa,
			"updated_at":       now,
			"updated_by":       workerID,
		}).Error; err != nil {
			return err
		}

		// ── 3. Ambil saldo rekening (dengan lock) ────────────────────────
		var saldo models.SaldoRekening
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("nasabah_id = ? AND reward_id = ?", p.NasabahID, p.RewardID).
			First(&saldo).Error; err != nil {
			return err
		}

		saldoSebelum := saldo.NominalSaldo
		saldoSesudah := saldoSebelum + p.NominalPenarikan

		// ── 4. Kembalikan saldo ──────────────────────────────────────────
		if err := tx.Model(&saldo).Updates(map[string]interface{}{
			"nominal_saldo":   saldoSesudah,
			"last_updated_at": now,
			"last_updated_by": workerID,
		}).Error; err != nil {
			return err
		}

		// ── 5. Catat riwayat arus saldo ──────────────────────────────────
		riwayat := models.RiwayatArusSaldo{
			RiwayatSaldoID: utils.GenerateID("RS"),
			RekeningID:     &saldo.RekeningID,
			NominalSebelum: saldoSebelum,
			NominalSesudah: saldoSesudah,
			CreatedBy:      &workerID,
		}
		if err := tx.Create(&riwayat).Error; err != nil {
			return err
		}

		// ── 6. Jika sembako: kembalikan stok ─────────────────────────────
		if penarikan.Reward != nil && penarikan.Reward.NamaReward == models.RewardEnumSembako {
			var details []models.DetailPenarikanSembako
			if err := tx.Where("penarikan_id = ?", p.PenarikanID).Find(&details).Error; err != nil {
				return err
			}

			for _, d := range details {
				if err := tx.Model(&models.KatalogSembako{}).
					Where("sembako_id = ?", d.SembakoID).
					Update("stok", gorm.Expr("stok + ?", d.Qty)).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}