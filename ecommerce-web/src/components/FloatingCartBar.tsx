import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { ArrowRight, Minus, Plus, ShoppingCart, Trash2, X } from 'lucide-react'
import { useCart } from '../context/CartContext'
import { formatRupiah } from '../utils/format'

/**
 * Floating Cart Card (sticky checkout bar) — muncul persisten di bawah layar
 * begitu keranjang berisi minimal 1 item (tamu localStorage maupun member
 * /cart). Disembunyikan saat keranjang kosong dan saat sudah di /checkout
 * (agar tidak menduplikasi tombol checkout di sana).
 */
export default function FloatingCartBar() {
  const { cart, busy, updateQty, removeItem } = useCart()
  const location = useLocation()
  const navigate = useNavigate()
  const [drawerOpen, setDrawerOpen] = useState(false)

  const items = cart?.cart.items ?? []
  const totalItems = cart?.total_items ?? 0
  const subtotal = cart?.subtotal ?? 0

  if (totalItems === 0 || items.length === 0) return null
  if (location.pathname === '/checkout') return null

  const shownNames = items.slice(0, 2).map((i) => i.product?.name ?? 'Produk')
  const extraCount = items.length - shownNames.length

  const changeQty = async (itemId: string, qty: number) => {
    if (qty < 1) return
    try {
      await updateQty(itemId, qty)
    } catch {
      // Error sudah tercermin lewat state keranjang; drawer tetap terbuka.
    }
  }

  const handleRemove = async (itemId: string) => {
    try {
      await removeItem(itemId)
    } catch {
      // Sama seperti di atas: abaikan, state keranjang yang berbicara.
    }
  }

  return (
    <>
      <div
        data-testid="floating-cart-bar"
        className="animate-rise fixed bottom-4 left-4 right-4 z-50 mx-auto max-w-3xl"
      >
        <div className="flex flex-col gap-3 border-2 border-black bg-black p-3 text-white shadow-[4px_4px_0px_0px_rgba(0,0,0,0.35)] sm:flex-row sm:items-center sm:gap-4 sm:p-4 dark:border-black dark:bg-white dark:text-black dark:shadow-[4px_4px_0px_0px_#fff]">
          <div className="flex min-w-0 flex-1 items-center gap-3">
            <span className="chip shrink-0 bg-yolk !text-black" data-testid="floating-cart-count">
              [ {totalItems} ITEM ]
            </span>
            <div className="min-w-0">
              <p className="truncate text-sm font-bold" data-testid="floating-cart-items">
                {shownNames.join(', ')}
                {extraCount > 0 ? ` +${extraCount} menu lagi` : ''}
              </p>
              <p className="font-display text-lg font-black leading-tight" data-testid="floating-cart-total">
                {formatRupiah(subtotal)}
              </p>
            </div>
          </div>
          <div className="flex shrink-0 gap-2">
            <button
              type="button"
              onClick={() => setDrawerOpen(true)}
              className="inline-flex items-center justify-center gap-2 border-2 border-white px-3 py-2 font-display text-xs font-black uppercase tracking-wide text-white transition-all hover:bg-white/10 active:translate-x-[2px] active:translate-y-[2px] dark:border-black dark:text-black dark:hover:bg-black/10"
            >
              <ShoppingCart className="h-4 w-4" /> Lihat Keranjang
            </button>
            <button
              type="button"
              onClick={() => navigate('/checkout')}
              className="btn btn-accent btn-sm !px-4 !py-2.5"
            >
              Checkout Sekarang <ArrowRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>

      {drawerOpen && (
        <div
          className="fixed inset-0 z-[60]"
          role="dialog"
          aria-modal="true"
          aria-label="Keranjang belanja"
          data-testid="cart-drawer"
        >
          <div className="absolute inset-0 bg-black/60" onClick={() => setDrawerOpen(false)} />
          <aside className="animate-slide-in-right absolute right-0 top-0 flex h-full w-full max-w-md flex-col overflow-y-auto border-l-4 border-black bg-white dark:border-white dark:bg-coal">
            <div className="flex items-center justify-between gap-3 border-b-2 border-black p-4 dark:border-white">
              <h2 className="font-display text-lg font-black uppercase">Keranjang</h2>
              <button
                onClick={() => setDrawerOpen(false)}
                className="btn btn-ghost btn-sm !px-2"
                aria-label="Tutup keranjang"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="flex flex-1 flex-col gap-3 p-4">
              {items.map((item) => (
                <article
                  key={item.id}
                  className="card flex flex-col gap-2 p-3"
                  data-testid="drawer-item"
                >
                  <p className="font-display text-sm font-black uppercase leading-tight">
                    {item.product?.name ?? 'Produk'}
                  </p>
                  <p className="text-xs text-zinc-600 dark:text-zinc-400">
                    {formatRupiah(item.price)} / item
                    {item.notes ? ` • Catatan: ${item.notes}` : ''}
                  </p>
                  <div className="flex items-center gap-3">
                    <div className="flex items-center border-2 border-black dark:border-white">
                      <button
                        className="px-2.5 py-1.5 hover:bg-zinc-100 disabled:opacity-40 dark:hover:bg-zinc-800"
                        onClick={() => changeQty(item.id, item.quantity - 1)}
                        disabled={busy || item.quantity <= 1}
                        aria-label="Kurangi"
                      >
                        <Minus className="h-4 w-4" />
                      </button>
                      <span className="w-8 text-center font-display font-black">{item.quantity}</span>
                      <button
                        className="px-2.5 py-1.5 hover:bg-zinc-100 disabled:opacity-40 dark:hover:bg-zinc-800"
                        onClick={() => changeQty(item.id, item.quantity + 1)}
                        disabled={busy}
                        aria-label="Tambah"
                      >
                        <Plus className="h-4 w-4" />
                      </button>
                    </div>
                    <button
                      onClick={() => handleRemove(item.id)}
                      disabled={busy}
                      className="btn btn-ghost btn-sm !text-red-600"
                      aria-label="Hapus item"
                    >
                      <Trash2 className="h-4 w-4" /> Hapus
                    </button>
                    <span className="ml-auto font-display text-sm font-black">
                      {formatRupiah(item.price * item.quantity)}
                    </span>
                  </div>
                </article>
              ))}
            </div>

            <div className="border-t-2 border-black p-4 dark:border-white">
              <div className="flex justify-between font-display text-lg font-black">
                <span>Subtotal</span>
                <span data-testid="drawer-subtotal">{formatRupiah(subtotal)}</span>
              </div>
              <p className="mt-1 text-xs text-zinc-500">
                Ongkir dihitung di halaman checkout (gratis ≥ Rp500.000).
              </p>
              <button
                onClick={() => {
                  setDrawerOpen(false)
                  navigate('/checkout')
                }}
                className="btn btn-accent mt-4 w-full"
              >
                Checkout Sekarang <ArrowRight className="h-4 w-4" />
              </button>
              <button
                onClick={() => {
                  setDrawerOpen(false)
                  navigate('/cart')
                }}
                className="btn btn-ghost mt-3 w-full"
              >
                Lihat Keranjang Lengkap
              </button>
            </div>
          </aside>
        </div>
      )}
    </>
  )
}
