import { Link } from 'react-router-dom'
import { Plus, ShoppingCart } from 'lucide-react'
import type { Product } from '../api/types'
import { formatRupiah } from '../utils/format'
import { ProductVisual } from './ProductVisual'
import { useCart } from '../context/CartContext'
import { useState } from 'react'

export default function ProductCard({ product }: { product: Product }) {
  const { addItem, busy } = useCart()
  const [added, setAdded] = useState(false)
  const [error, setError] = useState('')

  const handleAdd = async () => {
    setError('')
    try {
      // Tamu boleh menambah: masuk keranjang lokal (localStorage).
      // Member: tersimpan ke /cart API. Snapshot produk ikut disimpan
      // agar keranjang tamu tetap menampilkan nama & harga.
      await addItem(product.id, 1, product)
      setAdded(true)
      window.setTimeout(() => setAdded(false), 1500)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Gagal menambah')
    }
  }

  return (
    <article className="card flex flex-col overflow-hidden">
      <Link to={`/products/${product.id}`} className="block h-44 border-b-2 border-black dark:border-white">
        <ProductVisual product={product} />
      </Link>
      <div className="flex flex-1 flex-col gap-2 p-4">
        <div className="flex items-start justify-between gap-2">
          <Link to={`/products/${product.id}`}>
            <h3 className="font-display text-base font-black uppercase leading-tight hover:underline">
              {product.name}
            </h3>
          </Link>
          {product.category && (
            <span className="chip shrink-0 bg-bone dark:bg-carbon">{product.category.name}</span>
          )}
        </div>
        <p className="line-clamp-2 text-sm text-zinc-600 dark:text-zinc-400">{product.description}</p>
        <div className="mt-auto flex items-end justify-between gap-2 pt-2">
          <div>
            <p className="font-display text-lg font-black leading-none">{formatRupiah(product.price)}</p>
            <p className={`mt-1 text-xs font-semibold ${product.stock > 0 ? 'text-green-700 dark:text-green-400' : 'text-red-600'}`}>
              {product.stock > 0 ? `Stok: ${product.stock}` : 'Stok habis'}
            </p>
          </div>
          <button
            onClick={handleAdd}
            disabled={busy || product.stock <= 0}
            className="btn btn-primary btn-sm"
            title="Tambah ke keranjang"
          >
            {added ? (
              <>✓ Ditambah</>
            ) : (
              <>
                <Plus className="h-4 w-4" /> <ShoppingCart className="h-4 w-4" />
              </>
            )}
          </button>
        </div>
        {error && <p className="text-xs font-semibold text-red-600">{error}</p>}
      </div>
    </article>
  )
}
