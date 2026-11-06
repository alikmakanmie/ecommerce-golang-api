/**
 * E2E Guest Checkout vs API lokal (http://localhost:8080/api/v1).
 *
 * Simulasi pembayaran memakai webhook asli: test membaca WEBHOOK_SECRET dari
 * .env lokal backend (file dev, tidak ikut commit) dan menandatangani body
 * dengan HMAC-SHA256 — persis seperti yang dilakukan mock gateway. Tanpa
 * itu, order tidak bisa berubah pending -> paid dari browser.
 */
import { createHmac } from 'node:crypto'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { expect, test, type APIRequestContext, type Page } from '@playwright/test'

const API = 'http://localhost:8080/api/v1'

function webhookSecret(): string {
  const envPath = path.join(os.homedir(), 'workspace', 'ecommerce-api', '.env')
  const env = fs.readFileSync(envPath, 'utf8')
  const m = env.match(/^WEBHOOK_SECRET=(.+)$/m)
  if (!m) throw new Error('WEBHOOK_SECRET tidak ditemukan di .env lokal backend')
  return m[1].trim()
}

/** Selesaikan pembayaran order tamu lewat webhook (gateway mock). */
async function settleOrder(request: APIRequestContext, orderNumber: string) {
  const track = await request.get(`${API}/orders/track/${orderNumber}`)
  expect(track.ok()).toBeTruthy()
  const body = await track.json()
  const tx = body.data.payment.transaction_id as string
  expect(tx).toBeTruthy()
  const payload = JSON.stringify({ transaction_id: tx, transaction_status: 'settlement' })
  const sig = createHmac('sha256', webhookSecret()).update(payload).digest('hex')
  const res = await request.post(`${API}/payments/webhook`, {
    headers: { 'Content-Type': 'application/json', 'X-Webhook-Signature': sig },
    data: payload,
  })
  expect(res.ok()).toBeTruthy()
}

/** Ambil nomor order dari teks "Order ORD-..." di modal pembayaran. */
async function orderNumberFromModal(page: Page): Promise<string> {
  const text = await page.locator('text=/Order ORD-/').first().textContent()
  const m = text?.match(/ORD-[A-Z0-9-]+/)
  expect(m).not.toBeNull()
  return m![0]
}

/** Tambah N produk pertama yang masih ada stoknya dari katalog ke keranjang. */
async function addProducts(page: Page, count: number) {
  await page.goto('/')
  await expect(page.locator('text=Nasi Goreng Slaras').first()).toBeVisible()
  // Hanya tombol yang aktif (produk habis stok ber-tombol disabled).
  const addButtons = page.locator('button[title="Tambah ke keranjang"]:not([disabled])')
  for (let i = 0; i < count; i++) {
    const btn = addButtons.nth(i)
    // Gulir ke tengah viewport: floating cart bar di bawah layar tidak
    // boleh menutupi tombol yang diklik.
    await btn.evaluate((el) => el.scrollIntoView({ block: 'center' }))
    await btn.click()
  }
}

/** Klik tombol simulasi di modal, lalu selesaikan lewat webhook. */
async function simulatePayment(page: Page, request: APIRequestContext, orderNumber: string) {
  await page.getByRole('button', { name: 'Simulasikan Pembayaran Sukses' }).click()
  await expect(page.getByText('Menunggu Pembayaran')).toBeVisible()
  await settleOrder(request, orderNumber)
  await page.getByRole('button', { name: /Saya Sudah Bayar/ }).click()
  await expect(page.getByText('Lunas!')).toBeVisible({ timeout: 20_000 })
}

