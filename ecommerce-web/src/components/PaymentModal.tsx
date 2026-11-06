import { useEffect, useRef, useState } from 'react'
import { CheckCircle2, ExternalLink, Loader2, QrCode, RefreshCw } from 'lucide-react'
import { QRCodeSVG } from 'qrcode.react'
import { orderApi } from '../api/endpoints'
import type { Order } from '../api/types'
import { formatRupiah } from '../utils/format'
import { Modal } from './ProtectedRoute'
import { StatusBadge } from './StatusBadge'

/**
 * Modal pembayaran mock gateway.
 *
 * Backend TIDAK menyediakan jalur simulasi yang aman dari klien: webhook
 * butuh HMAC WEBHOOK_SECRET yang tidak boleh ada di frontend, dan halaman
 * mock-pay hanya menampilkan info transaksi. Jadi tombol "Simulasikan
 * Pembayaran Sukses" membuka halaman mock-pay di tab baru, lalu modal
 * mem-polling status order sampai berubah dari pending -> paid.
 * Order tamu (guest) dipantau lewat endpoint publik /orders/track/:order_number,
 * order member lewat /orders/:id (butuh JWT).
 */
export function PaymentModal({
  order,
  open,
  onClose,
  onPaid,
  guest = false,
}: {
  order: Order | null
  open: boolean
  onClose: () => void
  onPaid: (order: Order) => void
  guest?: boolean
}) {
  const [status, setStatus] = useState<Order['status']>('pending')
  const [checking, setChecking] = useState(false)
  const [simulating, setSimulating] = useState(false)
  const [error, setError] = useState('')
  const timerRef = useRef<number | null>(null)

  useEffect(() => {
    if (order) {
      setStatus(order.status)
      setSimulating(false)
      setError('')
    }
  }, [order, open])

  const checkStatus = async (silent = false) => {
    if (!order) return
    if (!silent) setChecking(true)
    try {
      // Tamu: endpoint publik /track (tanpa JWT). Member: /orders/:id.
      const fresh = guest
        ? await orderApi.track(order.order_number)
        : await orderApi.get(order.id)
      setStatus(fresh.status)
      if (fresh.status === 'paid') {
        setSimulating(false)
        onPaid(fresh)
      }
      return fresh
    } catch {
      if (!silent) setError('Gagal memeriksa status order')
      return undefined
    } finally {
      if (!silent) setChecking(false)
    }
  }

  // Poll otomatis tiap 4 detik selama modal terbuka & order masih pending.
  useEffect(() => {
    if (!open || !order || status !== 'pending') return
    timerRef.current = window.setInterval(() => {
      checkStatus(true)
    }, 4000)
    return () => {
      if (timerRef.current) window.clearInterval(timerRef.current)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, order?.id, status])

  if (!order) return null
  const payment = order.payment
  const paid = status === 'paid'

  const startSimulation = () => {
    setSimulating(true)
    setError('')
    if (payment?.payment_url) {
      window.open(payment.payment_url, '_blank', 'noopener,noreferrer')
    }
    // Cek segera + biarkan polling otomatis berjalan.
    window.setTimeout(() => checkStatus(true), 3000)
  }

  const webhookHint = payment?.payment_url
    ? `BODY='{"transaction_id":"${payment.transaction_id}","transaction_status":"settlement"}'\nSIG=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" | cut -d' ' -f2)\ncurl -X POST ${payment.payment_url.replace(/\/mock-pay\/.*$/, '')}/payments/webhook \\\n  -H 'Content-Type: application/json' -H "X-Webhook-Signature: $SIG" -d "$BODY"`
    : ''

  return (
    <Modal open={open} onClose={onClose} title={paid ? 'Pembayaran Berhasil' : 'Pembayaran'}>
      <div className="flex flex-col gap-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <span className="font-display text-sm font-black uppercase">Order {order.order_number}</span>
          <StatusBadge status={status} />
        </div>

        {paid ? (
          <div className="border-2 border-green-600 bg-green-50 p-4 text-center dark:bg-green-950">
            <CheckCircle2 className="mx-auto h-12 w-12 text-green-600" />
            <p className="mt-2 font-display text-xl font-black uppercase">Lunas!</p>
            <p className="mt-1 text-sm">
              Pembayaran {formatRupiah(order.total)} diterima. Pesananmu sedang disiapkan dapur Slaras.
            </p>
            <button onClick={onClose} className="btn btn-primary mt-4 w-full">
              Lihat Pesanan Saya
            </button>
          </div>
        ) : (
          <>
            <div className="border-2 border-black p-4 dark:border-white">
              <div className="flex justify-between text-sm">
                <span>Total dibayar</span>
                <span className="font-display text-xl font-black">{formatRupiah(order.total)}</span>
              </div>
              <div className="mt-1 flex justify-between text-sm">
                <span>Metode</span>
                <span className="font-bold uppercase">{order.payment_method}</span>
              </div>
              {payment?.expires_at && (
                <div className="mt-1 flex justify-between text-sm">
                  <span>Berlaku sampai</span>
                  <span className="font-bold">
                    {new Date(payment.expires_at).toLocaleString('id-ID')}
                  </span>
                </div>
              )}
            </div>

            {order.payment_method === 'qris' && payment?.qr_string && (
              <div className="flex flex-col items-center gap-2 border-2 border-dashed border-black p-4 text-center dark:border-white">
                <div className="border-2 border-black bg-white p-2 dark:border-white">
                  <QRCodeSVG value={payment.qr_string} size={192} />
                </div>
                <code className="break-all bg-bone px-2 py-1 text-xs font-bold dark:bg-carbon">
                  {payment.qr_string}
                </code>
                <p className="text-xs text-zinc-500">
                  Scan kode QRIS di atas dengan aplikasi pembayaran apa pun (simulasi).
                </p>
              </div>
            )}

            {order.payment_method === 'va' && (
              <div className="flex flex-col items-center gap-2 border-2 border-dashed border-black p-4 text-center dark:border-white">
                <QrCode className="h-10 w-10" />
                <p className="text-sm font-bold">Virtual Account (simulasi)</p>
                <code className="break-all bg-bone px-2 py-1 text-xs font-bold dark:bg-carbon">
                  {payment?.transaction_id ?? '-'}
                </code>
                <p className="text-xs text-zinc-500">
                  Nomor VA & instruksi transfer ada di halaman mock-pay (simulasi).
                </p>
              </div>
            )}

            {payment?.payment_url && (
              <a
                href={payment.payment_url}
                target="_blank"
                rel="noopener noreferrer"
                className="btn btn-ghost w-full"
              >
                <ExternalLink className="h-4 w-4" /> Buka Halaman Mock-Pay
              </a>
            )}

            <button onClick={startSimulation} disabled={simulating} className="btn btn-accent w-full">
              {simulating ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" /> Menunggu Pembayaran...
                </>
              ) : (
                'Simulasikan Pembayaran Sukses'
              )}
            </button>
            <p className="text-center text-xs text-zinc-500">
              Tombol ini membuka halaman mock-pay, lalu status order dipantau otomatis sampai berubah
              menjadi <strong>Dibayar</strong>.
            </p>

            <button onClick={() => checkStatus()} disabled={checking} className="btn btn-ghost w-full">
              <RefreshCw className={`h-4 w-4 ${checking ? 'animate-spin' : ''}`} />
              {checking ? 'Memeriksa...' : 'Saya Sudah Bayar — Cek Status'}
            </button>
            {error && <p className="text-sm font-semibold text-red-600">{error}</p>}

            <details className="border-2 border-zinc-300 p-3 text-xs dark:border-zinc-700">
              <summary className="cursor-pointer font-display font-black uppercase">
                Cara trigger webhook (untuk penguji)
              </summary>
              <p className="mt-2 text-zinc-600 dark:text-zinc-400">
                Webhook butuh HMAC <code>WEBHOOK_SECRET</code> dari server, jadi tidak bisa dipanggil
                dari browser. Jalankan perintah ini di server/terminal:
              </p>
              <pre className="mt-2 overflow-x-auto bg-black p-3 text-[11px] text-green-300">{webhookHint}</pre>
            </details>
          </>
        )}
      </div>
    </Modal>
  )
}
