package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ecommerce-api/internal/config"
	"ecommerce-api/internal/database"
	"ecommerce-api/internal/domain"
	"ecommerce-api/internal/repository"
	"ecommerce-api/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func setupRBACRouter(t *testing.T) (*gin.Engine, map[domain.Role]*domain.User) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		AppEnv:             "test",
		AppPort:            "8080",
		DBDriver:           "sqlite",
		SQLitePath:         ":memory:",
		JWTAccessSecret:    "test-access-secret-min-32-chars-0123456789",
		JWTRefreshSecret:   "test-refresh-secret-min-32-chars-0123456789",
		JWTAccessTTL:       15 * time.Minute,
		JWTRefreshTTL:      7 * 24 * time.Hour,
		PaymentMockBaseURL: "http://localhost:8080",
		CORSAllowedOrigins: []string{"*"},
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
	repo := repository.NewUserRepository(db)
	users := map[domain.Role]*domain.User{}
	for role, email := range map[domain.Role]string{
		domain.RoleSuperAdmin: "superadmin@slaras.local",
		domain.RoleAdmin:      "admin@ecommerce.local",
		domain.RoleCustomer:   "customer@ecommerce.local",
	} {
		hash, _ := utils.HashPassword("Password123!")
		u := &domain.User{Name: string(role), Email: email, PasswordHash: hash, Role: role, IsActive: true}
		if err := repo.Create(u); err != nil {
			t.Fatalf("seed %s: %v", role, err)
		}
		users[role] = u
	}
	return New(cfg, db), users
}

func tokenFor(t *testing.T, u *domain.User) string {
	t.Helper()
	tok, err := utils.GenerateToken(u.ID, u.Email, u.Role, "test-access-secret-min-32-chars-0123456789", 15*time.Minute)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	return tok
}

func doReq(t *testing.T, r *gin.Engine, method, path string, body any, tok string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeData(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var env map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode %d: %v\n%s", w.Code, err, w.Body.String())
	}
	data, _ := env["data"].(map[string]any)
	return data
}

