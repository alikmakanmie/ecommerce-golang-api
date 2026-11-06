import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ArrowLeft, Minus, Plus, ShoppingCart } from 'lucide-react'
import { catalogApi } from '../api/endpoints'
import { apiErrorMessage } from '../api/client'
import type { Product } from '../api/types'
import { formatRupiah } from '../utils/format'
import { ProductVisual } from '../components/ProductVisual'
import { Spinner } from '../components/ProtectedRoute'
import { useCart } from '../context/CartContext'

export default function ProductDetail() {
  const { id } = useParams<{ id: string }>()
  const [product, setProduct] = useState<Product | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [qty, setQty] = useState(1)
  const [added, setAdded] = useState(false)
  const [addError, setAddError] = useState('')
  const { addItem, busy } = useCart()

  useEffect(() => {
    if (!id) return
    setLoading(true)
    catalogApi
      .getProduct(id)
      .then(setProduct)
      .catch((err) => setError(apiErrorMessage(err, 'Produk tidak ditemukan')))
      .finally(() => setLoading(false))
  }, [id])

  if (loading) return <Spinner label="Memuat produk..." />
  if (error || !product) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-16 text-center">
        <p className="font-display text-2xl font-black uppercase text-red-600">{error || 'Produk tidak ditemukan'}</p>
        <Link to="/" className="btn btn-primary btn-sm mt-6">
          <ArrowLeft className="h-4 w-4" /> Kembali ke Menu
        </Link>
      </div>
    )
  }

  const handleAdd = async () => {
    setAddError('')
    try {
      // Tamu boleh menambah: masuk keranjang lokal (localStorage).
      await addItem(product.id, qty, product)
      setAdded(true)
      window.setTimeout(() => setAdded(false), 2000)
    } catch (e) {
      setAddError(e instanceof Error ? e.message : 'Gagal menambah')
    }
  }

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <Link to="/" className="btn btn-ghost btn-sm mb-6">
        <ArrowLeft className="h-4 w-4" /> Semua Menu
      </Link>
      <div className="card grid overflow-hidden md:grid-cols-2">
        <div className="h-72 border-b-2 border-black dark:border-white md:h-auto md:border-b-0 md:border-r-2">
          <ProductVisual product={product} size="lg" />
        </div>
        <div className="flex flex-col gap-4 p-6">
          {product.category && <span className="chip w-fit bg-bone dark:bg-carbon">{product.category.name}</span>}
          <h1 className="font-display text-3xl font-black uppercase leading-none sm:text-4xl">{product.name}</h1>
          <p className="text-base text-zinc-600 dark:text-zinc-400">{product.description}</p>
          <p className="font-display text-3xl font-black">{formatRupiah(product.price)}</p>
          <div className="grid grid-cols-2 gap-2 text-sm">
            <div className="border-2 border-black p-2 dark:border-white">
              <span className="block font-display text-[10px] font-black uppercase text-zinc-500">Stok</span>
              <span className="font-bold">{product.stock > 0 ? `${product.stock} porsi` : 'Habis'}</span>
            </div>
            <div className="border-2 border-black p-2 dark:border-white">
              <span className="block font-display text-[10px] font-black uppercase text-zinc-500">Berat</span>
              <span className="font-bold">{product.weight_gram} gram</span>
            </div>
            <div className="border-2 border-black p-2 dark:border-white">
              <span className="block font-display text-[10px] font-black uppercase text-zinc-500">SKU</span>
              <span className="font-bold">{product.sku}</span>
            </div>
            <div className="border-2 border-black p-2 dark:border-white">
              <span className="block font-display text-[10px] font-black uppercase text-zinc-500">Status</span>
              <span className="font-bold">{product.is_active ? 'Tersedia' : 'Nonaktif'}</span>
            </div>
          </div>
          <div className="mt-2 flex flex-wrap items-center gap-3">
            <div className="flex items-center border-2 border-black dark:border-white">
              <button
                className="px-3 py-2 hover:bg-zinc-100 dark:hover:bg-zinc-800 disabled:opacity-40"
                onClick={() => setQty((q) => Math.max(1, q - 1))}
                disabled={qty <= 1}
                aria-label="Kurangi"
              >
                <Minus className="h-4 w-4" />
              </button>
              <span className="w-10 text-center font-display text-lg font-black">{qty}</span>
              <button
                className="px-3 py-2 hover:bg-zinc-100 dark:hover:bg-zinc-800 disabled:opacity-40"
                onClick={() => setQty((q) => Math.min(product.stock, q + 1))}
                disabled={qty >= product.stock}
                aria-label="Tambah"
              >
                <Plus className="h-4 w-4" />
              </button>
            </div>
            <button onClick={handleAdd} disabled={busy || product.stock <= 0} className="btn btn-accent flex-1">
              <ShoppingCart className="h-4 w-4" />
              {added ? '✓ Ditambahkan!' : `Tambah • ${formatRupiah(product.price * qty)}`}
            </button>
          </div>
          {addError && <p className="text-sm font-semibold text-red-600">{addError}</p>}
        </div>
      </div>
    </div>
  )
}
