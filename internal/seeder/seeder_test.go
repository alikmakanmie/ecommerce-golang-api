package seeder

import (
	"strings"
	"testing"
	"time"

	"ecommerce-api/internal/config"
	"ecommerce-api/internal/database"
	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/repository"
	"ecommerce-api/internal/service"
	"ecommerce-api/pkg/utils"

	"gorm.io/gorm"
)

func setupSeederDB(t *testing.T) *gorm.DB {
	t.Helper()
	cfg := &config.Config{
		AppEnv:           "test",
		DBDriver:         "sqlite",
		SQLitePath:       ":memory:",
		JWTAccessSecret:  "test-access-secret-min-32-chars-0123456789",
		JWTRefreshSecret: "test-refresh-secret-min-32-chars-0123456789",
		JWTAccessTTL:     15 * time.Minute,
		JWTRefreshTTL:    7 * 24 * time.Hour,
	}
	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestRun_SeedsSuperadminAndIsIdempotent(t *testing.T) {
	db := setupSeederDB(t)
	repo := repository.NewUserRepository(db)

	// Jalankan 2x: tidak boleh duplikat (user, produk, order demo).
	for i := 0; i < 2; i++ {
		if err := Run(db); err != nil {
			t.Fatalf("Run #%d: %v", i+1, err)
		}
	}
	var users int64
	db.Model(&domain.User{}).Count(&users)
	if users != 6 {
		t.Fatalf("total user: got %d, want 6 (superadmin+2 admin+3 customer)", users)
	}
	sa, err := repo.FindByEmail("superadmin@slaras.local")
	if err != nil {
		t.Fatalf("superadmin tidak ditemukan: %v", err)
	}
	if sa.Role != domain.RoleSuperAdmin || !sa.IsActive {
		t.Fatalf("superadmin salah: role=%s active=%v", sa.Role, sa.IsActive)
	}
	if !utils.CheckPasswordHash("SuperAdmin123!", sa.PasswordHash) {
		t.Fatal("password superadmin tidak cocok")
	}
	barista, err := repo.FindByEmail("barista@slaras.local")
	if err != nil || barista.Role != domain.RoleAdmin {
		t.Fatalf("barista hilang/salah role: %v %v", barista, err)
	}
	var cats int64
	db.Model(&domain.Category{}).Count(&cats)
	if cats != 5 {
		t.Fatalf("kategori: got %d, want 5", cats)
	}
	var prods int64
	db.Model(&domain.Product{}).Where("is_active = ?", true).Count(&prods)
	if prods != 18 {
		t.Fatalf("produk aktif: got %d, want 18", prods)
	}
	var withoutImg int64
	db.Model(&domain.Product{}).Where("is_active = ? AND (image_url = '' OR image_url IS NULL)", true).Count(&withoutImg)
	if withoutImg != 0 {
		t.Fatalf("%d produk aktif tanpa image_url", withoutImg)
	}
	var badHPP int64
	db.Model(&domain.Product{}).Where("is_active = ? AND (cost_price * 100 < price * 50 OR cost_price * 100 > price * 65)", true).Count(&badHPP)
	if badHPP != 0 {
		t.Fatalf("%d produk HPP di luar 50-65%% harga jual", badHPP)
	}
	var orders int64
	db.Model(&domain.Order{}).Count(&orders)
	if orders != 20 {
		t.Fatalf("order demo: got %d, want 20", orders)
	}
}

func TestRun_AddsSuperadminToExistingDB(t *testing.T) {
	// Simulasi DB VPS yang sudah di-seed versi lama (hanya admin+customer).
	db := setupSeederDB(t)
	repo := repository.NewUserRepository(db)
	for _, u := range []struct {
		email, pass string
		role        domain.Role
	}{
		{"admin@ecommerce.local", "Admin123!", domain.RoleAdmin},
		{"customer@ecommerce.local", "Customer123!", domain.RoleCustomer},
	} {
		hash, _ := utils.HashPassword(u.pass)
		if err := repo.Create(&domain.User{Name: string(u.role), Email: u.email, PasswordHash: hash, Role: u.role, IsActive: true}); err != nil {
			t.Fatalf("seed lama: %v", err)
		}
	}
	if err := Run(db); err != nil {
		t.Fatalf("Run: %v", err)
	}
	sa, err := repo.FindByEmail("superadmin@slaras.local")
	if err != nil {
		t.Fatalf("superadmin tidak dibuat di DB lama: %v", err)
	}
	if sa.Role != domain.RoleSuperAdmin {
		t.Fatalf("role superadmin salah: %s", sa.Role)
	}
	var count int64
	db.Model(&domain.User{}).Count(&count)
	if count != 6 {
		t.Fatalf("total user: got %d, want 6", count)
	}
	var orders int64
	db.Model(&domain.Order{}).Count(&orders)
	if orders != 20 {
		t.Fatalf("order demo: got %d, want 20", orders)
	}
}

func TestRun_DemoOrdersFillProfit(t *testing.T) {
	db := setupSeederDB(t)
	if err := Run(db); err != nil {
		t.Fatalf("Run: %v", err)
	}
	analytics := service.NewAnalyticsService(db)
	rep, err := analytics.GetProfitAnalytics("30d", "", "")
	if err != nil {
		t.Fatalf("profit: %v", err)
	}
	if rep.Summary.GrossRevenue <= 0 || rep.Summary.NetProfit <= 0 {
		t.Fatalf("ringkasan kosong: omzet=%d laba=%d", rep.Summary.GrossRevenue, rep.Summary.NetProfit)
	}
	if rep.Summary.TotalPaidOrders != 18 {
		t.Fatalf("order beromzet: got %d, want 18 (20 order - 2 pending)", rep.Summary.TotalPaidOrders)
	}
	daysWithRevenue := 0
	for _, d := range rep.DailySeries {
		if d.Revenue > 0 {
			daysWithRevenue++
		}
	}
	if daysWithRevenue != 14 {
		t.Fatalf("hari beromzet: got %d, want 14", daysWithRevenue)
	}
	if len(rep.TopProducts) == 0 {
		t.Fatal("top products kosong")
	}
	withProcessor := 0
	for _, tx := range rep.RecentTransactions {
		if tx.ProcessedByName != "" {
			withProcessor++
		}
	}
	if withProcessor == 0 {
		t.Fatal("recent transactions: tidak ada satu pun processed_by_name")
	}
	// Semua gambar produk demo memakai Unsplash CDN format yang diminta.
	var badImg int64
	db.Model(&domain.Product{}).Where("is_active = ? AND image_url NOT LIKE ?", true, "https://images.unsplash.com/photo-%").Count(&badImg)
	if badImg != 0 {
		t.Fatalf("%d produk image_url bukan Unsplash CDN", badImg)
	}
	var noQ80 int64
	db.Model(&domain.Product{}).Where("is_active = ? AND image_url NOT LIKE ?", true, "%auto=format&fit=crop&w=800&q=80").Count(&noQ80)
	if noQ80 != 0 {
		t.Fatalf("%d produk image_url tanpa parameter ?auto=format&fit=crop&w=800&q=80", noQ80)
	}
	if !strings.HasPrefix(rep.DailySeries[0].Date, "2026-") {
		t.Fatalf("format tanggal daily series aneh: %s", rep.DailySeries[0].Date)
	}
}