func TestSuperadminUsers_RBAC(t *testing.T) {
	r, users := setupRBACRouter(t)
	saTok, adminTok, custTok := tokenFor(t, users[domain.RoleSuperAdmin]), tokenFor(t, users[domain.RoleAdmin]), tokenFor(t, users[domain.RoleCustomer])

	// Tanpa token -> 401; customer/admin -> 403; superadmin -> 200.
	if w := doReq(t, r, "GET", "/api/v1/superadmin/users", nil, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa token: got %d, want 401", w.Code)
	}
	for name, tok := range map[string]string{"admin": adminTok, "customer": custTok} {
		if w := doReq(t, r, "GET", "/api/v1/superadmin/users", nil, tok); w.Code != http.StatusForbidden {
			t.Fatalf("%s GET /superadmin/users: got %d, want 403", name, w.Code)
		}
		if w := doReq(t, r, "POST", "/api/v1/superadmin/users", map[string]any{"name": "X", "email": "x@y.z", "password": "Password123!", "role": "admin"}, tok); w.Code != http.StatusForbidden {
			t.Fatalf("%s POST /superadmin/users: got %d, want 403", name, w.Code)
		}
	}
	w := doReq(t, r, "GET", "/api/v1/superadmin/users", nil, saTok)
	if w.Code != http.StatusOK {
		t.Fatalf("superadmin GET: got %d, want 200: %s", w.Code, w.Body.String())
	}
	var list struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Data) != 3 {
		t.Fatalf("list: got %d user, want 3", len(list.Data))
	}

	// Stats: superadmin 200, admin 403.
	if w := doReq(t, r, "GET", "/api/v1/superadmin/users/stats", nil, saTok); w.Code != http.StatusOK {
		t.Fatalf("stats superadmin: got %d, want 200", w.Code)
	} else {
		stats := decodeData(t, w)
		if stats["total"] != float64(3) || stats["superadmin"] != float64(1) || stats["admin"] != float64(1) {
			t.Fatalf("stats salah: %v", stats)
		}
	}
	if w := doReq(t, r, "GET", "/api/v1/superadmin/users/stats", nil, adminTok); w.Code != http.StatusForbidden {
		t.Fatalf("stats admin: got %d, want 403", w.Code)
	}

	// Buat admin toko baru sebagai superadmin -> 201.
	w = doReq(t, r, "POST", "/api/v1/superadmin/users", map[string]any{
		"name": "Toko Baru", "email": "toko@slaras.local", "password": "Password123!", "phone": "0811", "role": "admin",
	}, saTok)
	if w.Code != http.StatusCreated {
		t.Fatalf("create user: got %d, want 201: %s", w.Code, w.Body.String())
	}
	newUser := decodeData(t, w)
	newID, _ := uuid.Parse(newUser["id"].(string))

	// Filter role=admin -> 2 admin; search -> 1.
	w = doReq(t, r, "GET", "/api/v1/superadmin/users?role=admin", nil, saTok)
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil || len(list.Data) != 2 {
		t.Fatalf("filter role=admin: got %d, want 2", len(list.Data))
	}
	w = doReq(t, r, "GET", "/api/v1/superadmin/users?search=toko@slaras", nil, saTok)
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil || len(list.Data) != 1 {
		t.Fatalf("search: got %d, want 1", len(list.Data))
	}

	// Ubah role admin baru -> customer (200); superadmin turunkan role sendiri -> 403.
	w = doReq(t, r, "PUT", fmt.Sprintf("/api/v1/superadmin/users/%s/role", newID), map[string]any{"role": "customer"}, saTok)
	if w.Code != http.StatusOK {
		t.Fatalf("update role: got %d, want 200: %s", w.Code, w.Body.String())
	}
	if w := doReq(t, r, "PUT", fmt.Sprintf("/api/v1/superadmin/users/%s/role", users[domain.RoleSuperAdmin].ID), map[string]any{"role": "admin"}, saTok); w.Code != http.StatusForbidden {
		t.Fatalf("self-demote: got %d, want 403", w.Code)
	}

	// Nonaktifkan customer -> 200; login customer ditolak; aktifkan lagi -> 200.
	custID := users[domain.RoleCustomer].ID
	w = doReq(t, r, "PUT", fmt.Sprintf("/api/v1/superadmin/users/%s/status", custID), map[string]any{"is_active": false}, saTok)
	if w.Code != http.StatusOK {
		t.Fatalf("deactivate: got %d, want 200: %s", w.Code, w.Body.String())
	}
	w = doReq(t, r, "POST", "/api/v1/auth/login", map[string]any{"email": "customer@ecommerce.local", "password": "Password123!"}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("login akun nonaktif: got %d, want 401", w.Code)
	}
	w = doReq(t, r, "PUT", fmt.Sprintf("/api/v1/superadmin/users/%s/status", custID), map[string]any{"is_active": true}, saTok)
	if w.Code != http.StatusOK {
		t.Fatalf("reactivate: got %d, want 200", w.Code)
	}

	// Hapus admin baru -> 200; hapus diri sendiri -> 403.
	w = doReq(t, r, "DELETE", fmt.Sprintf("/api/v1/superadmin/users/%s", newID), nil, saTok)
	if w.Code != http.StatusOK {
		t.Fatalf("delete user: got %d, want 200: %s", w.Code, w.Body.String())
	}
	if w := doReq(t, r, "DELETE", fmt.Sprintf("/api/v1/superadmin/users/%s", users[domain.RoleSuperAdmin].ID), nil, saTok); w.Code != http.StatusForbidden {
		t.Fatalf("self-delete: got %d, want 403", w.Code)
	}
}

func TestAdminStoreOps_AllowedForAdminAndSuperadmin(t *testing.T) {
	r, users := setupRBACRouter(t)
	adminTok, saTok, custTok := tokenFor(t, users[domain.RoleAdmin]), tokenFor(t, users[domain.RoleSuperAdmin]), tokenFor(t, users[domain.RoleCustomer])

	// Customer tidak boleh buat kategori/produk; admin & superadmin boleh.
	if w := doReq(t, r, "POST", "/api/v1/categories", map[string]any{"name": "Test", "slug": "test"}, custTok); w.Code != http.StatusForbidden {
		t.Fatalf("customer create category: got %d, want 403", w.Code)
	}
	w := doReq(t, r, "POST", "/api/v1/categories", map[string]any{"name": "Test", "slug": "test"}, adminTok)
	if w.Code != http.StatusCreated {
		t.Fatalf("admin create category: got %d, want 201: %s", w.Code, w.Body.String())
	}
	cat := decodeData(t, w)
	catID := cat["id"].(string)

	for name, tok := range map[string]string{"admin": adminTok, "superadmin": saTok} {
		w := doReq(t, r, "POST", "/api/v1/products", map[string]any{
			"category_id": catID, "name": "Produk " + name, "slug": "produk-" + name,
			"price": 10000, "stock": 10, "sku": "SKU-" + name,
		}, tok)
		if w.Code != http.StatusCreated {
			t.Fatalf("%s create product: got %d, want 201: %s", name, w.Code, w.Body.String())
		}
		w = doReq(t, r, "GET", "/api/v1/admin/analytics/profit", nil, tok)
		if w.Code != http.StatusOK {
			t.Fatalf("%s profit analytics: got %d, want 200", name, w.Code)
		}
	}
	if w := doReq(t, r, "GET", "/api/v1/admin/analytics/profit", nil, custTok); w.Code != http.StatusForbidden {
		t.Fatalf("customer profit analytics: got %d, want 403", w.Code)
	}
}
