import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Receipt, Trash2 } from 'lucide-react'
import { orderApi } from '../api/endpoints'
import type { GuestOrderSummary } from '../api/types'
import { formatDate, formatRupiah, orderTypeLabel } from '../utils/format'
import { Spinner } from '../components/ProtectedRoute'
import { StatusBadge } from '../components/StatusBadge'
import { clearGuestOrders, loadGuestOrders, updateGuestOrderStatus } from '../utils/guestOrders'
import { useAuth } from '../context/AuthContext'

export default function MyOrders() {
  const { user } = useAuth()
  const [orders, setOrders] = useState<GuestOrderSummary[]>(() => loadGuestOrders())
  const [refreshing, setRefreshing] = useState(false)

  // Segarkan status tiap order tamu dari endpoint /track publik.
  useEffect(() => {
    const list = loadGuestOrders()
    if (list.length === 0) return
    setRefreshing(true)
    Promise.allSettled(list.map((o) => orderApi.track(o.order_number))).then((results) => {
      const updated = list.map((o, i) => {
        const r = results[i]
        if (r.status === 'fulfilled') {
          updateGuestOrderStatus(o.order_number, r.value.status)
          return { ...o, status: r.value.status, total: r.value.total }
        }
        return o
      })
      setOrders(updated)
      setRefreshing(false)
    })
  }, [])

  const handleClear = () => {
    clearGuestOrders()
    setOrders([])
  }

  if (refreshing && orders.length === 0) return <Spinner label="Memuat pesanan..." />

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <h1 className="title">Pesanan Saya</h1>
      <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
        Pesanan tamu yang dibuat dari browser ini (tersimpan di localStorage).
        {refreshing ? ' Memperbarui status...' : ''}
      </p>

      {user && (
        <div className="mt-4 border-2 border-black bg-yolk/20 p-4 dark:border-white">
          <p className="text-sm font-bold">
            Kamu login sebagai {user.name} — riwayat pesanan membermu ada di halaman{' '}
            <Link to="/orders" className="underline">
              Pesanan
            </Link>
            .
          </p>
        </div>
      )}

      {orders.length === 0 ? (
        <div className="card mt-6 p-10 text-center">
          <Receipt className="mx-auto h-12 w-12" />
          <p className="mt-4 font-display text-xl font-black uppercase">Belum ada pesanan tamu</p>
          <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
            Pesan menu tanpa login, nomor ordermu akan muncul di sini.
          </p>
          <Link to="/" className="btn btn-primary btn-sm mt-6">
            Lihat Menu
          </Link>
        </div>
      ) : (
        <>
          <div className="mt-6 flex flex-col gap-4">
            {orders.map((o) => (
              <Link key={o.order_number} to={`/track/${o.order_number}`} className="card block p-4 transition-all hover:-translate-y-0.5">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="font-display text-base font-black">{o.order_number}</span>
                  <StatusBadge status={o.status} />
                </div>
                <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-sm text-zinc-600 dark:text-zinc-400">
                  <span>{o.customer_name}</span>
                  <span>{orderTypeLabel(o.order_type)}</span>
                  <span>{formatDate(o.created_at)}</span>
                </div>
                <p className="mt-2 font-display text-lg font-black">{formatRupiah(o.total)}</p>
              </Link>
            ))}
          </div>
          <button onClick={handleClear} className="btn btn-ghost btn-sm mt-6">
            <Trash2 className="h-4 w-4" /> Hapus Riwayat Tamu di Browser Ini
          </button>
        </>
      )}
    </div>
  )
}