test.describe('Guest Checkout', () => {
  test('(a) tamu: dine-in Meja 04, bayar simulasi, sampai /track paid + Pesanan Saya', async ({
    page,
    request,
  }) => {
    await addProducts(page, 2)
    await page.goto('/cart')
    await expect(page.locator('article')).toHaveCount(2)
    await page.getByRole('button', { name: 'Checkout', exact: true }).click()
    await expect(page).toHaveURL(/\/checkout$/)

    // Tab Dine-In aktif secara default.
    await expect(page.getByRole('tab', { name: /dine-in/i })).toHaveAttribute('aria-selected', 'true')
    await page.getByLabel(/Nama Pemesan/).fill('Budi Tamu')
    await page.getByRole('button', { name: 'Meja 4' }).click()
    await page.getByPlaceholder(/less sugar/).first().fill('less sugar')
    await page.getByRole('button', { name: /Bayar Sekarang/ }).click()

    // Modal pembayaran QRIS tampil (kode QR asli ter-render).
    await expect(page.getByText('Scan kode QRIS')).toBeVisible()
    const orderNumber = await orderNumberFromModal(page)
    await simulatePayment(page, request, orderNumber)

    // Tutup modal -> diarahkan ke /track/:order_number.
    await page.getByRole('button', { name: 'Lihat Pesanan Saya' }).click()
    await expect(page).toHaveURL(new RegExp(`/track/${orderNumber}$`))
    await expect(page.getByText('Dibayar')).toBeVisible()
    await expect(page.getByText('Meja 4', { exact: true })).toBeVisible()
    await expect(page.getByText('less sugar')).toBeVisible()

    // Riwayat tamu tersimpan di localStorage.
    await page.goto('/my-orders')
    await expect(page.locator(`text=${orderNumber}`).first()).toBeVisible()
  })

  test('(b) tamu: checkout Takeaway', async ({ page, request }) => {
    await addProducts(page, 1)
    await page.goto('/checkout')
    await page.getByRole('tab', { name: /takeaway/i }).click()
    await page.getByLabel(/Nama Pemesan/).fill('Sari Tamu')
    await page.getByLabel(/No. WhatsApp/).fill('081111111111')
    await page.getByRole('button', { name: /Bayar Sekarang/ }).click()
    const orderNumber = await orderNumberFromModal(page)
    await simulatePayment(page, request, orderNumber)
    await page.getByRole('button', { name: 'Lihat Pesanan Saya' }).click()
    await expect(page).toHaveURL(new RegExp(`/track/${orderNumber}$`))
    await expect(page.getByText('Dibayar')).toBeVisible()
    await expect(page.getByText('Takeaway (Bungkus)')).toBeVisible()
  })

  test('(c) tamu: checkout Delivery', async ({ page, request }) => {
    await addProducts(page, 1)
    await page.goto('/checkout')
    await page.getByRole('tab', { name: /delivery/i }).click()
    await page.getByLabel(/Nama Pemesan/).fill('Dedi Tamu')
    await page.getByLabel(/No. WhatsApp/).fill('082222222222')
    await page.getByLabel(/Jalan \/ Alamat/).fill('Jl. Malioboro No. 12')
    await page.getByLabel('Kota').fill('Yogyakarta')
    await page.getByLabel('Provinsi').fill('DIY')
    await page.getByLabel('Kode Pos').fill('55165')
    await page.getByRole('button', { name: /Bayar Sekarang/ }).click()
    const orderNumber = await orderNumberFromModal(page)
    await simulatePayment(page, request, orderNumber)
    await page.getByRole('button', { name: 'Lihat Pesanan Saya' }).click()
    await expect(page).toHaveURL(new RegExp(`/track/${orderNumber}$`))
    await expect(page.getByText('Dibayar')).toBeVisible()
    await expect(page.getByText('Delivery')).toBeVisible()
    await expect(page.getByText(/Malioboro/)).toBeVisible()
  })

  test('(d) validasi: dine-in tanpa meja & delivery tanpa alamat ditolak', async ({ page }) => {
    await addProducts(page, 1)
    await page.goto('/checkout')
    await page.getByLabel(/Nama Pemesan/).fill('Validasi Tamu')
    await page.getByRole('button', { name: /Bayar Sekarang/ }).click()
    await expect(page.getByText('Pilih nomor meja dulu')).toBeVisible()
    await expect(page).toHaveURL(/\/checkout$/)

    await page.getByRole('tab', { name: /delivery/i }).click()
    await page.getByLabel(/No. WhatsApp/).fill('083333333333')
    await page.getByRole('button', { name: /Bayar Sekarang/ }).click()
    await expect(page.getByText('Alamat pengiriman wajib diisi lengkap')).toBeVisible()
    await expect(page).toHaveURL(/\/checkout$/)
  })

  test('(f) Pesanan Saya: order tamu muncul sesudah checkout', async ({ page }) => {
    await page.goto('/my-orders')
    await expect(page.getByText('Belum ada pesanan tamu')).toBeVisible()
    await addProducts(page, 1)
    await page.goto('/checkout')
    await page.getByLabel(/Nama Pemesan/).fill('Fajar Tamu')
    await page.getByRole('button', { name: 'Meja 7' }).click()
    await page.getByRole('button', { name: /Bayar Sekarang/ }).click()
    const orderNumber = await orderNumberFromModal(page)
    await page.getByRole('button', { name: 'Tutup', exact: true }).first().click()
    await page.goto('/my-orders')
    await expect(page.locator(`text=${orderNumber}`).first()).toBeVisible()
  })
})

