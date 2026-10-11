package workers

import (
	"context"
	"enviroo-be/internal/adapters/fcm"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"enviroo-be/internal/utils"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

type ReminderWorker struct {
	DB           *gorm.DB
	NotifSvc     services.NotifikasiService
	SendHour     int
	stopCh       chan struct{}
	lastSentDate string
}

func NewReminderWorker(db *gorm.DB, fcmClient *fcm.Client, sendHour int) *ReminderWorker {
	notifRepo := repositories.NewNotifikasiRepository(db)
	notifSvc := services.NewNotifikasiService(notifRepo, fcmClient)
	return &ReminderWorker{
		DB:       db,
		NotifSvc: notifSvc,
		SendHour: sendHour,
		stopCh:   make(chan struct{}),
	}
}

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

func (w *ReminderWorker) Stop() {
	close(w.stopCh)
}

func (w *ReminderWorker) kirimReminderHariIni(now time.Time) {
	todayHari := utils.HariOf(now)
	todayMingguKe := utils.WeekOfMonth(now)
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
			if jadwal.JenisJadwal == models.JadwalPengangkutan {
				w.kirimReminderPengangkutan(jadwal, false)
			}
		}
	}

	tomorrow := now.AddDate(0, 0, 1)
	tomorrowHari := utils.HariOf(tomorrow)
	tomorrowMingguKe := utils.WeekOfMonth(tomorrow)
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
			if jadwal.JenisJadwal == models.JadwalPengangkutan {
				w.kirimReminderPengangkutan(jadwal, true)
			}
		}
	}
}

func (w *ReminderWorker) kirimReminderPengangkutan(jadwal models.Jadwal, isH1 bool) {
	if jadwal.TargetBankID == nil {
		log.Printf("[ReminderWorker] Jadwal pengangkutan %s tidak memiliki target_bank_id, skip", jadwal.JadwalID)
		return
	}
	targetBankID := *jadwal.TargetBankID

	var bsi, bsu models.BankSampah
	if err := w.DB.Where("bank_id = ?", jadwal.BankID).First(&bsi).Error; err != nil {
		log.Printf("[ReminderWorker] BSI tidak ditemukan untuk jadwal %s: %v", jadwal.JadwalID, err)
		return
	}
	if err := w.DB.Where("bank_id = ?", targetBankID).First(&bsu).Error; err != nil {
		log.Printf("[ReminderWorker] BSU tidak ditemukan untuk jadwal %s: %v", jadwal.JadwalID, err)
		return
	}

	type adminUser struct {
		UserID   string
		FCMToken string
	}

	var targets []adminUser
	if err := w.DB.Table("admin").
		Select("users.user_id, users.fcm_token").
		Joins("JOIN users ON users.user_id = admin.user_id").
		Where("admin.bank_id IN ? AND admin.status_admin = ?",
			[]string{jadwal.BankID, targetBankID}, models.Aktif).
		Scan(&targets).Error; err != nil {
		log.Printf("[ReminderWorker] Gagal query admin untuk jadwal %s: %v", jadwal.JadwalID, err)
		return
	}

	refID := jadwal.JadwalID
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
