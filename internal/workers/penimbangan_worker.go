package workers

import (
	"log"
	"time"

	"gorm.io/gorm"
)

type PenimbanganWorker struct {
	DB     *gorm.DB
	stopCh chan struct{}
}

func NewPenimbanganWorker(db *gorm.DB) *PenimbanganWorker {
	return &PenimbanganWorker{
		DB:     db,
		stopCh: make(chan struct{}),
	}
}

func (w *PenimbanganWorker) Start() {
	go func() {
		log.Println("[PenimbanganWorker] Worker started")
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.tutupSesiKedaluwarsa()
			case <-w.stopCh:
				log.Println("[PenimbanganWorker] Worker stopped")
				return
			}
		}
	}()
}

func (w *PenimbanganWorker) Stop() {
	close(w.stopCh)
}

// tutupSesiKedaluwarsa menutup sesi penimbangan yang masih aktif
// namun sudah melewati 2 jam setelah jam_selesai jadwalnya.
func (w *PenimbanganWorker) tutupSesiKedaluwarsa() {
	result := w.DB.Exec(`
		UPDATE penimbangan p
		SET status_penimbangan = 'selesai',
		    ended_at            = NOW()
		FROM jadwal j
		WHERE j.jadwal_id              = p.jadwal_id
		  AND p.status_penimbangan     = 'aktif'
		  AND (DATE(p.started_at) || ' ' || j.jam_selesai)::timestamp
		      + INTERVAL '2 hours'     < NOW()
	`)

	if result.Error != nil {
		log.Printf("[PenimbanganWorker] Gagal menutup sesi kedaluwarsa: %v", result.Error)
		return
	}

	if result.RowsAffected > 0 {
		log.Printf("[PenimbanganWorker] %d sesi penimbangan ditutup otomatis", result.RowsAffected)
	}
}
