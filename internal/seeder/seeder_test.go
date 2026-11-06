package seeder

import (
	"testing"
	"time"

	"ecommerce-api/internal/config"
	"ecommerce-api/internal/database"
	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/repository"
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

	// Jalankan 2x: tidak boleh duplikat.
	for i := 0; i < 2; i++ {
		if err := Run(db); err != nil {
			t.Fatalf("Run #%d: %v", i+1, err)
		}
	}
	var count int64
	if err := db.Model(&domain.User{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Fatalf("total user: got %d, want 3 (superadmin+admin+customer)", count)
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
	admin, err := repo.FindByEmail("admin@ecommerce.local")
	if err != nil || admin.Role != domain.RoleAdmin {
		t.Fatalf("admin toko hilang/salah role: %v %v", admin, err)
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
	if count != 3 {
		t.Fatalf("total user: got %d, want 3", count)
	}
}
