import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ArrowRight, Minus, Plus, ShoppingCart, Trash2 } from 'lucide-react'
import { useCart } from '../context/CartContext'
import { formatRupiah, shippingCost } from '../utils/format'
import { Spinner } from '../components/ProtectedRoute'
import { ProductVisual } from '../components/ProductVisual'

export default function Cart() {
  const { cart, loading, busy, updateQty, removeItem } = useCart()
  const navigate = useNavigate()
  const [error, setError] = useState('')

  if (loading) return <Spinner label="Memuat keranjang..." />

  const items = cart?.cart.items ?? []
  const subtotal = cart?.subtotal ?? 0
  const shipping = items.length > 0 ? shippingCost(subtotal) : 0

  const changeQty = async (itemId: string, qty: number) => {
    if (qty < 1) return
    setError('')
    try {
      await updateQty(itemId, qty)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Gagal mengubah kuantitas')
    }
  }

  const handleRemove = async (itemId: string) => {
    setError('')
    try {
      await removeItem(itemId)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Gagal menghapus item')
    }
  }

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <h1 className="title">Keranjang</h1>
      <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
        {cart?.total_items ?? 0} item • Subtotal {formatRupiah(subtotal)}
      </p>

      {error && (
        <p className="mt-4 border-2 border-red-500 bg-red-50 p-3 text-sm font-semibold text-red-700 dark:bg-red-950 dark:text-red-300">
          {error}
        </p>
      )}

      {items.length === 0 ? (
        <div className="card mt-6 p-10 text-center">
          <ShoppingCart className="mx-auto h-12 w-12" />
          <p className="mt-4 font-display text-xl font-black uppercase">Keranjang masih kosong</p>
          <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
            Yuk, isi dengan menu favoritmu dari Slaras.
          </p>
          <Link to="/" className="btn btn-primary btn-sm mt-6">
            Lihat Menu
          </Link>
        </div>
      ) : (
        <div className="mt-6 grid gap-6 lg:grid-cols-3">
          <div className="flex flex-col gap-4 lg:col-span-2">
            {items.map((item) => (
              <article key={item.id} className="card flex gap-4 p-4">
                <Link
                  to={`/products/${item.product_id}`}
                  className="h-24 w-24 shrink-0 overflow-hidden border-2 border-black dark:border-white"
                >
                  {item.product ? (
                    <ProductVisual product={item.product} size="sm" />
                  ) : (
                    <div className="flex h-full w-full items-center justify-center bg-zinc-200 dark:bg-zinc-800">
                      <ShoppingCart className="h-8 w-8" />
                    </div>
                  )}
                </Link>
                <div className="flex flex-1 flex-col gap-1">
                  <Link
                    to={`/products/${item.product_id}`}
                    className="font-display text-base font-black uppercase leading-tight hover:underline"
                  >
                    {item.product?.name ?? 'Produk'}
                  </Link>
                  <p className="text-sm text-zinc-600 dark:text-zinc-400">
                    {formatRupiah(item.price)} / item
                  </p>
                  {item.notes && (
                    <p className="text-xs font-semibold text-zinc-500 dark:text-zinc-400">
                      Catatan: {item.notes}
                    </p>
                  )}
                  <div className="mt-auto flex flex-wrap items-center gap-3">
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
                      title="Hapus item"
                    >
                      <Trash2 className="h-4 w-4" /> Hapus
                    </button>
                    <span className="ml-auto font-display text-base font-black">
                      {formatRupiah(item.price * item.quantity)}
                    </span>
                  </div>
                </div>
              </article>
            ))}
          </div>

          <aside className="card h-fit p-5">
            <h2 className="font-display text-lg font-black uppercase">Ringkasan</h2>
            <div className="mt-4 flex flex-col gap-2 text-sm">
              <div className="flex justify-between">
                <span>Subtotal</span>
                <span className="font-bold">{formatRupiah(subtotal)}</span>
              </div>
              <div className="flex justify-between">
                <span>Ongkir</span>
                <span className="font-bold">
                  {shipping === 0 ? 'GRATIS' : formatRupiah(shipping)}
                </span>
              </div>
              {shipping > 0 && (
                <p className="text-xs text-zinc-500">
                  Gratis ongkir untuk pembelian ≥ Rp500.000 (kurang {formatRupiah(500000 - subtotal)} lagi)
                </p>
              )}
              <div className="mt-2 flex justify-between border-t-2 border-black pt-3 font-display text-lg font-black dark:border-white">
                <span>Total</span>
                <span>{formatRupiah(subtotal + shipping)}</span>
              </div>
            </div>
            <button onClick={() => navigate('/checkout')} className="btn btn-accent mt-5 w-full">
              Checkout <ArrowRight className="h-4 w-4" />
            </button>
            <Link to="/" className="btn btn-ghost mt-3 w-full">
              Lanjut Belanja
            </Link>
          </aside>
        </div>
      )}
    </div>
  )
}
