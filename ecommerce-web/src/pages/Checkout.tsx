import { useEffect, useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Bike, MapPin, QrCode, Store, UtensilsCrossed, Wallet } from 'lucide-react'
import { addressApi, orderApi } from '../api/endpoints'
import { apiErrorMessage } from '../api/client'
import type { Address, GuestCheckoutInput, Order, OrderType } from '../api/types'
import { useCart } from '../context/CartContext'
import { useAuth } from '../context/AuthContext'
import { formatRupiah, shippingCost } from '../utils/format'
import { Spinner } from '../components/ProtectedRoute'
import { PaymentModal } from '../components/PaymentModal'
import { saveGuestOrder } from '../utils/guestOrders'

const ORDER_TYPES: { value: OrderType; label: string; icon: typeof UtensilsCrossed }[] = [
  { value: 'dine_in', label: 'Dine-In (Makan di Tempat)', icon: UtensilsCrossed },
  { value: 'takeaway', label: 'Takeaway (Bungkus)', icon: Store },
  { value: 'delivery', label: 'Delivery', icon: Bike },
]

function MethodPicker({ method, setMethod }: { method: string; setMethod: (m: string) => void }) {
  const opt = (value: string, icon: React.ReactNode, title: string, desc: string) => (
    <label
      className={`flex cursor-pointer items-center gap-3 border-2 p-4 ${
        method === value
          ? 'border-black bg-yolk/30 shadow-[3px_3px_0px_0px_#000] dark:border-white dark:shadow-[3px_3px_0px_0px_#fff]'
          : 'border-zinc-300 hover:border-black dark:border-zinc-700 dark:hover:border-white'
      }`}
    >
      <input
        type="radio"
        name="method"
        className="accent-black"
        checked={method === value}
        onChange={() => setMethod(value)}
      />
      {icon}
      <span>
        <span className="block font-display text-sm font-black uppercase">{title}</span>
        <span className="block text-xs text-zinc-600 dark:text-zinc-400">{desc}</span>
      </span>
    </label>
  )
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {opt(
        'qris',
        <QrCode className="h-6 w-6" />,
        'QRIS',
        'Scan QR, semua e-wallet & m-banking',
      )}
      {opt(
        'va',
        <Wallet className="h-6 w-6" />,
        'Virtual Account',
        'Transfer ke nomor VA (simulasi)',
      )}
    </div>
  )
}

