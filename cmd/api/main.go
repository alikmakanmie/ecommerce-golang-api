package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ecommerce-api/internal/config"
	"ecommerce-api/internal/database"
	"ecommerce-api/internal/router"
	"ecommerce-api/internal/seeder"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("migrate error: %v", err)
	}
	if err := database.BackfillCostPrice(db); err != nil {
		log.Fatalf("backfill cost_price error: %v", err)
	}
	log.Println("database connected & migrated")
	if cfg.SeedDB {
		if err := seeder.Run(db); err != nil {
			log.Fatalf("seeder error: %v", err)
		}
		log.Println("seeder selesai")
	}
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	if cfg.AppEnv != "production" && cfg.WebhookSecret == "" {
		log.Println("peringatan: WEBHOOK_SECRET kosong — semua panggilan webhook akan ditolak (401)")
	}
	r := router.New(cfg, db)
	srv := &http.Server{
		Addr:           ":" + cfg.AppPort,
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		log.Printf("server listening on :%s (env=%s, db=%s)", cfg.AppPort, cfg.AppEnv, cfg.DBDriver)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("server stopped")
}
