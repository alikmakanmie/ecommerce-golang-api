import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ArrowLeft, XCircle } from 'lucide-react'
import { orderApi } from '../api/endpoints'
import { apiErrorMessage } from '../api/client'
import type { Order } from '../api/types'
import { formatDate, formatRupiah } from '../utils/format'
import { Spinner } from '../components/ProtectedRoute'
import { StatusBadge } from '../components/StatusBadge'
import { PaymentModal } from '../components/PaymentModal'

export default function OrderDetail() {
  const { id } = useParams<{ id: string }>()
  const [order, setOrder] = useState<Order | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [payOpen, setPayOpen] = useState(false)
  const [cancelling, setCancelling] = useState(false)

  const fetchOrder = useCallback(async () => {
    if (!id) return
    setLoading(true)
    try {
      const data = await orderApi.get(id)
      setOrder(data)
    } catch (err) {
      setError(apiErrorMessage(err, 'Pesanan tidak ditemukan'))
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    fetchOrder()
  }, [fetchOrder])

  if (loading) return <Spinner label="Memuat pesanan..." />
  if (error || !order) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-16 text-center">
        <p className="font-display text-2xl font-black uppercase text-red-600">{error || 'Pesanan tidak ditemukan'}</p>
        <Link to="/orders" className="btn btn-primary btn-sm mt-6">
          <ArrowLeft className="h-4 w-4" /> Kembali ke Pesanan
        </Link>
      </div>
    )
  }

  const handleCancel = async () => {
    if (!window.confirm('Batalkan pesanan ini?')) return
    setCancelling(true)
    try {
      const updated = await orderApi.cancel(order.id)
      setOrder(updated)
    } catch (err) {
      setError(apiErrorMessage(err, 'Gagal membatalkan pesanan'))
    } finally {
      setCancelling(false)
    }
  }

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <Link to="/orders" className="btn btn-ghost btn-sm mb-6">
        <ArrowLeft className="h-4 w-4" /> Semua Pesanan
      </Link>
      <div className="card p-6">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 className="font-display text-2xl font-black uppercase sm:text-3xl">{order.order_number}</h1>
            <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">{formatDate(order.created_at)}</p>
          </div>
          <StatusBadge status={order.status} />
        </div>

        <div className="mt-6 grid gap-6 md:grid-cols-2">
          <section>
            <h2 className="font-display text-sm font-black uppercase">Item Pesanan</h2>
            <ul className="mt-2 flex flex-col gap-2">
              {order.items.map((item) => (
                <li key={item.id} className="flex justify-between gap-2 border-2 border-zinc-200 p-3 text-sm dark:border-zinc-700">
                  <span>
                    {item.quantity}× {item.product_name}
                    <span className="block text-xs text-zinc-500">@ {formatRupiah(item.price)}</span>
                  </span>
                  <span className="font-bold">{formatRupiah(item.subtotal)}</span>
                </li>
              ))}
            </ul>
            <div className="mt-3 flex flex-col gap-1 border-t-2 border-black pt-3 text-sm dark:border-white">
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
          </section>

          <section className="flex flex-col gap-4">
            <div>
              <h2 className="font-display text-sm font-black uppercase">Pengiriman</h2>
              <p className="mt-2 text-sm">
                <strong>{order.shipping_name}</strong> • {order.shipping_phone}
              </p>
              <p className="text-sm text-zinc-600 dark:text-zinc-400">{order.shipping_address_text}</p>
              {order.notes && <p className="mt-1 text-sm italic">Catatan: {order.notes}</p>}
            </div>
            <div>
              <h2 className="font-display text-sm font-black uppercase">Pembayaran</h2>
              <p className="mt-2 text-sm">
                Metode: <strong className="uppercase">{order.payment_method}</strong>
              </p>
              {order.payment && (
                <>
                  <p className="text-sm">
                    Transaksi: <code className="break-all text-xs">{order.payment.transaction_id}</code>
                  </p>
                  <p className="text-sm">
                    Status bayar: <strong>{order.payment.status}</strong>
                  </p>
                  {order.payment.qr_string && (
                    <p className="text-sm">
                      QR: <code className="break-all text-xs">{order.payment.qr_string}</code>
                    </p>
                  )}
                </>
              )}
            </div>
            <div className="mt-auto flex flex-wrap gap-2">
              {order.status === 'pending' && (
                <button onClick={() => setPayOpen(true)} className="btn btn-accent btn-sm">
                  Bayar / Simulasi
                </button>
              )}
              {(order.status === 'pending' || order.status === 'paid') && (
                <button onClick={handleCancel} disabled={cancelling} className="btn btn-danger btn-sm">
                  <XCircle className="h-4 w-4" /> {cancelling ? 'Membatalkan...' : 'Batalkan Pesanan'}
                </button>
              )}
            </div>
          </section>
        </div>
      </div>

      <PaymentModal
        order={order}
        open={payOpen}
        onClose={() => {
          setPayOpen(false)
          fetchOrder()
        }}
        // Jangan tutup modal di sini: biarkan modal menampilkan "Lunas!",
        // halaman diperbarui saat modal ditutup (onClose di atas).
        onPaid={() => {}}
      />
    </div>
  )
}
