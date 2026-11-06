import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { PackageSearch, XCircle } from 'lucide-react'
import { orderApi } from '../api/endpoints'
import { apiErrorMessage } from '../api/client'
import type { Order } from '../api/types'
import { formatDate, formatRupiah } from '../utils/format'
import { Spinner } from '../components/ProtectedRoute'
import { StatusBadge } from '../components/StatusBadge'

export default function Orders() {
  const [orders, setOrders] = useState<Order[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [cancelling, setCancelling] = useState<string | null>(null)

  const fetchOrders = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const res = await orderApi.list(1, 50)
      setOrders(res.orders)
    } catch (err) {
      setError(apiErrorMessage(err, 'Gagal memuat pesanan'))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchOrders()
  }, [fetchOrders])

  const handleCancel = async (id: string) => {
    if (!window.confirm('Batalkan pesanan ini? Stok akan dikembalikan.')) return
    setCancelling(id)
    try {
      await orderApi.cancel(id)
      await fetchOrders()
    } catch (err) {
      setError(apiErrorMessage(err, 'Gagal membatalkan pesanan'))
    } finally {
      setCancelling(null)
    }
  }

  if (loading) return <Spinner label="Memuat pesanan..." />

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <h1 className="title">Pesanan Saya</h1>
      <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
        {orders.length} pesanan • Status: pending → paid → processing → shipped → completed
      </p>
      {error && (
        <p className="mt-4 border-2 border-red-500 bg-red-50 p-3 text-sm font-semibold text-red-700 dark:bg-red-950 dark:text-red-300">
          {error}
        </p>
      )}
      {orders.length === 0 ? (
        <div className="card mt-6 p-10 text-center">
          <PackageSearch className="mx-auto h-12 w-12" />
          <p className="mt-4 font-display text-xl font-black uppercase">Belum ada pesanan</p>
          <Link to="/" className="btn btn-primary btn-sm mt-6">
            Pesan Menu Sekarang
          </Link>
        </div>
      ) : (
        <div className="mt-6 flex flex-col gap-4">
          {orders.map((o) => (
            <article key={o.id} className="card p-5">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <Link
                    to={`/orders/${o.id}`}
                    className="font-display text-lg font-black uppercase hover:underline"
                  >
                    {o.order_number}
                  </Link>
                  <p className="text-sm text-zinc-600 dark:text-zinc-400">
                    {formatDate(o.created_at)} • {o.items.length} item • {o.payment_method.toUpperCase()}
                  </p>
                </div>
                <StatusBadge status={o.status} />
              </div>
              <ul className="mt-3 flex flex-col gap-1 text-sm">
                {o.items.map((item) => (
                  <li key={item.id} className="flex justify-between gap-2">
                    <span className="truncate">
                      {item.quantity}× {item.product_name}
                    </span>
                    <span className="shrink-0 font-bold">{formatRupiah(item.subtotal)}</span>
                  </li>
                ))}
              </ul>
              <div className="mt-3 flex flex-wrap items-center justify-between gap-3 border-t-2 border-black pt-3 dark:border-white">
                <span className="font-display text-lg font-black">Total {formatRupiah(o.total)}</span>
                <div className="flex gap-2">
                  <Link to={`/orders/${o.id}`} className="btn btn-ghost btn-sm">
                    Detail
                  </Link>
                  {(o.status === 'pending' || o.status === 'paid') && (
                    <button
                      onClick={() => handleCancel(o.id)}
                      disabled={cancelling === o.id}
                      className="btn btn-danger btn-sm"
                    >
                      <XCircle className="h-4 w-4" /> {cancelling === o.id ? 'Membatalkan...' : 'Batalkan'}
                    </button>
                  )}
                </div>
              </div>
            </article>
          ))}
        </div>
      )}
    </div>
  )
}
