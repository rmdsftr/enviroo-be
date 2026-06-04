package workers

import (
	"context"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"enviroo-be/pkg/utils"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// ReminderWorker mengirim push notification reminder jadwal penimbangan
// dan pengangkutan satu kali per hari kepada nasabah dan admin/petugas
// yang terkait.
type ReminderWorker struct {
	DB       *gorm.DB
	NotifSvc services.NotifikasiService
	// Jam (WIB 24h) kapan reminder dikirim setiap harinya, misalnya 7 = pukul 07:00
	SendHour int
	stopCh   chan struct{}
	// Mencegah double-send dalam satu hari yang sama
	lastSentDate string
}

func NewReminderWorker(db *gorm.DB, fcmClient *utils.FCMClient, sendHour int) *ReminderWorker {
	notifRepo := repositories.NewNotifikasiRepository(db)
	notifSvc := services.NewNotifikasiService(notifRepo, fcmClient)
	return &ReminderWorker{
		DB:       db,
		NotifSvc: notifSvc,
		SendHour: sendHour,
		stopCh:   make(chan struct{}),
	}
}

// Start menjalankan worker di goroutine terpisah.
// Worker melakukan tick setiap menit untuk memastikan reminder
// dikirim tepat pada jam yang ditentukan.
func (w *ReminderWorker) Start() {
	go func() {
		log.Println("[ReminderWorker] Worker started")
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				now := time.Now()
				todayStr := now.Format("2006-01-02")
				if now.Hour() == w.SendHour && w.lastSentDate != todayStr {
					log.Printf("[ReminderWorker] Mengirim reminder jadwal untuk tanggal %s", todayStr)
					w.kirimReminderHariIni(now)
					w.lastSentDate = todayStr
				}
			case <-w.stopCh:
				log.Println("[ReminderWorker] Worker stopped")
				return
			}
		}
	}()
}

// Stop menghentikan worker secara graceful.
func (w *ReminderWorker) Stop() {
	close(w.stopCh)
}

// kirimReminderHariIni mencari semua jadwal aktif yang berlaku hari ini
// dan besok, lalu mendistribusikan notifikasi ke penerima yang sesuai.
func (w *ReminderWorker) kirimReminderHariIni(now time.Time) {
	weekdays := []models.HariEnum{
		models.Minggu, models.Senin, models.Selasa, models.Rabu,
		models.Kamis, models.Jumat, models.Sabtu,
	}

	// ── H-0: Jadwal hari ini ──────────────────────────────────────────────
	todayHari := weekdays[now.Weekday()]
	todayMingguKe := getWeekOfMonthWorker(now)
	todayStr := now.Format("2006-01-02")

	var jadwalsHariIni []models.Jadwal
	if err := w.DB.
		Where(
			"(is_rutin = true AND is_active = true AND hari = ? AND minggu_ke = ?) OR (is_rutin = false AND is_active = true AND DATE(tanggal) = ?)",
			todayHari, todayMingguKe, todayStr,
		).
		Find(&jadwalsHariIni).Error; err != nil {
		log.Printf("[ReminderWorker] Gagal query jadwal H-0: %v", err)
	} else if len(jadwalsHariIni) > 0 {
		log.Printf("[ReminderWorker] H-0: Ditemukan %d jadwal aktif, mengirim reminder hari ini...", len(jadwalsHariIni))
		for _, jadwal := range jadwalsHariIni {
			switch jadwal.JenisJadwal {
			case models.JadwalPenimbangan:
				w.kirimReminderPenimbangan(jadwal, false)
			case models.JadwalPengangkutan:
				w.kirimReminderPengangkutan(jadwal, false)
			}
		}
	}

	// ── H-1: Jadwal besok ─────────────────────────────────────────────────
	tomorrow := now.AddDate(0, 0, 1)
	tomorrowHari := weekdays[tomorrow.Weekday()]
	tomorrowMingguKe := getWeekOfMonthWorker(tomorrow)
	tomorrowStr := tomorrow.Format("2006-01-02")

	var jadwalsH1 []models.Jadwal
	if err := w.DB.
		Where(
			"(is_rutin = true AND is_active = true AND hari = ? AND minggu_ke = ?) OR (is_rutin = false AND is_active = true AND DATE(tanggal) = ?)",
			tomorrowHari, tomorrowMingguKe, tomorrowStr,
		).
		Find(&jadwalsH1).Error; err != nil {
		log.Printf("[ReminderWorker] Gagal query jadwal H-1: %v", err)
	} else if len(jadwalsH1) > 0 {
		log.Printf("[ReminderWorker] H-1: Ditemukan %d jadwal untuk besok, mengirim reminder...", len(jadwalsH1))
		for _, jadwal := range jadwalsH1 {
			switch jadwal.JenisJadwal {
			case models.JadwalPenimbangan:
				w.kirimReminderPenimbangan(jadwal, true)
			case models.JadwalPengangkutan:
				w.kirimReminderPengangkutan(jadwal, true)
			}
		}
	}
}

