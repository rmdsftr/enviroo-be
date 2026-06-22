package main

import (
	"context"
	"enviroo-be/internal/config"
	"enviroo-be/internal/database"
	"enviroo-be/internal/routes"
	"enviroo-be/internal/workers"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {

	cfg := config.LoadConfig()

	db, err := database.ConnectDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// ── Start background worker ──────────────────────────────────────────
	penarikanWorker := workers.NewPenarikanWorker(db, 1*time.Minute)
	penarikanWorker.Start()

	penimbanganWorker := workers.NewPenimbanganWorker(db)
	penimbanganWorker.Start()

	cfStorage, err := storage.NewCloudflareStorage(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Cloudflare Storage: %v", err)
	}

	mailer := utils.NewMailer(cfg)

	var fcmClient *utils.FCMClient
	if cfg.FCMCredentialsFile != "" {
		fc, err := utils.NewFCMClient(context.Background(), cfg.FCMCredentialsFile)
		if err != nil {
			log.Printf("Warning: gagal inisialisasi FCM client: %v", err)
		} else {
			fcmClient = fc
			log.Println("FCM client berhasil diinisialisasi")
		}
	} else {
		log.Println("Warning: FCM_CREDENTIALS_FILE tidak diset, push notification dinonaktifkan")
	}

	// ── Start reminder worker (kirim notif jadwal pukul 07:00 setiap hari) ──
	reminderWorker := workers.NewReminderWorker(db, fcmClient, 18)
	reminderWorker.Start()

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "https://enviroo.tech", "https://www.enviroo.tech"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	routes.SetupRoutes(r, db, cfStorage, mailer, fcmClient)

	// ── Run Server in Goroutine ─────────────────────────────────────────
	go func() {
		log.Printf("Server starting on %s", cfg.ServerPort)
		if err := r.Run(cfg.ServerPort); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// ── Graceful Shutdown ────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	penarikanWorker.Stop()
	penimbanganWorker.Stop()
	reminderWorker.Stop()
	log.Println("Server stopped gracefully.")
}
