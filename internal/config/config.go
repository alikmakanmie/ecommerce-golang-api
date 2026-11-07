package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv             string
	AppPort            string
	DBDriver           string
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	DBSSLMode          string
	SQLitePath         string
	DatabaseURL        string
	JWTAccessSecret    string
	JWTRefreshSecret   string
	JWTAccessTTL       time.Duration
	JWTRefreshTTL      time.Duration
	PaymentMockBaseURL string
	WebhookSecret      string
	CORSAllowedOrigins []string
	SeedDB             bool
	// URL publik API (untuk membangun URL absolut hasil upload).
	PublicBaseURL string
	// Rate limit global API per IP (request/detik + burst). Default
	// 20/40; saat benchmark dibuka lebar via env, lalu dikembalikan.
	PublicRateLimit float64
	PublicBurst     int
	// Batas upload serentak (semaphore POST /upload). Default 20.
	UploadMaxConcurrent int
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	accessMin, err := strconv.Atoi(getEnv("JWT_ACCESS_TTL_MINUTES", "15"))
	if err != nil || accessMin <= 0 {
		return nil, fmt.Errorf("JWT_ACCESS_TTL_MINUTES tidak valid")
	}
	refreshDays, err := strconv.Atoi(getEnv("JWT_REFRESH_TTL_DAYS", "7"))
	if err != nil || refreshDays <= 0 {
		return nil, fmt.Errorf("JWT_REFRESH_TTL_DAYS tidak valid")
	}
	rateLimit, err := strconv.ParseFloat(getEnv("PUBLIC_RATE_LIMIT", "20"), 64)
	if err != nil || rateLimit <= 0 {
		return nil, fmt.Errorf("PUBLIC_RATE_LIMIT tidak valid")
	}
	burst, err := strconv.Atoi(getEnv("PUBLIC_BURST", "40"))
	if err != nil || burst <= 0 {
		return nil, fmt.Errorf("PUBLIC_BURST tidak valid")
	}
	uploadMax, err := strconv.Atoi(getEnv("UPLOAD_MAX_CONCURRENT", "20"))
	if err != nil || uploadMax <= 0 {
		return nil, fmt.Errorf("UPLOAD_MAX_CONCURRENT tidak valid")
	}
	cfg := &Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		AppPort:            getEnv("APP_PORT", "8080"),
		DBDriver:           getEnv("DB_DRIVER", "postgres"),
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "5432"),
		DBUser:             getEnv("DB_USER", "ecommerce"),
		DBPassword:         getEnv("DB_PASSWORD", "ecommerce123"),
		DBName:             getEnv("DB_NAME", "ecommerce_db"),
		DBSSLMode:          getEnv("DB_SSLMODE", "disable"),
		SQLitePath:         getEnv("SQLITE_PATH", "./ecommerce.db"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		JWTAccessSecret:    os.Getenv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret:   os.Getenv("JWT_REFRESH_SECRET"),
		JWTAccessTTL:       time.Duration(accessMin) * time.Minute,
		JWTRefreshTTL:      time.Duration(refreshDays) * 24 * time.Hour,
		PaymentMockBaseURL: getEnv("PAYMENT_MOCK_BASE_URL", "http://localhost:8080"),
		WebhookSecret:      os.Getenv("WEBHOOK_SECRET"),
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:8080")),
		SeedDB:             getEnvBool("SEED_DB", false),
		PublicBaseURL:      getEnv("PUBLIC_BASE_URL", "https://api.alikmakanmie.my.id"),
		PublicRateLimit:    rateLimit,
		PublicBurst:        burst,
		UploadMaxConcurrent: uploadMax,
	}
	if len(cfg.JWTAccessSecret) < 32 || len(cfg.JWTRefreshSecret) < 32 {
		return nil, fmt.Errorf("JWT_ACCESS_SECRET dan JWT_REFRESH_SECRET wajib minimal 32 karakter")
	}
	if cfg.WebhookSecret != "" && len(cfg.WebhookSecret) < 32 {
		return nil, fmt.Errorf("WEBHOOK_SECRET wajib minimal 32 karakter bila di-set")
	}
	if cfg.AppEnv == "production" && cfg.WebhookSecret == "" {
		return nil, fmt.Errorf("WEBHOOK_SECRET wajib di-set saat APP_ENV=production (webhook tanpa signature akan selalu ditolak)")
	}
	if cfg.DBDriver != "postgres" && cfg.DBDriver != "sqlite" {
		return nil, fmt.Errorf("DB_DRIVER harus postgres atau sqlite, dapat: %s", cfg.DBDriver)
	}
	return cfg, nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (c *Config) PostgresDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta",
		c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort, c.DBSSLMode)
}
