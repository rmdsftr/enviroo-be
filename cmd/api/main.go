package main

import (
	"context"
	"enviroo-be/internal/adapters/fcm"
	"enviroo-be/internal/adapters/mailer"
	"enviroo-be/internal/adapters/storage"
	"enviroo-be/internal/config"
	"enviroo-be/internal/database"
	"enviroo-be/internal/middleware"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/routes"
	"enviroo-be/internal/services"
	"enviroo-be/internal/workers"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

func main() {
	cfg := config.LoadConfig()

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 ||
		jwtSecret == "enviroo-jwt-secret-ganti-di-production" ||
		jwtSecret == "enviroo-secret-dev-fallback-change-me" {
		log.Fatal("JWT_SECRET tidak diset, terlalu pendek, atau masih memakai nilai default yang lemah. " +
			"Set JWT_SECRET yang kuat (mis. `openssl rand -base64 48`) di environment.")
	}

	db, err := database.ConnectDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	cfStorage, err := storage.NewCloudflareStorage(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Cloudflare Storage: %v", err)
	}

	mail := mailer.New(cfg)

	var fcmClient *fcm.Client
	if cfg.FCMCredentialsFile != "" {
		fc, err := fcm.New(context.Background(), cfg.FCMCredentialsFile)
		if err != nil {
			log.Printf("Warning: gagal inisialisasi FCM client: %v", err)
		} else {
			fcmClient = fc
			log.Println("FCM client berhasil diinisialisasi")
		}
	} else {
		log.Println("Warning: FCM_CREDENTIALS_FILE tidak diset, push notification dinonaktifkan")
	}

	notifSvcWorker := services.NewNotifikasiService(repositories.NewNotifikasiRepository(db), fcmClient)
	penimbanganSvc := services.NewPenimbanganService(db, repositories.NewPenimbanganRepo(db), notifSvcWorker)
	penimbanganWorker := workers.NewPenimbanganWorker(penimbanganSvc)
	penimbanganWorker.Start()

	reminderWorker := workers.NewReminderWorker(db, fcmClient, 18)
	reminderWorker.Start()

	penarikanWorker := workers.NewPenarikanWorker(db, fcmClient, 1*time.Minute)
	penarikanWorker.Start()

	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	if err := r.SetTrustedProxies([]string{"127.0.0.1", "172.16.0.0/12"}); err != nil {
		log.Printf("Warning: gagal set trusted proxies: %v", err)
	}

	rdb, rateStore := bangunRateLimitStore(cfg)

	globalIP := middleware.NewIPRateLimiter("glob_ip", rate.Every(300*time.Millisecond), 100).WithStore(rateStore)
	globalUser := middleware.NewIPRateLimiter("glob_usr", rate.Every(600*time.Millisecond), 50).WithStore(rateStore)
	r.Use(globalIP.Middleware(middleware.DimIP()))
	r.Use(globalUser.Middleware(middleware.DimUser()))

	if len(cfg.CORSOrigins) == 0 {
		log.Fatal("CORS_ORIGINS kosong. Isi dengan origin web admin, mis. https://admin.enviroo.tech")
	}
	log.Printf("CORS origins: %v", cfg.CORSOrigins)

	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	routes.SetupRoutes(r, db, cfStorage, mail, fcmClient, cfg.APIPublicURL, rateStore)

	go func() {
		log.Printf("Server starting on %s", cfg.ServerPort)
		if err := r.Run(cfg.ServerPort); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	penarikanWorker.Stop()
	penimbanganWorker.Stop()
	reminderWorker.Stop()
	if rdb != nil {
		_ = rdb.Close()
	}
	log.Println("Server stopped gracefully.")
}

func bangunRateLimitStore(cfg *config.Config) (*redis.Client, middleware.Penyimpanan) {
	memori := middleware.NewStoreMemori()
	if cfg.RedisAddr == "" {
		log.Println("Warning: REDIS_ADDR tidak diset, rate limiter memakai in-memory (tidak terbagi antar instance)")
		return nil, memori
	}

	rdb := middleware.NewClientRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	store := middleware.NewStoreRedis(rdb, cfg.RedisTimeout)
	if err := store.Ping(); err != nil {
		log.Printf("Warning: Redis di %s tidak bisa dihubungi (%v). Rate limiter jalan in-memory dan akan mencoba Redis lagi otomatis.", cfg.RedisAddr, err)
	} else {
		log.Printf("Rate limiter: backend Redis (%s)", cfg.RedisAddr)
	}

	return rdb, middleware.NewStoreFallback(store, memori)
}
