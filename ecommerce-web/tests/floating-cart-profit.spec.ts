/**
 * E2E Floating Cart Bar + Dashboard Profit Admin vs API lokal
 * (http://localhost:8080/api/v1). Dev server dijalankan otomatis oleh
 * playwright.config.ts (port 5173, VITE_API_URL ke API lokal).
 *
 * Simulasi pembayaran memakai webhook asli (baca WEBHOOK_SECRET dari .env
 * lokal backend) — sama seperti tests/guest-checkout.spec.ts.
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

function formatIDR(n: number): string {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(n)
}

/** Selesaikan pembayaran order tamu/member lewat webhook (gateway mock). */
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

/** Klik tombol simulasi di modal, lalu selesaikan lewat webhook. */
async function simulatePayment(page: Page, request: APIRequestContext, orderNumber: string) {
  await page.getByRole('button', { name: 'Simulasikan Pembayaran Sukses' }).click()
  await expect(page.getByText('Menunggu Pembayaran')).toBeVisible()
  await settleOrder(request, orderNumber)
  await page.getByRole('button', { name: /Saya Sudah Bayar/ }).click()
  await expect(page.getByText('Lunas!')).toBeVisible({ timeout: 20_000 })
}

/** Tambah N produk pertama yang masih ada stoknya dari katalog ke keranjang. */
async function addProducts(page: Page, count: number) {
  await page.goto('/')
  await expect(page.locator('text=Nasi Goreng Slaras').first()).toBeVisible()
  const addButtons = page.locator('button[title="Tambah ke keranjang"]:not([disabled])')
  for (let i = 0; i < count; i++) {
    const btn = addButtons.nth(i)
    await btn.evaluate((el) => el.scrollIntoView({ block: 'center' }))
    await btn.click()
  }
}

/** Tambah SATU produk tertentu (indeks ke-n di antara tombol yang aktif). */
async function addProductAt(page: Page, index: number) {
  await page.goto('/')
  await expect(page.locator('text=Nasi Goreng Slaras').first()).toBeVisible()
  const btn = page.locator('button[title="Tambah ke keranjang"]:not([disabled])').nth(index)
  await btn.evaluate((el) => el.scrollIntoView({ block: 'center' }))
  await btn.click()
}

async function loginAs(page: Page, email: string, password: string) {
  await page.goto('/login')
  await page.getByLabel(/Email/).fill(email)
  await page.getByLabel(/Kata Sandi|Password/).fill(password)
  await page.getByRole('button', { name: /Masuk/ }).click()
  await expect(page).toHaveURL(/\/$/)
}

/** Ambil access token yang sudah tersimpan di localStorage sesudah login via UI. */
async function storedAccessToken(page: Page): Promise<string> {
  const token = await page.evaluate(() => localStorage.getItem('snf_access_token'))
  expect(token, 'access token harus tersimpan di localStorage sesudah login').toBeTruthy()
  return token!
}

interface ApiProduct {
  id: string
  name: string
  price: number
  stock: number
  cost_price?: number
}

async function apiProducts(request: APIRequestContext): Promise<ApiProduct[]> {
  const res = await request.get(`${API}/products?limit=100`)
  expect(res.ok()).toBeTruthy()
  const body = await res.json()
  return body.data as ApiProduct[]
}

