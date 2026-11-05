package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort  string
	Environment string
	// APIPublicURL: alamat publik backend (mis. https://svc.enviroo.tech),
	// dipakai untuk link install APK di email. Kosong = pakai URL R2 langsung.
	APIPublicURL string
	// CORSOrigins: origin web yang boleh memanggil API (env CORS_ORIGINS,
	// dipisah koma), mis. https://admin.enviroo.tech.
	CORSOrigins []string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string

	CFAccountID       string
	CFAccessKeyID     string
	CFSecretAccessKey string
	CFBucket          string
	CFPublicURL       string

	FCMCredentialsFile string

	SMTPHost       string
	SMTPPort       string
	SMTPEmail      string
	SMTPPassword   string
	SMTPSenderName string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, reading from environment variables")
	}

	return &Config{
		ServerPort:  getEnv("SERVER_PORT", ":8080"),
		Environment: getEnv("ENVIRONMENT", "development"),
		APIPublicURL: getEnv("API_PUBLIC_URL", ""),
		CORSOrigins:  parseDaftar(getEnv("CORS_ORIGINS", "http://localhost:5173")),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "postgres"),
		DBPassword:  getEnv("DB_PASSWORD", ""),
		DBName:      getEnv("DB_NAME", "enviroo"),
		DBSSLMode:   getEnv("DB_SSLMODE", "require"),
		CFAccountID:       getEnv("CF_ACCOUNT_ID", ""),
		CFAccessKeyID:     getEnv("CF_ACCESS_KEY_ID", ""),
		CFSecretAccessKey: getEnv("CF_SECRET_ACCESS_KEY", ""),
		CFBucket:          getEnv("CF_BUCKET", ""),
		CFPublicURL:       getEnv("CF_PUBLIC_URL", ""),
		SMTPHost:          getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:          getEnv("SMTP_PORT", "587"),
		SMTPEmail:         getEnv("SMTP_EMAIL", ""),
		SMTPPassword:      getEnv("SMTP_PASSWORD", ""),
		SMTPSenderName:     getEnv("SMTP_SENDER_NAME", "Enviroo"),
		FCMCredentialsFile: getEnv("FCM_CREDENTIALS_FILE", ""),
	}
}

// parseDaftar memecah nilai env "a, b,c" menjadi [a b c]; item kosong dibuang.
func parseDaftar(raw string) []string {
	hasil := []string{}
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			hasil = append(hasil, item)
		}
	}
	return hasil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