export default function Checkout() {
  const { user } = useAuth()
  const { cart, loading: cartLoading, refresh, clearLocal, isGuestCart, setItemNotes } = useCart()
  const navigate = useNavigate()

  // --- Alur member (tetap seperti sebelumnya) ---
  const [addresses, setAddresses] = useState<Address[]>([])
  const [loadingAddr, setLoadingAddr] = useState(true)
  const [selectedAddress, setSelectedAddress] = useState('')

  // --- Bersama ---
  const [method, setMethod] = useState('qris')
  const [notes, setNotes] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [createdOrder, setCreatedOrder] = useState<Order | null>(null)
  const [payOpen, setPayOpen] = useState(false)

  // --- Alur tamu ---
  const [orderType, setOrderType] = useState<OrderType>('dine_in')
  const [customerName, setCustomerName] = useState('')
  const [phone, setPhone] = useState('')
  const [tableNumber, setTableNumber] = useState('')
  const [delivery, setDelivery] = useState({ street: '', city: '', province: '', postal_code: '' })

  useEffect(() => {
    if (!user) {
      setLoadingAddr(false)
      return
    }
    addressApi
      .list()
      .then((list) => {
        setAddresses(list)
        const def = list.find((a) => a.is_default) ?? list[0]
        if (def) setSelectedAddress(def.id)
      })
      .catch(() => setAddresses([]))
      .finally(() => setLoadingAddr(false))
  }, [user])

  const items = cart?.cart.items ?? []
  const subtotal = cart?.subtotal ?? 0
  // Tamu: dine-in/takeaway tanpa ongkir; delivery pakai aturan yang sama
  // dengan backend (Rp10.000 bila < Rp500.000, gratis bila >= Rp500.000)
  // sebagai estimasi — total final mengikuti respons server.
  const shipping = !user
    ? orderType === 'delivery'
      ? shippingCost(subtotal)
      : 0
    : items.length > 0
      ? shippingCost(subtotal)
      : 0

  if (cartLoading || loadingAddr) return <Spinner label="Menyiapkan checkout..." />

  if (items.length === 0 && !createdOrder) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-16 text-center">
        <p className="font-display text-2xl font-black uppercase">Keranjang kosong</p>
        <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
          Tambah menu dulu sebelum checkout.
        </p>
        <Link to="/" className="btn btn-primary btn-sm mt-6">
          Lihat Menu
        </Link>
      </div>
    )
  }

  const submitGuest = async (e: FormEvent) => {
    e.preventDefault()
    const name = customerName.trim()
    if (!name) {
      setError('Nama pemesan wajib diisi')
      return
    }
    if (orderType === 'dine_in' && !tableNumber) {
      setError('Pilih nomor meja dulu')
      return
    }
    if ((orderType === 'takeaway' || orderType === 'delivery') && !phone.trim()) {
      setError('Nomor WhatsApp wajib diisi')
      return
    }
    if (orderType === 'delivery') {
      const missing = !delivery.street.trim() || !delivery.city.trim() || !delivery.province.trim() || !delivery.postal_code.trim()
      if (missing) {
        setError('Alamat pengiriman wajib diisi lengkap (jalan, kota, provinsi, kode pos)')
        return
      }
    }
    setError('')
    setBusy(true)
    try {
      const input: GuestCheckoutInput = {
        items: items.map((it) => ({
          product_id: it.product_id,
          quantity: it.quantity,
          notes: it.notes?.trim() || undefined,
        })),
        customer_name: name,
        customer_phone: phone.trim() || undefined,
        order_type: orderType,
        table_number: orderType === 'dine_in' ? tableNumber : undefined,
        payment_method: method === 'va' ? 'va' : 'qris',
        notes: notes.trim() || undefined,
        delivery_address:
          orderType === 'delivery'
            ? {
                street: delivery.street.trim(),
                city: delivery.city.trim(),
                province: delivery.province.trim(),
                postal_code: delivery.postal_code.trim(),
              }
            : undefined,
      }
      const order = await orderApi.guestCheckout(input)
      saveGuestOrder(order)
      setCreatedOrder(order)
      setPayOpen(true)
      clearLocal()
    } catch (err) {
      setError(apiErrorMessage(err, 'Checkout gagal'))
    } finally {
      setBusy(false)
    }
  }

  const submitMember = async (e: FormEvent) => {
    e.preventDefault()
    if (!selectedAddress) {
      setError('Pilih alamat pengiriman dulu')
      return
    }
    setError('')
    setBusy(true)
    try {
      const order = await orderApi.checkout(selectedAddress, method, notes.trim() || undefined)
      setCreatedOrder(order)
      setPayOpen(true)
      await refresh()
    } catch (err) {
      setError(apiErrorMessage(err, 'Checkout gagal'))
    } finally {
      setBusy(false)
    }
  }

  // Jangan navigasi di sini: biarkan modal menampilkan status "Lunas!",
  // navigasi terjadi saat modal ditutup (onClose di bawah).
  const handlePaid = () => {}

  const handlePayClose = () => {
    setPayOpen(false)
    if (!createdOrder) return
    if (isGuestCart || !createdOrder.user_id) {
      navigate(`/track/${createdOrder.order_number}`)
    } else {
      navigate(`/orders/${createdOrder.id}`)
    }
  }

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <h1 className="title">Checkout</h1>
      <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
        {user
          ? 'Pilih alamat, metode bayar, lalu buat pesanan.'
          : 'Pesan tanpa login — pilih cara makanmu, bayar QRIS, selesai.'}
      </p>

      <form onSubmit={user ? submitMember : submitGuest} className="mt-6 grid gap-6 lg:grid-cols-3">
        <div className="flex flex-col gap-6 lg:col-span-2">
          {user ? (
            <>
              <section className="card p-5">
                <h2 className="flex items-center gap-2 font-display text-lg font-black uppercase">
                  <MapPin className="h-5 w-5" /> Alamat Pengiriman
                </h2>
                {addresses.length === 0 ? (
                  <div className="mt-4 border-2 border-dashed border-zinc-400 p-4 text-center">
                    <p className="text-sm">Belum ada alamat tersimpan.</p>
                    <Link to="/profile" className="btn btn-primary btn-sm mt-3">
                      Tambah Alamat di Profil
                    </Link>
                  </div>
                ) : (
                  <div className="mt-4 flex flex-col gap-3">
                    {addresses.map((a) => (
                      <label
                        key={a.id}
                        className={`flex cursor-pointer gap-3 border-2 p-3 transition-all ${
                          selectedAddress === a.id
                            ? 'border-black bg-yolk/30 shadow-[3px_3px_0px_0px_#000] dark:border-white dark:shadow-[3px_3px_0px_0px_#fff]'
                            : 'border-zinc-300 hover:border-black dark:border-zinc-700 dark:hover:border-white'
                        }`}
                      >
                        <input
                          type="radio"
                          name="address"
                          className="mt-1 accent-black"
                          checked={selectedAddress === a.id}
                          onChange={() => setSelectedAddress(a.id)}
                        />
                        <span>
                          <span className="font-display text-sm font-black uppercase">
                            {a.label} {a.is_default && <span className="chip ml-1 bg-yolk !text-black">Default</span>}
                          </span>
                          <span className="block text-sm font-bold">{a.recipient_name} • {a.phone}</span>
                          <span className="block text-sm text-zinc-600 dark:text-zinc-400">
                            {a.street}, {a.city}, {a.province} {a.postal_code}
                          </span>
                        </span>
                      </label>
                    ))}
                    <Link to="/profile" className="btn btn-ghost btn-sm w-fit">
                      + Kelola Alamat
                    </Link>
                  </div>
                )}
              </section>

              <section className="card p-5">
                <h2 className="flex items-center gap-2 font-display text-lg font-black uppercase">
                  <Wallet className="h-5 w-5" /> Metode Pembayaran
                </h2>
                <div className="mt-4">
                  <MethodPicker method={method} setMethod={setMethod} />
                </div>
                <div className="mt-4">
                  <label className="label" htmlFor="notes">
                    Catatan Pesanan (opsional)
                  </label>
                  <textarea
                    id="notes"
                    className="input"
                    rows={2}
                    placeholder="Contoh: tidak pedas, antar jam 12 siang..."
                    value={notes}
                    onChange={(e) => setNotes(e.target.value)}
                  />
                </div>
              </section>
            </>
          ) : (
            <>
              <section className="card p-5">
                <h2 className="font-display text-lg font-black uppercase">Cara Makan</h2>
                <div className="mt-4 grid gap-3 sm:grid-cols-3" role="tablist" aria-label="Jenis pesanan">
                  {ORDER_TYPES.map(({ value, label, icon: Icon }) => (
                    <button
                      key={value}
                      type="button"
                      role="tab"
                      aria-selected={orderType === value}
                      onClick={() => setOrderType(value)}
                      className={`flex flex-col items-start gap-2 border-2 p-4 text-left font-display text-sm font-black uppercase transition-all ${
                        orderType === value
                          ? 'border-black bg-yolk text-black shadow-[4px_4px_0px_0px_#000] dark:border-white dark:shadow-[4px_4px_0px_0px_#fff]'
                          : 'border-zinc-300 hover:border-black dark:border-zinc-700 dark:hover:border-white'
                      }`}
                    >
                      <Icon className="h-6 w-6" />
                      {label}
                    </button>
                  ))}
                </div>
              </section>

              <section className="card p-5">
                <h2 className="font-display text-lg font-black uppercase">Data Pemesan</h2>
                <div className="mt-4 flex flex-col gap-4">
                  <div>
                    <label className="label" htmlFor="customer-name">
                      Nama Pemesan *
                    </label>
                    <input
                      id="customer-name"
                      className="input"
                      placeholder="Contoh: Budi"
                      value={customerName}
                      onChange={(e) => setCustomerName(e.target.value)}
                    />
                  </div>

                  {orderType === 'dine_in' && (
                    <div>
                      <span className="label">Nomor Meja *</span>
                      <div className="grid grid-cols-5 gap-2">
                        {Array.from({ length: 20 }, (_, i) => i + 1).map((n) => {
                          const label = `Meja ${n}`
                          const active = tableNumber === label
                          return (
                            <button
                              key={n}
                              type="button"
                              onClick={() => setTableNumber(label)}
                              aria-pressed={active}
                              className={`border-2 px-2 py-2.5 font-display text-xs font-black uppercase transition-all sm:text-sm ${
                                active
                                  ? 'border-black bg-yolk text-black shadow-[3px_3px_0px_0px_#000] dark:border-white dark:shadow-[3px_3px_0px_0px_#fff]'
                                  : 'border-zinc-300 hover:border-black dark:border-zinc-700 dark:hover:border-white'
                              }`}
                            >
                              {label}
                            </button>
                          )
                        })}
                      </div>
                    </div>
                  )}

                  {orderType !== 'dine_in' && (
                    <div>
                      <label className="label" htmlFor="customer-phone">
                        No. WhatsApp *
                      </label>
                      <input
                        id="customer-phone"
                        className="input"
                        inputMode="tel"
                        placeholder="Contoh: 081234567890"
                        value={phone}
                        onChange={(e) => setPhone(e.target.value)}
                      />
                    </div>
                  )}

                  {orderType === 'delivery' && (
                    <div className="flex flex-col gap-4 border-2 border-dashed border-zinc-400 p-4">
                      <p className="font-display text-xs font-black uppercase tracking-wider">
                        Alamat Pengiriman *
                      </p>
                      <div>
                        <label className="label" htmlFor="addr-street">
                          Jalan / Alamat
                        </label>
                        <input
                          id="addr-street"
                          className="input"
                          placeholder="Contoh: Jl. Malioboro No. 12"
                          value={delivery.street}
                          onChange={(e) => setDelivery((d) => ({ ...d, street: e.target.value }))}
                        />
                      </div>
                      <div className="grid gap-4 sm:grid-cols-2">
                        <div>
                          <label className="label" htmlFor="addr-city">
                            Kota
                          </label>
                          <input
                            id="addr-city"
                            className="input"
                            placeholder="Yogyakarta"
                            value={delivery.city}
                            onChange={(e) => setDelivery((d) => ({ ...d, city: e.target.value }))}
                          />
                        </div>
                        <div>
                          <label className="label" htmlFor="addr-province">
                            Provinsi
                          </label>
                          <input
                            id="addr-province"
                            className="input"
                            placeholder="DIY"
                            value={delivery.province}
                            onChange={(e) => setDelivery((d) => ({ ...d, province: e.target.value }))}
                          />
                        </div>
                      </div>
                      <div>
                        <label className="label" htmlFor="addr-postal">
                          Kode Pos
                        </label>
                        <input
                          id="addr-postal"
                          className="input"
                          inputMode="numeric"
                          placeholder="55165"
                          value={delivery.postal_code}
                          onChange={(e) => setDelivery((d) => ({ ...d, postal_code: e.target.value }))}
                        />
                      </div>
                    </div>
                  )}

                  {orderType === 'dine_in' && (
                    <div>
                      <span className="label">Catatan per Menu (opsional)</span>
                      <div className="flex flex-col gap-2">
                        {items.map((item) => (
                          <div key={item.id} className="flex flex-col gap-1 border-2 border-zinc-300 p-3 dark:border-zinc-700">
                            <span className="text-sm font-bold">
                              {item.quantity}× {item.product?.name ?? 'Produk'}
                            </span>
                            <input
                              className="input !py-2 text-xs"
                              placeholder="Contoh: less sugar / normal ice"
                              value={item.notes ?? ''}
                              onChange={(e) => setItemNotes(item.id, e.target.value)}
                            />
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  <div>
                    <label className="label" htmlFor="guest-notes">
                      Catatan Pesanan (opsional)
                    </label>
                    <textarea
                      id="guest-notes"
                      className="input"
                      rows={2}
                      placeholder="Contoh: tidak pedas, bungkus terpisah..."
                      value={notes}
                      onChange={(e) => setNotes(e.target.value)}
                    />
                  </div>
                </div>
              </section>

              <section className="card p-5">
                <h2 className="flex items-center gap-2 font-display text-lg font-black uppercase">
                  <Wallet className="h-5 w-5" /> Metode Pembayaran
                </h2>
                <div className="mt-4">
                  <MethodPicker method={method} setMethod={setMethod} />
                </div>
                {!user && (
                  <p className="mt-3 text-xs text-zinc-500">
                    Tanpa login — struk & status pesanan tersimpan di browser ini (menu{' '}
                    <Link to="/my-orders" className="font-bold underline">
                      Pesanan Saya
                    </Link>
                    ).
                  </p>
                )}
              </section>
            </>
          )}
        </div>

        <aside className="card h-fit p-5">
          <h2 className="font-display text-lg font-black uppercase">Ringkasan</h2>
          <ul className="mt-3 flex flex-col gap-2 text-sm">
            {items.map((item) => (
              <li key={item.id} className="flex flex-col gap-0.5">
                <span className="flex justify-between gap-2">
                  <span className="truncate">
                    {item.quantity}× {item.product?.name ?? 'Produk'}
                  </span>
                  <span className="shrink-0 font-bold">{formatRupiah(item.price * item.quantity)}</span>
                </span>
                {item.notes && (
                  <span className="text-xs text-zinc-500 dark:text-zinc-400">Catatan: {item.notes}</span>
                )}
              </li>
            ))}
          </ul>
          <div className="mt-4 flex flex-col gap-2 border-t-2 border-black pt-3 text-sm dark:border-white">
            <div className="flex justify-between">
              <span>Subtotal</span>
              <span className="font-bold">{formatRupiah(subtotal)}</span>
            </div>
            <div className="flex justify-between">
              <span>Ongkir{!user && orderType === 'delivery' ? ' (estimasi)' : ''}</span>
              <span className="font-bold">{shipping === 0 ? 'GRATIS' : formatRupiah(shipping)}</span>
            </div>
            <div className="flex justify-between font-display text-lg font-black">
              <span>Total</span>
              <span>{formatRupiah(subtotal + shipping)}</span>
            </div>
          </div>
          {error && (
            <p className="mt-4 border-2 border-red-500 bg-red-50 p-3 text-sm font-semibold text-red-700 dark:bg-red-950 dark:text-red-300">
              {error}
            </p>
          )}
          <button type="submit" disabled={busy || items.length === 0} className="btn btn-accent mt-5 w-full">
            {busy ? 'Memproses...' : `Bayar Sekarang • ${formatRupiah(subtotal + shipping)}`}
          </button>
          {!user && (
            <p className="mt-3 text-center text-xs text-zinc-500">
              Sudah punya akun?{' '}
              <Link to="/login" className="font-bold underline">
                Masuk sebagai member
              </Link>{' '}
              untuk riwayat permanen.
            </p>
          )}
        </aside>
      </form>

      <PaymentModal
        order={createdOrder}
        open={payOpen}
        guest={!user}
        onClose={handlePayClose}
        onPaid={handlePaid}
      />
    </div>
  )
}