test.describe('Member & Admin tetap jalan', () => {
  test('(e) member login: checkout lama + admin panel', async ({ page, request }) => {
    // Pastikan member punya alamat (via API, bila belum ada).
    const login = await request.post(`${API}/auth/login`, {
      data: { email: 'customer@ecommerce.local', password: 'Customer123!' },
    })
    expect(login.ok()).toBeTruthy()
    const { data } = await login.json()
    const token = data.access_token as string
    const addrList = await request.get(`${API}/users/me/addresses`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    const addrBody = await addrList.json()
    if (!addrBody.data || addrBody.data.length === 0) {
      const created = await request.post(`${API}/users/me/addresses`, {
        headers: { Authorization: `Bearer ${token}` },
        data: {
          label: 'Rumah',
          recipient_name: 'Customer',
          phone: '081234567891',
          street: 'Jl. Test No. 1',
          city: 'Yogyakarta',
          province: 'DIY',
          postal_code: '55165',
          is_default: true,
        },
      })
      expect(created.ok()).toBeTruthy()
    }

    // Login via UI.
    await page.goto('/login')
    await page.getByLabel(/Email/).fill('customer@ecommerce.local')
    await page.getByLabel(/Kata Sandi|Password/).fill('Customer123!')
    await page.getByRole('button', { name: /Masuk/ }).click()
    await expect(page).toHaveURL(/\/$/)

    await addProducts(page, 1)
    await page.goto('/cart')
    await expect(page.locator('article')).toHaveCount(1)
    await page.getByRole('button', { name: 'Checkout', exact: true }).click()
    await expect(page).toHaveURL(/\/checkout$/)
    await expect(page.getByText('Alamat Pengiriman')).toBeVisible()
    await page.getByRole('button', { name: /Bayar Sekarang/ }).click()
    const orderNumber = await orderNumberFromModal(page)
    await simulatePayment(page, request, orderNumber)
    await page.getByRole('button', { name: 'Lihat Pesanan Saya' }).click()
    await expect(page).toHaveURL(/\/orders\//)
    await expect(page.getByText('Dibayar')).toBeVisible()

    // Admin panel tetap bisa dibuka.
    await page.goto('/login')
    await page.getByLabel(/Email/).fill('admin@ecommerce.local')
    await page.getByLabel(/Kata Sandi|Password/).fill('Admin123!')
    await page.getByRole('button', { name: /Masuk/ }).click()
    await expect(page).toHaveURL(/\/$/)
    await page.goto('/admin')
    await expect(page.getByText('Admin Panel')).toBeVisible()
  })
})
