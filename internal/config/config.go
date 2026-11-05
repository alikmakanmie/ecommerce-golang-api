package config

import (
	"fmt"
	"os"
	"strconv"
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
	SeedDB             bool
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
		SeedDB:             getEnvBool("SEED_DB", false),
	}
	if len(cfg.JWTAccessSecret) < 32 || len(cfg.JWTRefreshSecret) < 32 {
		return nil, fmt.Errorf("JWT_ACCESS_SECRET dan JWT_REFRESH_SECRET wajib minimal 32 karakter")
	}
	if cfg.DBDriver != "postgres" && cfg.DBDriver != "sqlite" {
		return nil, fmt.Errorf("DB_DRIVER harus postgres atau sqlite, dapat: %s", cfg.DBDriver)
	}
	return cfg, nil
}

func (c *Config) PostgresDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta",
		c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort, c.DBSSLMode)
}
