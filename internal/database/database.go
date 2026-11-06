package database

import (
	"fmt"
	"strings"
	"time"

	"ecommerce-api/internal/config"
	"ecommerce-api/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch cfg.DBDriver {
	case "postgres":
		dialector = postgres.Open(cfg.PostgresDSN())
	case "sqlite":
		dsn := cfg.SQLitePath
		if !strings.Contains(dsn, "?") {
			dsn += "?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&cache=shared"
		}
		dialector = sqlite.Open(dsn)
	default:
		return nil, fmt.Errorf("driver tidak didukung: %s", cfg.DBDriver)
	}
	logLevel := logger.Silent
	if cfg.AppEnv == "development" {
		logLevel = logger.Warn
	}
	db, err := gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logLevel)})
	if err != nil {
		return nil, fmt.Errorf("gagal konek database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if cfg.DBDriver == "sqlite" {
		// WAL + busy timeout ditegakkan juga lewat PRAGMA agar berlaku
		// walau DSN suatu saat berubah; satu koneksi menghindari
		// "database is locked" pada driver sqlite.
		var mode string
		_ = db.Raw("PRAGMA journal_mode=WAL;").Scan(&mode).Error
		_ = db.Exec("PRAGMA busy_timeout = 5000;").Error
		if cfg.AppEnv == "development" {
			fmt.Printf("sqlite journal_mode: %s\n", mode)
		}
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetMaxOpenConns(1)
	} else {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
	}
	sqlDB.SetConnMaxLifetime(time.Hour)
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.User{},
		&domain.Address{},
		&domain.RefreshToken{},
		&domain.Category{},
		&domain.Product{},
		&domain.Cart{},
		&domain.CartItem{},
		&domain.Order{},
		&domain.OrderItem{},
		&domain.Payment{},
	)
}

// BackfillCostPrice mengisi harga modal produk lama yang belum punya
// cost_price (0) dengan 60% dari harga jual. Idempotent: hanya menyentuh
// baris yang cost_price-nya masih 0. Dipanggil sekali tiap startup
// sesudah AutoMigrate, jadi produk lama langsung terhitung profitnya.
func BackfillCostPrice(db *gorm.DB) error {
	return db.Model(&domain.Product{}).
		Where("cost_price = 0").
		UpdateColumn("cost_price", gorm.Expr("price * 60 / 100")).Error
}