async function apiProfit(request: APIRequestContext, token: string, period: string) {
  const res = await request.get(`${API}/admin/analytics/profit?period=${period}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  expect(res.ok()).toBeTruthy()
  const body = await res.json()
  return body.data
}

/** Total yang tampil di floating bar harus sama dengan harga produk yang namanya tertera di bar. */
async function expectBarTotalMatchesNames(page: Page, request: APIRequestContext) {
  const namesText = (await page.getByTestId('floating-cart-items').textContent()) ?? ''
  const names = namesText
    .replace(/\+\d+ menu lagi$/, '')
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
  expect(names.length).toBeGreaterThan(0)
  const products = await apiProducts(request)
  let expected = 0
  for (const name of names) {
    const p = products.find((x) => x.name === name)
    expect(p, `produk "${name}" harus ada di API`).toBeTruthy()
    expected += p!.price
  }
  // Total bar = harga tiap menu yang tampil; bila ada "+N menu lagi",
  // total bisa lebih besar — di sini hanya 1-2 item jadi nama lengkap.
  const totalText = (await page.getByTestId('floating-cart-total').textContent()) ?? ''
  expect(totalText).toContain(formatIDR(expected))
}

test.describe('Floating Cart Bar', () => {
  test('(a) tamu: tambah 1 produk -> bar muncul; tambah ke-2 -> badge & total berubah', async ({
    page,
    request,
  }) => {
    await expect(page.getByTestId('floating-cart-bar')).toHaveCount(0)
    await addProductAt(page, 0)
    const bar = page.getByTestId('floating-cart-bar')
    await expect(bar).toBeVisible()
    await expect(page.getByTestId('floating-cart-count')).toHaveText(/\[ 1 ITEM \]/)
    await expectBarTotalMatchesNames(page, request)

    await addProductAt(page, 1)
    await expect(page.getByTestId('floating-cart-count')).toHaveText(/\[ 2 ITEM \]/)
    await expectBarTotalMatchesNames(page, request)
  })

  test('(b) drawer keranjang dari floating bar: qty +/-, hapus, angka ikut berubah', async ({
    page,
  }) => {
    await addProducts(page, 2)
    const bar = page.getByTestId('floating-cart-bar')
    await expect(bar).toBeVisible()
    await bar.getByRole('button', { name: 'Lihat Keranjang' }).click()
    const drawer = page.getByTestId('cart-drawer')
    await expect(drawer).toBeVisible()
    await expect(page.getByTestId('drawer-item')).toHaveCount(2)

    // Qty item pertama +1 -> total item 3.
    await page
      .getByTestId('drawer-item')
      .first()
      .getByRole('button', { name: 'Tambah' })
      .click()
    await expect(page.getByTestId('floating-cart-count')).toHaveText(/\[ 3 ITEM \]/)
    // Qty item pertama -1 -> kembali 2.
    await page
      .getByTestId('drawer-item')
      .first()
      .getByRole('button', { name: 'Kurangi' })
      .click()
    await expect(page.getByTestId('floating-cart-count')).toHaveText(/\[ 2 ITEM \]/)

    // Subtotal drawer harus sama dengan total di floating bar.
    const drawerSubtotal = (await page.getByTestId('drawer-subtotal').textContent()) ?? ''
    const barTotal = (await page.getByTestId('floating-cart-total').textContent()) ?? ''
    expect(drawerSubtotal.trim()).toBe(barTotal.trim())

    // Hapus item pertama -> tinggal 1 item.
    await page
      .getByTestId('drawer-item')
      .first()
      .getByRole('button', { name: 'Hapus item' })
      .click()
    await expect(page.getByTestId('floating-cart-count')).toHaveText(/\[ 1 ITEM \]/)
    await expect(page.getByTestId('drawer-item')).toHaveCount(1)
    await drawer.getByRole('button', { name: 'Tutup keranjang' }).click()
    await expect(drawer).toBeHidden()
  })

  test('(c) CHECKOUT SEKARANG dari floating bar -> dine-in sampai paid di /track', async ({
    page,
    request,
  }) => {
    await addProducts(page, 1)
    const bar = page.getByTestId('floating-cart-bar')
    await expect(bar).toBeVisible()
    await bar.getByRole('button', { name: /Checkout Sekarang/ }).click()
    await expect(page).toHaveURL(/\/checkout$/)

    await page.getByLabel(/Nama Pemesan/).fill('Bar Tamu')
    await page.getByRole('button', { name: 'Meja 4' }).click()
    await page.getByRole('button', { name: /Bayar Sekarang/ }).click()
    await expect(page.getByText('Scan kode QRIS')).toBeVisible()
    const orderNumber = await orderNumberFromModal(page)
    await simulatePayment(page, request, orderNumber)
    await page.getByRole('button', { name: 'Lihat Pesanan Saya' }).click()
    await expect(page).toHaveURL(new RegExp(`/track/${orderNumber}$`))
    await expect(page.getByText('Dibayar')).toBeVisible()
  })

  test('(d) bar hilang saat keranjang kosong & saat di /checkout', async ({ page }) => {
    await addProducts(page, 1)
    await expect(page.getByTestId('floating-cart-bar')).toBeVisible()
    // Hapus satu-satunya item lewat drawer -> bar hilang.
    await page
      .getByTestId('floating-cart-bar')
      .getByRole('button', { name: 'Lihat Keranjang' })
      .click()
    await page
      .getByTestId('drawer-item')
      .first()
      .getByRole('button', { name: 'Hapus item' })
      .click()
    await expect(page.getByTestId('floating-cart-bar')).toHaveCount(0)

    // Tambah lagi, lalu buka /checkout -> bar tidak tampil di sana.
    await addProducts(page, 1)
    await expect(page.getByTestId('floating-cart-bar')).toBeVisible()
    await page.goto('/checkout')
    await expect(page.getByTestId('floating-cart-bar')).toHaveCount(0)
  })
})

test.describe('Dashboard Profit Admin', () => {
  test('(e) KPI cocok dengan API, filter periode mengubah grafik', async ({ page, request }) => {
    await loginAs(page, 'admin@ecommerce.local', 'Admin123!')
    await page.goto('/admin')
    await expect(page.getByTestId('profit-dashboard')).toBeVisible()
    await expect(page.getByTestId('kpi-gross-revenue')).toBeVisible()
    await expect(page.getByTestId('kpi-net-profit')).toBeVisible()
    await expect(page.getByTestId('kpi-margin')).toBeVisible()
    await expect(page.getByTestId('profit-chart')).toBeVisible()
    await expect(page.getByTestId('top-products-table')).toBeVisible()
    await expect(page.getByTestId('recent-transactions-table')).toBeVisible()

    const token = await storedAccessToken(page)
    // Bandingkan KPI dengan API langsung; UI di-refetch tiap iterasi agar
    // angka yang dibandingkan selalu dari momen yang sama.
    await expect
      .poll(
        async () => {
          await page.getByRole('tab', { name: '30 HARI' }).click()
          await page.waitForTimeout(700)
          const api = await apiProfit(request, token, '30d')
          const ui = (await page.getByTestId('kpi-gross-revenue').textContent()) ?? ''
          return ui.includes(formatIDR(api.summary.gross_revenue))
        },
        { timeout: 20_000 },
      )
      .toBe(true)

    // Filter periode mengubah panjang deret grafik.
    await page.getByRole('tab', { name: '7 HARI' }).click()
    await expect(page.locator('[data-testid="chart-bar-revenue"]')).toHaveCount(7)
    await page.getByRole('tab', { name: 'HARI INI' }).click()
    await expect(page.locator('[data-testid="chart-bar-revenue"]')).toHaveCount(1)
    await page.getByRole('tab', { name: '30 HARI' }).click()
    await expect(page.locator('[data-testid="chart-bar-revenue"]')).toHaveCount(30)
  })

  test('(f) admin ubah cost_price produk via form -> tersimpan di API', async ({
    page,
    request,
  }) => {
    await loginAs(page, 'admin@ecommerce.local', 'Admin123!')
    await page.goto('/admin')
    await page.getByRole('button', { name: /Produk \(/ }).click()
    const firstArticle = page.locator('article').first()
    await expect(firstArticle).toBeVisible()
    const productName = ((await firstArticle.locator('p.font-display').first().textContent()) ?? '').trim()
    expect(productName.length).toBeGreaterThan(0)

    await firstArticle.getByRole('button', { name: 'Ubah' }).click()
    await page.getByLabel(/Harga Modal/).fill('7777')
    await page.getByRole('button', { name: 'Simpan Perubahan' }).click()
    await expect(page.getByText('Produk diperbarui')).toBeVisible()

    const products = await apiProducts(request)
    const edited = products.find((p) => p.name === productName)
    expect(edited, `produk "${productName}" harus ada di API`).toBeTruthy()
    expect(edited!.cost_price).toBe(7777)
  })

  test('(g) EXPORT CSV mengunduh file laporan', async ({ page }) => {
    await loginAs(page, 'admin@ecommerce.local', 'Admin123!')
    await page.goto('/admin')
    await expect(page.getByTestId('profit-dashboard')).toBeVisible()
    const downloadPromise = page.waitForEvent('download')
    await page.getByTestId('export-csv').click()
    const download = await downloadPromise
    expect(download.suggestedFilename()).toMatch(/^laporan-profit-\d{4}-\d{2}-\d{2}\.csv$/)
    const filePath = await download.path()
    expect(filePath).not.toBeNull()
    const content = fs.readFileSync(filePath!, 'utf8')
    expect(content).toContain('LAPORAN PROFIT')
    expect(content).toContain('Total Omzet (Gross Revenue)')
    expect(content).toContain('TRANSAKSI TERBARU')
    expect(content).toContain('MENU PALING MENGUNTUNGKAN')
  })
})

test.describe('Alur lama tetap jalan', () => {
  test('(h) member: floating bar -> checkout lama sampai paid di /orders/:id', async ({
    page,
    request,
  }) => {
    await loginAs(page, 'customer@ecommerce.local', 'Customer123!')
    // Pastikan member punya alamat (pakai token dari localStorage — tanpa
    // login API tambahan, agar tidak kena rate-limit login 5/menit).
    const token = await storedAccessToken(page)
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

    await addProducts(page, 1)
    const bar = page.getByTestId('floating-cart-bar')
    await expect(bar).toBeVisible()
    await bar.getByRole('button', { name: /Checkout Sekarang/ }).click()
    await expect(page).toHaveURL(/\/checkout$/)
    await expect(page.getByText('Alamat Pengiriman')).toBeVisible()
    await page.getByRole('button', { name: /Bayar Sekarang/ }).click()
    const orderNumber = await orderNumberFromModal(page)
    await simulatePayment(page, request, orderNumber)
    await page.getByRole('button', { name: 'Lihat Pesanan Saya' }).click()
    await expect(page).toHaveURL(/\/orders\//)
    await expect(page.getByText('Dibayar')).toBeVisible()
  })
})