// kirimReminderPenimbangan mengirim reminder ke semua nasabah aktif
// yang terdaftar di bank penyelenggara jadwal penimbangan.
// isH1=true kirim notif "besok", isH1=false kirim notif "hari ini".
func (w *ReminderWorker) kirimReminderPenimbangan(jadwal models.Jadwal, isH1 bool) {
	// Ambil nama bank
	var bank models.BankSampah
	if err := w.DB.Where("bank_id = ?", jadwal.BankID).First(&bank).Error; err != nil {
		log.Printf("[ReminderWorker] Bank tidak ditemukan untuk jadwal %s: %v", jadwal.JadwalID, err)
		return
	}

	// Ambil semua nasabah aktif di bank tersebut beserta data user-nya
	type nasabahUser struct {
		UserID   string
		FCMToken string
	}
	var targets []nasabahUser
	if err := w.DB.Table("nasabah").
		Select("users.user_id, users.fcm_token").
		Joins("JOIN users ON users.user_id = nasabah.user_id").
		Where("nasabah.bank_id = ? AND nasabah.status_nasabah = ?", jadwal.BankID, models.Aktif).
		Scan(&targets).Error; err != nil {
		log.Printf("[ReminderWorker] Gagal query nasabah untuk jadwal %s: %v", jadwal.JadwalID, err)
		return
	}

	refID := jadwal.JadwalID.String()
	sent := 0
	for _, t := range targets {
		var err error
		if isH1 {
			err = w.NotifSvc.NotifReminderPenimbanganH1(
				context.Background(), t.UserID, t.FCMToken,
				bank.NamaBank, jadwal.JamMulai, jadwal.JamSelesai, refID,
			)
		} else {
			err = w.NotifSvc.NotifReminderPenimbangan(
				context.Background(), t.UserID, t.FCMToken,
				bank.NamaBank, jadwal.JamMulai, jadwal.JamSelesai, refID,
			)
		}
		if err != nil {
			fmt.Printf("[ReminderWorker] Gagal kirim reminder penimbangan ke user %s: %v\n", t.UserID, err)
		} else {
			sent++
		}
	}
	label := "hari ini"
	if isH1 {
		label = "besok (H-1)"
	}
	log.Printf("[ReminderWorker] Reminder penimbangan %s jadwal %s dikirim ke %d nasabah", label, jadwal.JadwalID, sent)
}

// kirimReminderPengangkutan mengirim reminder ke semua admin/petugas aktif
// dari BSU (target) maupun BSI (inisiator) jadwal pengangkutan.
// isH1=true kirim notif "besok", isH1=false kirim notif "hari ini".
func (w *ReminderWorker) kirimReminderPengangkutan(jadwal models.Jadwal, isH1 bool) {
	if jadwal.TargetBankID == "" {
		log.Printf("[ReminderWorker] Jadwal pengangkutan %s tidak memiliki target_bank_id, skip", jadwal.JadwalID)
		return
	}

	// Ambil nama BSI (inisiator) dan BSU (target)
	var bsi, bsu models.BankSampah
	if err := w.DB.Where("bank_id = ?", jadwal.BankID).First(&bsi).Error; err != nil {
		log.Printf("[ReminderWorker] BSI tidak ditemukan untuk jadwal %s: %v", jadwal.JadwalID, err)
		return
	}
	if err := w.DB.Where("bank_id = ?", jadwal.TargetBankID).First(&bsu).Error; err != nil {
		log.Printf("[ReminderWorker] BSU tidak ditemukan untuk jadwal %s: %v", jadwal.JadwalID, err)
		return
	}

	// Query admin/petugas aktif dari kedua bank
	type adminUser struct {
		UserID   string
		FCMToken string
	}

	var targets []adminUser
	if err := w.DB.Table("admin").
		Select("users.user_id, users.fcm_token").
		Joins("JOIN users ON users.user_id = admin.user_id").
		Where("admin.bank_id IN ? AND admin.status_admin = ?",
			[]string{jadwal.BankID, jadwal.TargetBankID}, models.Aktif).
		Scan(&targets).Error; err != nil {
		log.Printf("[ReminderWorker] Gagal query admin untuk jadwal %s: %v", jadwal.JadwalID, err)
		return
	}

	refID := jadwal.JadwalID.String()
	sent := 0
	for _, t := range targets {
		var err error
		if isH1 {
			err = w.NotifSvc.NotifReminderPengangkutanH1(
				context.Background(), t.UserID, t.FCMToken,
				bsu.NamaBank, bsi.NamaBank, jadwal.JamMulai, jadwal.JamSelesai, refID,
			)
		} else {
			err = w.NotifSvc.NotifReminderPengangkutan(
				context.Background(), t.UserID, t.FCMToken,
				bsu.NamaBank, bsi.NamaBank, jadwal.JamMulai, jadwal.JamSelesai, refID,
			)
		}
		if err != nil {
			fmt.Printf("[ReminderWorker] Gagal kirim reminder pengangkutan ke user %s: %v\n", t.UserID, err)
		} else {
			sent++
		}
	}
	label := "hari ini"
	if isH1 {
		label = "besok (H-1)"
	}
	log.Printf("[ReminderWorker] Reminder pengangkutan %s jadwal %s dikirim ke %d admin/petugas", label, jadwal.JadwalID, sent)
}

// getWeekOfMonthWorker menghitung minggu ke-berapa dalam bulan untuk tanggal tertentu.
// (duplikasi dari jadwal_controller agar workers tidak bergantung pada package controllers)
func getWeekOfMonthWorker(date time.Time) int {
	firstOfMonth := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
	firstWeekday := int(firstOfMonth.Weekday())
	adjustedDay := date.Day() + firstWeekday - 1
	return (adjustedDay / 7) + 1
}
