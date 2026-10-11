package workers

import (
	"log"
	"time"

	"enviroo-be/internal/services"
)

type PenimbanganWorker struct {
	Svc       services.PenimbanganService
	stopCh    chan struct{}
	lastSweep time.Time
}

func NewPenimbanganWorker(svc services.PenimbanganService) *PenimbanganWorker {
	return &PenimbanganWorker{
		Svc:    svc,
		stopCh: make(chan struct{}),
	}
}

func (w *PenimbanganWorker) Start() {
	go func() {
		log.Println("[PenimbanganWorker] Worker started")
		expiryTicker := time.NewTicker(30 * time.Minute)
		defer expiryTicker.Stop()

		pendingTicker := time.NewTicker(1 * time.Minute)
		defer pendingTicker.Stop()

		const jedaSapuan = 10 * time.Minute

		jamBuatSesi := w.Svc.JamBuatSesiPending()

		for {
			select {
			case <-expiryTicker.C:
				_, _ = w.Svc.TutupSesiTerlewat()

			case <-pendingTicker.C:
				now := time.Now()
				if now.Hour() < jamBuatSesi || now.Sub(w.lastSweep) < jedaSapuan {
					continue
				}

				dibuat, sukses := w.Svc.BuatSesiPendingBesok(now)
				if sukses {
					w.lastSweep = now
				} else {
					log.Printf("[PenimbanganWorker] Ada kegagalan saat bikin sesi pending, bakal dicoba lagi menit berikutnya")
				}
				if dibuat > 0 {
					log.Printf("[PenimbanganWorker] %d sesi pending H-1 dibuat (%s)", dibuat, now.Format("2006-01-02 15:04"))
				}

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
