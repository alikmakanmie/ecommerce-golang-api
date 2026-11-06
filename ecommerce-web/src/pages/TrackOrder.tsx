import { useCallback, useEffect, useRef, useState, type FormEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Printer, RefreshCw, Search, UtensilsCrossed } from 'lucide-react'
import { orderApi } from '../api/endpoints'
import { apiErrorMessage } from '../api/client'
import type { Order } from '../api/types'
import { formatDate, formatRupiah, orderTypeLabel } from '../utils/format'
import { Spinner } from '../components/ProtectedRoute'
import { StatusBadge } from '../components/StatusBadge'
import { updateGuestOrderStatus } from '../utils/guestOrders'

const POLLABLE: Order['status'][] = ['pending', 'paid', 'processing']

export default function TrackOrder() {
  const { order_number } = useParams<{ order_number: string }>()
  const navigate = useNavigate()
  const [order, setOrder] = useState<Order | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const timerRef = useRef<number | null>(null)

  const fetchOrder = useCallback(
    async (number: string, silent = false) => {
      if (!silent) setLoading(true)
      try {
        const data = await orderApi.track(number)
        setOrder(data)
        setError('')
        updateGuestOrderStatus(data.order_number, data.status)
        return data
      } catch (err) {
        if (!silent) {
          setOrder(null)
          setError(apiErrorMessage(err, 'Pesanan tidak ditemukan'))
        }
        return null
      } finally {
        if (!silent) setLoading(false)
      }
    },
    [],
  )

  useEffect(() => {
    if (!order_number) return
    fetchOrder(order_number)
  }, [order_number, fetchOrder])

  // Poll status live tiap 5 detik selama pesanan masih berjalan.
  useEffect(() => {
    if (!order || !POLLABLE.includes(order.status)) return
    timerRef.current = window.setInterval(() => {
      fetchOrder(order.order_number, true)
    }, 5000)
    return () => {
      if (timerRef.current) window.clearInterval(timerRef.current)
    }
  }, [order?.order_number, order?.status, fetchOrder])

  const submitSearch = (e: FormEvent) => {
    e.preventDefault()
    const number = search.trim().toUpperCase()
    if (number) navigate(`/track/${number}`)
  }

  if (loading) return <Spinner label="Mencari pesanan..." />

  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <h1 className="title">Lacak Pesanan</h1>
      <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
        Status pesanan diperbarui otomatis setiap beberapa detik.
      </p>

      <form onSubmit={submitSearch} className="mt-6 flex gap-2">
        <input
          className="input"
          placeholder="Masukkan nomor order, contoh: ORD-20261106-XXXXXX"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          aria-label="Nomor order"
        />
        <button type="submit" className="btn btn-primary shrink-0">
          <Search className="h-4 w-4" /> Lacak
        </button>
      </form>

      {error && !order && (
        <div className="mt-6 border-2 border-red-500 bg-red-50 p-5 text-center dark:bg-red-950">
          <p className="font-display text-lg font-black uppercase text-red-700 dark:text-red-300">
            {error}
          </p>
          <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
            Periksa lagi nomor ordernya, atau pesan menu baru dari katalog.
          </p>
          <Link to="/" className="btn btn-primary btn-sm mt-4">
            Lihat Menu
          </Link>
        </div>
      )}

      {order && (
        <article className="card mt-6 p-6">
          <div className="flex flex-wrap items-start justify-between gap-3 border-b-2 border-black pb-4 dark:border-white">
            <div className="flex items-center gap-3">
              <span className="flex h-11 w-11 items-center justify-center border-2 border-black bg-yolk text-black dark:border-white">
                <UtensilsCrossed className="h-6 w-6" strokeWidth={2.5} />
              </span>
              <div>
                <p className="font-display text-lg font-black uppercase leading-none">Slaras Nusa Foodlab</p>
                <p className="mt-1 text-xs text-zinc-500">Struk Digital • {formatDate(order.created_at)}</p>
              </div>
            </div>
            <StatusBadge status={order.status} />
          </div>

          <dl className="mt-4 grid gap-2 text-sm sm:grid-cols-2">
            <div>
              <dt className="label !mb-0">Nomor Order</dt>
              <dd className="font-display text-base font-black">{order.order_number}</dd>
            </div>
            <div>
              <dt className="label !mb-0">Pemesan</dt>
              <dd className="font-bold">{order.customer_name ?? order.shipping_name}</dd>
            </div>
            <div>
              <dt className="label !mb-0">Jenis Pesanan</dt>
              <dd className="font-bold">{orderTypeLabel(order.order_type)}</dd>
            </div>
            {order.table_number && (
              <div>
                <dt className="label !mb-0">Nomor Meja</dt>
                <dd className="font-display text-base font-black">{order.table_number}</dd>
              </div>
            )}
            <div>
              <dt className="label !mb-0">Pembayaran</dt>
              <dd className="font-bold uppercase">
                {order.payment_method} • {order.payment?.status ?? order.status}
              </dd>
            </div>
            {order.shipping_address_text && (
              <div>
                <dt className="label !mb-0">Alamat / Keterangan</dt>
                <dd className="font-bold">{order.shipping_address_text}</dd>
              </div>
            )}
          </dl>

          <ul className="mt-5 flex flex-col gap-2 border-t-2 border-dashed border-zinc-400 pt-4 text-sm">
            {order.items.map((item) => (
              <li key={item.id} className="flex flex-col gap-0.5">
                <span className="flex justify-between gap-2">
                  <span>
                    {item.quantity}× {item.product_name}
                  </span>
                  <span className="shrink-0 font-bold">{formatRupiah(item.subtotal)}</span>
                </span>
                {item.notes && (
                  <span className="text-xs text-zinc-500 dark:text-zinc-400">
                    Catatan: {item.notes}
                  </span>
                )}
              </li>
            ))}
          </ul>

          <div className="mt-4 flex flex-col gap-2 border-t-2 border-black pt-3 text-sm dark:border-white">
            <div className="flex justify-between">
              <span>Subtotal</span>
              <span className="font-bold">{formatRupiah(order.subtotal)}</span>
            </div>
            <div className="flex justify-between">
              <span>Ongkir</span>
              <span className="font-bold">{order.shipping_cost === 0 ? 'GRATIS' : formatRupiah(order.shipping_cost)}</span>
            </div>
            <div className="flex justify-between font-display text-lg font-black">
              <span>Total</span>
              <span>{formatRupiah(order.total)}</span>
            </div>
          </div>

          {order.notes && (
            <p className="mt-4 border-2 border-dashed border-zinc-400 p-3 text-sm">
              <span className="font-display text-xs font-black uppercase">Catatan pesanan: </span>
              {order.notes}
            </p>
          )}

          {order.status === 'pending' && (
            <p className="mt-4 border-2 border-yolk bg-yolk/20 p-3 text-sm font-semibold">
              Menunggu pembayaran — selesaikan pembayaran QRIS/VA dari modal pembayaran, atau
              buka kembali halaman ini setelah membayar.
            </p>
          )}

          <div className="mt-6 flex flex-wrap gap-3">
            <button
              onClick={() => fetchOrder(order.order_number)}
              className="btn btn-ghost btn-sm"
            >
              <RefreshCw className="h-4 w-4" /> Refresh Status
            </button>
            <button onClick={() => window.print()} className="btn btn-primary btn-sm">
              <Printer className="h-4 w-4" /> Cetak Struk
            </button>
            <Link to="/" className="btn btn-ghost btn-sm">
              Pesan Lagi
            </Link>
          </div>
        </article>
      )}
    </div>
  )
}
