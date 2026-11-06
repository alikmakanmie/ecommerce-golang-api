# SLARAS NUSA FOODLAB — Website

Frontend toko kuliner **SLARAS NUSA FOODLAB** (React + Vite + TypeScript + Tailwind CSS).
Desain Neobrutalism gaya BoldKit: tipografi tebal, border hitam tebal, hard shadow,
dark/light mode. Terintegrasi penuh dengan backend Go di `../ecommerce-api`.

## Menjalankan

```bash
npm install
npm run dev        # http://localhost:5173
npm run build      # hasil di dist/
npm run preview    # sajikan hasil build
```

## Environment

Salin `.env.example` ke `.env` bila perlu mengubah URL API:

```bash
VITE_API_URL=https://api.alikmakanmie.my.id/api/v1   # default (produksi)
# VITE_API_URL=http://localhost:8080/api/v1           # pengembangan lokal
```

## Fitur

- **Auth**: login, register, auto-refresh JWT (401 → refresh → retry), edit profil, CRUD alamat.
- **Katalog**: daftar produk, filter kategori, pencarian real-time (debounce), filter rentang harga, halaman detail.
- **Keranjang**: tambah, ubah kuantitas, hapus, subtotal sinkron ke `GET /cart`.
- **Guest Checkout (tanpa login)**: pengunjung bebas menambah menu — keranjang tamu
  tersimpan di `localStorage` (`snf_guest_cart`). Di checkout ada tab BoldKit
  **Dine-In** (nama + nomor meja 1–20 + catatan per menu), **Takeaway**
  (nama + No. WhatsApp), dan **Delivery** (nama + WA + alamat lengkap).
  Pesanan dibuat lewat `POST /orders/guest-checkout` (publik), lalu modal QRIS
  (kode QR asli dari `qr_string`) + tombol simulasi bayar, dan struk digital di
  `/track/:order_number` (polling status live + cetak struk).
- **Pesanan Saya** (`/my-orders`, publik): riwayat `order_number` tamu di browser
  ini (localStorage `snf_guest_orders`), status disegarkan dari `/orders/track`.
  Saat tamu login, isi keranjang lokalnya dipindahkan ke `/cart` API.
- **Checkout**: pilih alamat, metode QRIS / Virtual Account, buat order, modal pembayaran mock-pay.
- **Pembayaran (simulasi)**: backend tidak menyediakan jalur simulasi aman dari klien (webhook butuh HMAC `WEBHOOK_SECRET`). Tombol **"Simulasikan Pembayaran Sukses"** membuka halaman mock-pay di tab baru, lalu modal mem-polling status order sampai `pending → paid`. Perintah webhook untuk penguji ditampilkan di dalam modal.
- **Pesanan**: riwayat order + batalkan order (pending/paid).
- **Admin** (`role == admin`): CRUD produk, CRUD kategori, ubah status order (processing/shipped/completed).

## Akun demo

| Role | Email | Password |
| --- | --- | --- |
| Admin | admin@ecommerce.local | Admin123! |
| Customer | customer@ecommerce.local | Customer123! |

## Testing (E2E Playwright)

```bash
npx playwright test tests/guest-checkout.spec.ts   # butuh API lokal jalan di :8080
```

Skenario: guest dine-in Meja 04 sampai paid + `/track` + Pesanan Saya,
takeaway, delivery, validasi (tanpa meja / tanpa alamat), member checkout +
admin panel. Simulasi bayar memakai webhook asli (test membaca
`WEBHOOK_SECRET` dari `.env` lokal backend, file dev yang tidak ikut commit).

## Struktur

```
src/
  api/        # client axios + interceptor refresh token, endpoints, types
  components/ # Navbar, ProductCard, ProductVisual, Modal, StatusBadge
  context/    # AuthContext, CartContext, ThemeContext
  pages/      # Home, ProductDetail, Login, Register, Cart, Checkout,
              # Orders, OrderDetail, TrackOrder, MyOrders, Profile, Admin
  utils/      # formatRupiah, formatDate, ongkir, guestOrders
tests/        # E2E Playwright (guest checkout)
```
