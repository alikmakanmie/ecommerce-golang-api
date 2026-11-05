# E-Commerce REST API (Golang)

Backend REST API e-commerce lengkap: auth JWT (access + refresh), RBAC admin/customer,
katalog produk, keranjang, checkout, order, dan mock payment gateway (simulasi Midtrans/Xendit/QRIS).

## Tech Stack

- Go 1.24, Gin, GORM
- PostgreSQL (produksi / Docker) dan SQLite (lokal / testing)
- JWT access + refresh token, bcrypt
- Clean Architecture: handler → service → repository → domain

## Cara Menjalankan

### Lokal (SQLite, tanpa Docker)

```bash
cp .env.example .env
# pastikan .env berisi DB_DRIVER=sqlite
go mod tidy
go run ./cmd/api
```

Server berjalan di `http://localhost:8080`, health: `GET /health`.

### Docker Compose (PostgreSQL)

```bash
cp .env.example .env
# ubah DB_DRIVER=postgres di .env
docker compose up --build -d
```

## Akun Seeder (SEED_DB=true)

| Role | Email | Password |
| --- | --- | --- |
| Admin | admin@ecommerce.local | Admin123! |
| Customer | customer@ecommerce.local | Customer123! |

Seeder juga membuat 4 kategori dan 8 produk dummy. Seeder idempotent (skip bila admin sudah ada).

## Endpoint

| Method | Endpoint | Auth |
| --- | --- | --- |
| GET | /health, /api/v1/health | publik |
| POST | /api/v1/auth/register | publik |
| POST | /api/v1/auth/login | publik |
| POST | /api/v1/auth/refresh | publik |
| POST | /api/v1/auth/logout | user |
| GET | /api/v1/products, /api/v1/products/:id | publik |
| GET | /api/v1/categories, /api/v1/categories/:id | publik |
| POST/PUT/DELETE | /api/v1/products, /api/v1/products/:id | admin |
| POST/PUT/DELETE | /api/v1/categories, /api/v1/categories/:id | admin |
| GET/PUT | /api/v1/users/me | user |
| GET/POST/DELETE | /api/v1/users/me/addresses | user |
| PUT | /api/v1/users/me/addresses/:id/default | user |
| GET | /api/v1/cart | user |
| POST | /api/v1/cart/items | user |
| PUT/DELETE | /api/v1/cart/items/:id | user |
| POST | /api/v1/orders/checkout | user |
| GET | /api/v1/orders, /api/v1/orders/:id | user |
| POST | /api/v1/orders/:id/cancel | user |
| GET | /api/v1/admin/orders | admin |
| PATCH | /api/v1/admin/orders/:id/status | admin |
| POST | /api/v1/payments/webhook | publik (simulasi) |
| GET | /api/v1/mock-pay/:transaction_id | publik |

Filter produk: `?search=hp&category_id=<uuid>&min_price=100000&max_price=5000000&page=1&limit=20`.

Status order: `pending → paid → processing → shipped → completed`, `pending/paid → cancelled`.

## Contoh cURL

```bash
# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Budi","email":"budi@example.com","password":"Password123","phone":"081234567890"}'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"customer@ecommerce.local","password":"Customer123!"}'

# List produk
curl http://localhost:8080/api/v1/products

# Tambah ke keranjang (ganti TOKEN dan PRODUCT_ID)
curl -X POST http://localhost:8080/api/v1/cart/items \
  -H "Authorization: Bearer TOKEN" -H 'Content-Type: application/json' \
  -d '{"product_id":"PRODUCT_ID","quantity":2}'

# Checkout (ganti ADDRESS_ID)
curl -X POST http://localhost:8080/api/v1/orders/checkout \
  -H "Authorization: Bearer TOKEN" -H 'Content-Type: application/json' \
  -d '{"address_id":"ADDRESS_ID","payment_method":"qris"}'

# Webhook settlement (ganti TRANSACTION_ID dari respons checkout)
curl -X POST http://localhost:8080/api/v1/payments/webhook \
  -H 'Content-Type: application/json' \
  -d '{"transaction_id":"TRANSACTION_ID","transaction_status":"settlement"}'
```

Ongkir dummy: Rp10.000 bila subtotal < Rp500.000, gratis bila >= Rp500.000.
Stok dikurangi saat checkout dan dikembalikan bila order dibatalkan / payment expire.

## Mock Payment Gateway

`internal/payment/mock_gateway.go` menghasilkan `transaction_id` (`MOCK-...`),
`payment_url` (`/api/v1/mock-pay/<tx>`), dan `qr_string` untuk metode `qris`.
Webhook menerima `{"transaction_id": "...", "transaction_status": "settlement|expire|cancel|failed|pending"}`
(field `status` juga diterima sebagai alias). `settlement` mengubah order menjadi `paid`,
`expire`/`cancel` membatalkan order pending dan mengembalikan stok. Webhook settlement idempotent.

## Testing

```bash
go test -v ./...
```

Test memakai SQLite in-memory (`:memory:`), tanpa server HTTP dan tanpa file `.db`.

## Struktur Folder

```
cmd/api/main.go
internal/config, database, domain, repository, service, payment, middleware, handler, router, seeder
pkg/utils
```
