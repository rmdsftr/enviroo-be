package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort  string
	Environment string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string

	// Cloudflare
	CFAccountID       string
	CFAccessKeyID     string
	CFSecretAccessKey string
	CFBucket          string
	CFPublicURL       string

	// SMTP Email
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
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "postgres"),
		DBPassword:  getEnv("DB_PASSWORD", "behelhijau"),
		DBName:      getEnv("DB_NAME", "enviroo"),
		CFAccountID:       getEnv("CF_ACCOUNT_ID", ""),
		CFAccessKeyID:     getEnv("CF_ACCESS_KEY_ID", ""),
		CFSecretAccessKey: getEnv("CF_SECRET_ACCESS_KEY", ""),
		CFBucket:          getEnv("CF_BUCKET", ""),
		CFPublicURL:       getEnv("CF_PUBLIC_URL", ""),
		SMTPHost:          getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:          getEnv("SMTP_PORT", "587"),
		SMTPEmail:         getEnv("SMTP_EMAIL", ""),
		SMTPPassword:      getEnv("SMTP_PASSWORD", ""),
		SMTPSenderName:    getEnv("SMTP_SENDER_NAME", "Enviroo"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
