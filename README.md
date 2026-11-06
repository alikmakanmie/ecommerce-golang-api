# E-Commerce REST API (Golang)

Backend REST API e-commerce lengkap: auth JWT (access + refresh), RBAC superadmin/admin/customer,
katalog produk, keranjang, checkout, order, dan mock payment gateway (simulasi Midtrans/Xendit/QRIS).

> **Frontend web (React + Tailwind BoldKit)** kini di repositori terpisah:
> https://github.com/alikmakanmie/slaras-nusa-web
> (Aplikasi mobile menyusul di `slaras-nusa-mobile`.)

## Tech Stack

- Go 1.26, Gin, GORM
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

# Webhook settlement (ganti TRANSACTION_ID dari respons checkout).
# Webhook WAJIB ditandatangani HMAC-SHA256 (kunci WEBHOOK_SECRET di .env):
BODY='{"transaction_id":"TRANSACTION_ID","transaction_status":"settlement"}'
SIG=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" | cut -d' ' -f2)
curl -X POST http://localhost:8080/api/v1/payments/webhook \
  -H 'Content-Type: application/json' \
  -H "X-Webhook-Signature: $SIG" \
  -d "$BODY"
```

Ongkir dummy: Rp10.000 bila subtotal < Rp500.000, gratis bila >= Rp500.000.
Stok dikurangi saat checkout dan dikembalikan bila order dibatalkan / payment expire.

## Production Tuning

- `APP_ENV=production` → Gin ReleaseMode (tanpa log debug), GORM logger silent.
- `http.Server` eksplisit: `ReadTimeout 10s`, `WriteTimeout 10s`, `IdleTimeout 60s`,
  `MaxHeaderBytes 1MB` (anti Slowloris / header raksasa).
- SQLite: DSN `_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&cache=shared`
  + PRAGMA `journal_mode=WAL`, `busy_timeout=5000`, pool 1 koneksi (mencegah
  `database is locked`). Untuk beban produksi nyata, pakai PostgreSQL
  (pool 10 idle / 100 open) — lihat catatan benchmark di bawah.
- Build produksi: `go build -ldflags="-s -w" -o bin/api ./cmd/api`
  (binary 23 MB → 15 MB).
- Batas file descriptor VM ini keras di 4096 (`ulimit -n 65535` ditolak kernel);
  itu cukup untuk ~4000 koneksi serentak, di atasnya perlu naikkan dari luar container.

## Security

- Rate limiter per-IP (token bucket, in-memory): seluruh `/api/v1` maks 20 req/detik
  (burst 40); `POST /api/v1/auth/login` maks 5 percobaan/menit (burst 5) → 429.
  IP diambil dari header `X-Forwarded-For` pertama bila ada, else RemoteAddr.
- Security headers di semua respons: `X-Content-Type-Options: nosniff`,
  `X-Frame-Options: DENY`, `X-XSS-Protection: 1; mode=block`,
  `Referrer-Policy: strict-origin-when-cross-origin`, CSP `default-src 'none'`.
- CORS ketat: `Access-Control-Allow-Origin` hanya di-echo bila Origin ada di
  `CORS_ALLOWED_ORIGINS` (dipisah koma); origin asing tidak mendapat header CORS.
- Webhook payment: header `X-Webhook-Signature` berisi HMAC-SHA256 hex dari raw
  body dengan kunci `WEBHOOK_SECRET` (prefix `sha256=` juga diterima, perbandingan
  constant-time via `hmac.Equal`). Tanpa header / signature salah / secret kosong
  → 401. `APP_ENV=production` + `WEBHOOK_SECRET` kosong = server menolak start.
- Audit: `go vet ./...`, `govulncheck ./...`, dan skrip `security-test.sh`
  (JWT palsu/rusak/expired → 401, customer ke endpoint admin → 403,
  SQL injection di `?search=` → 200 tanpa dump, rate limit login → 429,
  webhook tanpa signature → 401).

## Benchmark

Script tanpa dependensi eksternal: `benchmark/bench.go` (stdlib saja).

```bash
go build -o /tmp/bench ./benchmark/bench.go
/tmp/bench -concurrency 50 -duration 15s -xff-prefix 203.0.113
/tmp/bench -method POST -path /api/v1/cart/items -token "$TOKEN" -concurrency 50 -duration 15s -xff-prefix 198.51.100 -body '{"product_id":"...","quantity":1}'
```

PENTING: rate limiter bekerja per-IP, jadi benchmark HARUS memakai `-xff-prefix`
agar tiap worker mensimulasikan IP klien berbeda. Tanpa itu, satu IP dibatasi
20 req/detik dan hampir semua request dijawab 429 (itu batas desain, bukan
kapasitas server — terverifikasi: 1 IP × 50 koneksi hanya mendapat ~23 sukses/detik).
Hasil terukur (VM 4 vCPU AMD EPYC, RAM 7,7 GB, localhost) ada di `benchmark/`.

| Skenario | Sebelum | Sesudah (final) |
| --- | --- | --- |
| Read 50 koneksi | 1515 RPS, avg 27,9 ms, p95 92,5 ms, p99 143,4 ms, 0% error | 1121 RPS sukses, avg 13,4 ms*, p95 75,0 ms, p99 101,6 ms (69,7% request dijawab 429 oleh limiter — lihat catatan) |
| Read 100 koneksi | 1360 RPS, avg 61,1 ms, p95 200,7 ms, p99 301,0 ms, 0% error | 1543 RPS, avg 64,6 ms, p95 141,6 ms, p99 190,0 ms, 0% error |
| Write 50 koneksi (PUT /users/me, auth) | — (tidak diukur sebelum) | 976 RPS, avg 51,0 ms, p95 136,9 ms, p99 189,2 ms, 0% error |

\* Pada run read-50 final, klien closed-loop menembak jauh di atas batas 20 req/detik/IP,
sehingga sebagian besar request dijawab 429 dalam <1 ms (latency rata-rata tercampur
429 + 200). Angka sukses 1121 RPS adalah kapasitas read yang relevan; run 1-IP
membuktikan limiter: 50 koneksi dari 1 IP hanya mendapat ~23 sukses/detik.
Run "sesudah" pertama sempat terkontaminasi instalasi govulncheck di background
dan diulang setelahnya — angka di tabel ini dari run ulang dengan binary final.

Catatan hasil: mode WAL + 1 koneksi SQLite membuat throughput read turun
dibanding baseline (pool 10 koneksi) pada konkurensi tinggi — trade-off yang
disengaja demi konsistensi tulis. Untuk beban produksi, migrasikan ke PostgreSQL.

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

## Foto produk (uploads)

Arsitektur: file asli (master) disimpan di VM Dhodi, salinan web di-sync ke VPS.

- Asli: `uploads-original/` (gitignored) — kirim foto ke Dhodi atau taruh langsung di sini.
- Salinan web: `uploads/` (gitignored) — sisi terpanjang maks 1280px, JPEG q82 / PNG & WebP dioptimasi.
- Sync: `python3 scripts/sync-uploads.py sync` (rsync ke VPS via kunci SSH `~/.ssh/ecommerce-vps`, tanpa password); cron tiap 5 menit.
- Ingest foto baru: `python3 scripts/sync-uploads.py ingest <file> [nama]` — mencetak URL publiknya.
- API menyajikan salinan web di `GET /uploads/<file>` (route statis, direktori `./uploads`).
- `image_url` produk diisi URL publik, mis. `http://203.175.11.164:2150/uploads/<file>` (atau `https://api.alikmakanmie.my.id/uploads/<file>` setelah domain live).
