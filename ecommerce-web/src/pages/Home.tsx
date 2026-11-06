import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { ChevronLeft, ChevronRight, Search, SlidersHorizontal } from 'lucide-react'
import { catalogApi } from '../api/endpoints'
import { apiErrorMessage } from '../api/client'
import type { Category, Product } from '../api/types'
import ProductCard from '../components/ProductCard'
import { Spinner } from '../components/ProtectedRoute'
import { CategoryIcon } from '../components/ProductVisual'

export default function Home() {
  const [categories, setCategories] = useState<Category[]>([])
  const [products, setProducts] = useState<Product[]>([])
  const [total, setTotal] = useState(0)
  const [totalPages, setTotalPages] = useState(1)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [categoryId, setCategoryId] = useState('')
  const [minPriceInput, setMinPriceInput] = useState('')
  const [maxPriceInput, setMaxPriceInput] = useState('')
  const [minPrice, setMinPrice] = useState<number | undefined>(undefined)
  const [maxPrice, setMaxPrice] = useState<number | undefined>(undefined)
  const [page, setPage] = useState(1)

  useEffect(() => {
    catalogApi
      .listCategories()
      .then(setCategories)
      .catch(() => setCategories([]))
  }, [])

  // Debounce pencarian real-time 400ms
  useEffect(() => {
    const t = window.setTimeout(() => {
      setSearch(searchInput.trim())
      setPage(1)
    }, 400)
    return () => window.clearTimeout(t)
  }, [searchInput])

  const fetchProducts = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const res = await catalogApi.listProducts({
        search: search || undefined,
        category_id: categoryId || undefined,
        min_price: minPrice,
        max_price: maxPrice,
        page,
        limit: 12,
      })
      setProducts(res.products)
      setTotal(res.total)
      setTotalPages(res.totalPages)
    } catch (err) {
      setError(apiErrorMessage(err, 'Gagal memuat produk'))
      setProducts([])
    } finally {
      setLoading(false)
    }
  }, [search, categoryId, minPrice, maxPrice, page])

  useEffect(() => {
    fetchProducts()
  }, [fetchProducts])

  const activeCategory = useMemo(
    () => categories.find((c) => c.id === categoryId),
    [categories, categoryId],
  )

  const applyPriceFilter = () => {
    const min = minPriceInput ? Number(minPriceInput) : undefined
    const max = maxPriceInput ? Number(maxPriceInput) : undefined
    setMinPrice(Number.isFinite(min) && (min as number) > 0 ? min : undefined)
    setMaxPrice(Number.isFinite(max) && (max as number) > 0 ? max : undefined)
    setPage(1)
  }

  const resetFilters = () => {
    setSearchInput('')
    setSearch('')
    setCategoryId('')
    setMinPriceInput('')
    setMaxPriceInput('')
    setMinPrice(undefined)
    setMaxPrice(undefined)
    setPage(1)
  }

  return (
    <div>
      {/* Hero */}
      <section className="border-b-4 border-black bg-yolk text-black dark:border-white dark:bg-yolk">
        <div className="mx-auto max-w-7xl px-4 py-10 sm:py-14">
          <p className="chip border-black bg-white !text-black">Kuliner Nusantara • Sejak 2024</p>
          <h1 className="mt-4 max-w-3xl font-display text-4xl font-black uppercase leading-[0.95] tracking-tight sm:text-6xl">
            Makan enak, harga bersahabat.
          </h1>
          <p className="mt-4 max-w-xl text-base font-semibold sm:text-lg">
            Nasi goreng, rendang, kopi gula aren, sampai nasi box untuk acaramu — dimasak segar
            setiap hari oleh Slaras Nusa Foodlab.
          </p>
          <div className="mt-6 flex flex-wrap gap-3">
            <a href="#katalog" className="btn border-black bg-black !text-white shadow-[4px_4px_0px_0px_#fff]">
              Lihat Menu
            </a>
            <Link to="/cart" className="btn border-black bg-white !text-black shadow-[4px_4px_0px_0px_#000]">
              Keranjang Saya
            </Link>
          </div>
        </div>
      </section>

      {/* Katalog */}
      <section id="katalog" className="mx-auto max-w-7xl px-4 py-8">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 className="title">Menu Slaras</h2>
            <p className="mt-1 text-sm font-medium text-zinc-600 dark:text-zinc-400">
              {total} menu tersedia{activeCategory ? ` • ${activeCategory.name}` : ''}
            </p>
          </div>
        </div>

        {/* Filter */}
        <div className="card mt-6 p-4">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
              <input
                className="input !pl-10"
                placeholder="Cari nasi goreng, rendang, kopi..."
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
              />
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <span className="inline-flex items-center gap-1 font-display text-xs font-black uppercase">
                <SlidersHorizontal className="h-4 w-4" /> Harga Rp
              </span>
              <input
                className="input !w-28 !py-2"
                type="number"
                min={0}
                placeholder="Min"
                value={minPriceInput}
                onChange={(e) => setMinPriceInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && applyPriceFilter()}
              />
              <span className="font-black">–</span>
              <input
                className="input !w-28 !py-2"
                type="number"
                min={0}
                placeholder="Max"
                value={maxPriceInput}
                onChange={(e) => setMaxPriceInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && applyPriceFilter()}
              />
              <button onClick={applyPriceFilter} className="btn btn-primary btn-sm">
                Terapkan
              </button>
              {(search || categoryId || minPrice !== undefined || maxPrice !== undefined) && (
                <button onClick={resetFilters} className="btn btn-ghost btn-sm">
                  Reset
                </button>
              )}
            </div>
          </div>
          <div className="mt-4 flex flex-wrap gap-2">
            <button
              onClick={() => {
                setCategoryId('')
                setPage(1)
              }}
              className={`chip cursor-pointer transition-all ${categoryId === '' ? 'bg-black text-white dark:bg-white dark:text-black' : 'bg-bone hover:bg-zinc-200 dark:bg-carbon'}`}
            >
              Semua
            </button>
            {categories.map((c) => (
              <button
                key={c.id}
                onClick={() => {
                  setCategoryId(c.id)
                  setPage(1)
                }}
                className={`chip cursor-pointer transition-all ${categoryId === c.id ? 'bg-black text-white dark:bg-white dark:text-black' : 'bg-bone hover:bg-zinc-200 dark:bg-carbon'}`}
              >
                <CategoryIcon slug={c.slug} />
                {c.name}
              </button>
            ))}
          </div>
        </div>

        {loading ? (
          <Spinner label="Memuat menu..." />
        ) : error ? (
          <div className="card mt-6 border-red-500 p-6 text-center dark:border-red-500">
            <p className="font-display text-lg font-black uppercase text-red-600">{error}</p>
            <button onClick={fetchProducts} className="btn btn-primary btn-sm mt-4">
              Coba Lagi
            </button>
          </div>
        ) : products.length === 0 ? (
          <div className="card mt-6 p-10 text-center">
            <p className="font-display text-xl font-black uppercase">Menu tidak ditemukan</p>
            <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
              Coba kata kunci atau filter lain.
            </p>
            <button onClick={resetFilters} className="btn btn-primary btn-sm mt-4">
              Reset Filter
            </button>
          </div>
        ) : (
          <>
            <div className="mt-6 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {products.map((p) => (
                <ProductCard key={p.id} product={p} />
              ))}
            </div>
            {totalPages > 1 && (
              <div className="mt-8 flex items-center justify-center gap-3">
                <button
                  className="btn btn-ghost btn-sm"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                >
                  <ChevronLeft className="h-4 w-4" /> Sebelumnya
                </button>
                <span className="font-display text-sm font-black">
                  Halaman {page} / {totalPages}
                </span>
                <button
                  className="btn btn-ghost btn-sm"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                >
                  Berikutnya <ChevronRight className="h-4 w-4" />
                </button>
              </div>
            )}
          </>
        )}
      </section>
    </div>
  )
}
