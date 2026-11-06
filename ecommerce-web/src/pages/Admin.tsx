import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { catalogApi, orderApi } from '../api/endpoints'
import { apiErrorMessage } from '../api/client'
import type { Category, CategoryInput, Order, OrderStatus, Product, ProductInput } from '../api/types'
import { formatDate, formatRupiah } from '../utils/format'
import { Modal, Spinner } from '../components/ProtectedRoute'
import { NEXT_STATUS, StatusBadge } from '../components/StatusBadge'
import ProfitDashboard from './ProfitDashboard'

type Tab = 'profit' | 'products' | 'categories' | 'orders'

const EMPTY_PRODUCT: ProductInput = {
  category_id: '',
  name: '',
  slug: '',
  description: '',
  price: 0,
  cost_price: undefined,
  stock: 0,
  sku: '',
  image_url: '',
  weight_gram: 0,
  is_active: true,
}

const EMPTY_CATEGORY: CategoryInput = { name: '', description: '', image_url: '' }

function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/(^-|-$)/g, '')
}

export default function Admin() {
  const [tab, setTab] = useState<Tab>('profit')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  const [categories, setCategories] = useState<Category[]>([])
  const [products, setProducts] = useState<Product[]>([])
  const [orders, setOrders] = useState<Order[]>([])
  const [loading, setLoading] = useState(true)

  const [productModal, setProductModal] = useState<{ open: boolean; editing: Product | null }>({
    open: false,
    editing: null,
  })
  const [productForm, setProductForm] = useState<ProductInput>(EMPTY_PRODUCT)
  const [savingProduct, setSavingProduct] = useState(false)

  const [categoryModal, setCategoryModal] = useState<{ open: boolean; editing: Category | null }>({
    open: false,
    editing: null,
  })
  const [categoryForm, setCategoryForm] = useState<CategoryInput>(EMPTY_CATEGORY)
  const [savingCategory, setSavingCategory] = useState(false)

  const [busyId, setBusyId] = useState<string | null>(null)

  const fetchAll = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [cats, prods, ords] = await Promise.all([
        catalogApi.listCategories(),
        catalogApi.listProducts({ limit: 100 }),
        orderApi.listAll(1, 50),
      ])
      setCategories(cats)
      setProducts(prods.products)
      setOrders(ords.orders)
    } catch (err) {
      setError(apiErrorMessage(err, 'Gagal memuat data admin'))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchAll()
  }, [fetchAll])

  const flash = (msg: string) => {
    setNotice(msg)
    window.setTimeout(() => setNotice(''), 4000)
  }

  // ---------- Produk ----------
  const openProductForm = (p: Product | null) => {
    if (p) {
      setProductForm({
        category_id: p.category_id,
        name: p.name,
        slug: p.slug,
        description: p.description,
        price: p.price,
        cost_price: p.cost_price ?? undefined,
        stock: p.stock,
        sku: p.sku,
        image_url: p.image_url,
        weight_gram: p.weight_gram,
        is_active: p.is_active,
      })
    } else {
      setProductForm({ ...EMPTY_PRODUCT, category_id: categories[0]?.id ?? '' })
    }
    setProductModal({ open: true, editing: p })
  }

  const saveProduct = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setSavingProduct(true)
    try {
      const input: ProductInput = {
        ...productForm,
        slug: (productForm.slug ?? '').trim() || slugify(productForm.name),
        price: Number(productForm.price),
        // Kosong/0 = backend default 60% dari harga (create) / tidak diubah (update).
        cost_price:
          productForm.cost_price === undefined || Number.isNaN(Number(productForm.cost_price))
            ? 0
            : Number(productForm.cost_price),
        stock: Number(productForm.stock),
        weight_gram: Number(productForm.weight_gram) || 0,
      }
      if (productModal.editing) {
        await catalogApi.updateProduct(productModal.editing.id, input)
        flash('Produk diperbarui')
      } else {
        await catalogApi.createProduct(input)
        flash('Produk dibuat')
      }
      setProductModal({ open: false, editing: null })
      await fetchAll()
    } catch (err) {
      setError(apiErrorMessage(err, 'Gagal menyimpan produk'))
    } finally {
      setSavingProduct(false)
    }
  }

  const deleteProduct = async (p: Product) => {
    if (!window.confirm(`Hapus produk "${p.name}"?`)) return
    setBusyId(p.id)
    try {
      await catalogApi.deleteProduct(p.id)
      flash('Produk dihapus')
      await fetchAll()
    } catch (err) {
      setError(apiErrorMessage(err, 'Gagal menghapus produk'))
    } finally {
      setBusyId(null)
    }
  }

  // ---------- Kategori ----------
  const openCategoryForm = (c: Category | null) => {
    setCategoryForm(c ? { name: c.name, description: c.description, image_url: c.image_url } : EMPTY_CATEGORY)
    setCategoryModal({ open: true, editing: c })
  }

  const saveCategory = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setSavingCategory(true)
    try {
      if (categoryModal.editing) {
        await catalogApi.updateCategory(categoryModal.editing.id, categoryForm)
        flash('Kategori diperbarui')
      } else {
        await catalogApi.createCategory(categoryForm)
        flash('Kategori dibuat')
      }
      setCategoryModal({ open: false, editing: null })
      await fetchAll()
    } catch (err) {
      setError(apiErrorMessage(err, 'Gagal menyimpan kategori'))
    } finally {
      setSavingCategory(false)
    }
  }

  const deleteCategory = async (c: Category) => {
    if (!window.confirm(`Hapus kategori "${c.name}"? Produk di dalamnya bisa ikut bermasalah.`)) return
    setBusyId(c.id)
    try {
      await catalogApi.deleteCategory(c.id)
      flash('Kategori dihapus')
      await fetchAll()
    } catch (err) {
      setError(apiErrorMessage(err, 'Gagal menghapus kategori'))
    } finally {
      setBusyId(null)
    }
  }

  // ---------- Pesanan ----------
  const changeOrderStatus = async (orderId: string, status: OrderStatus) => {
    setBusyId(orderId)
    try {
      await orderApi.updateStatus(orderId, status)
      flash(`Status order diubah ke ${status}`)
      const res = await orderApi.listAll(1, 50)
      setOrders(res.orders)
    } catch (err) {
      setError(apiErrorMessage(err, 'Gagal mengubah status order'))
    } finally {
      setBusyId(null)
    }
  }

  const tabCls = (t: Tab) =>
    `btn btn-sm ${tab === t ? 'btn-primary' : 'btn-ghost'}`

  if (loading) return <Spinner label="Memuat panel admin..." />

  return (
    <div className="mx-auto max-w-7xl px-4 py-8">
      <h1 className="title">Admin Panel</h1>
      <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
        Kelola produk, kategori, dan pesanan SLARAS NUSA FOODLAB.
      </p>

      {error && (
        <p className="mt-4 border-2 border-red-500 bg-red-50 p-3 text-sm font-semibold text-red-700 dark:bg-red-950 dark:text-red-300">
          {error}
        </p>
      )}
      {notice && (
        <p className="mt-4 border-2 border-green-600 bg-green-50 p-3 text-sm font-semibold text-green-700 dark:bg-green-950 dark:text-green-300">
          {notice}
        </p>
      )}

      <div className="mt-6 flex flex-wrap gap-2">
        <button className={tabCls('profit')} onClick={() => setTab('profit')}>
          Dashboard Profit
        </button>
        <button className={tabCls('products')} onClick={() => setTab('products')}>
          Produk ({products.length})
        </button>
        <button className={tabCls('categories')} onClick={() => setTab('categories')}>
          Kategori ({categories.length})
        </button>
        <button className={tabCls('orders')} onClick={() => setTab('orders')}>
          Pesanan ({orders.length})
        </button>
      </div>

      {tab === 'profit' && <ProfitDashboard />}

      {tab === 'products' && (
        <section className="mt-6">
          <button onClick={() => openProductForm(null)} className="btn btn-accent btn-sm">
            <Plus className="h-4 w-4" /> Tambah Produk
          </button>
          <div className="mt-4 flex flex-col gap-3">
            {products.map((p) => (
              <article key={p.id} className="card flex flex-wrap items-center gap-3 p-4">
                <div className="min-w-0 flex-1">
                  <p className="font-display text-base font-black uppercase">{p.name}</p>
                  <p className="text-sm text-zinc-600 dark:text-zinc-400">
                    {p.category?.name ?? '-'} • {p.sku} • Stok {p.stock} •{' '}
                    {p.is_active ? 'Aktif' : 'Nonaktif'}
                    {p.cost_price !== undefined && p.cost_price > 0 && (
                      <> • HPP {formatRupiah(p.cost_price)}</>
                    )}
                  </p>
                </div>
                <span className="font-display text-lg font-black">{formatRupiah(p.price)}</span>
                <button onClick={() => openProductForm(p)} className="btn btn-ghost btn-sm">
                  <Pencil className="h-4 w-4" /> Ubah
                </button>
                <button
                  onClick={() => deleteProduct(p)}
                  disabled={busyId === p.id}
                  className="btn btn-danger btn-sm"
                >
                  <Trash2 className="h-4 w-4" /> Hapus
                </button>
              </article>
            ))}
            {products.length === 0 && <p className="card p-6 text-center">Belum ada produk.</p>}
          </div>
        </section>
      )}

      {tab === 'categories' && (
        <section className="mt-6">
          <button onClick={() => openCategoryForm(null)} className="btn btn-accent btn-sm">
            <Plus className="h-4 w-4" /> Tambah Kategori
          </button>
          <div className="mt-4 grid gap-3 sm:grid-cols-2">
            {categories.map((c) => (
              <article key={c.id} className="card p-4">
                <p className="font-display text-base font-black uppercase">{c.name}</p>
                <p className="text-sm text-zinc-600 dark:text-zinc-400">
                  slug: {c.slug} • {c.description || 'tanpa deskripsi'}
                </p>
                <div className="mt-3 flex gap-2">
                  <button onClick={() => openCategoryForm(c)} className="btn btn-ghost btn-sm">
                    <Pencil className="h-4 w-4" /> Ubah
                  </button>
                  <button
                    onClick={() => deleteCategory(c)}
                    disabled={busyId === c.id}
                    className="btn btn-danger btn-sm"
                  >
                    <Trash2 className="h-4 w-4" /> Hapus
                  </button>
                </div>
              </article>
            ))}
          </div>
        </section>
      )}

      {tab === 'orders' && (
        <section className="mt-6 flex flex-col gap-3">
          {orders.map((o) => (
            <article key={o.id} className="card p-4">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <Link
                    to={`/orders/${o.id}`}
                    className="font-display text-base font-black uppercase hover:underline"
                  >
                    {o.order_number}
                  </Link>
                  <p className="text-sm text-zinc-600 dark:text-zinc-400">
                    {formatDate(o.created_at)} • {o.shipping_name} • {o.payment_method.toUpperCase()} •{' '}
                    {o.items.length} item
                  </p>
                </div>
                <StatusBadge status={o.status} />
              </div>
              <div className="mt-3 flex flex-wrap items-center gap-3">
                <span className="font-display font-black">{formatRupiah(o.total)}</span>
                {NEXT_STATUS[o.status].length > 0 ? (
                  <select
                    className="input !w-auto !py-1.5 text-sm"
                    value=""
                    disabled={busyId === o.id}
                    onChange={(e) => {
                      if (e.target.value) changeOrderStatus(o.id, e.target.value as OrderStatus)
                    }}
                  >
                    <option value="">Ubah status...</option>
                    {NEXT_STATUS[o.status].map((s) => (
                      <option key={s} value={s}>
                        → {s}
                      </option>
                    ))}
                  </select>
                ) : (
                  <span className="text-xs text-zinc-500">Tidak ada transisi status lagi</span>
                )}
              </div>
            </article>
          ))}
          {orders.length === 0 && <p className="card p-6 text-center">Belum ada pesanan masuk.</p>}
        </section>
      )}

      {/* Modal form produk */}
      <Modal
        open={productModal.open}
        onClose={() => setProductModal({ open: false, editing: null })}
        title={productModal.editing ? 'Ubah Produk' : 'Tambah Produk'}
      >
        <form onSubmit={saveProduct} className="flex flex-col gap-3">
          <div>
            <label className="label">Kategori</label>
            <select
              className="input"
              value={productForm.category_id}
              onChange={(e) => setProductForm({ ...productForm, category_id: e.target.value })}
              required
            >
              <option value="">— Pilih kategori —</option>
              {categories.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="label">Nama Produk</label>
            <input
              className="input"
              value={productForm.name}
              onChange={(e) => setProductForm({ ...productForm, name: e.target.value })}
              required
              minLength={2}
              maxLength={150}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">Harga (Rp)</label>
              <input
                className="input"
                type="number"
                min={0}
                value={productForm.price}
                onChange={(e) => setProductForm({ ...productForm, price: Number(e.target.value) })}
                required
              />
            </div>
            <div>
              <label className="label">Stok</label>
              <input
                className="input"
                type="number"
                min={0}
                value={productForm.stock}
                onChange={(e) => setProductForm({ ...productForm, stock: Number(e.target.value) })}
                required
              />
            </div>
          </div>
          <div>
            <label className="label" htmlFor="product-cost-price">
              Harga Modal / HPP (Rp)
            </label>
            <input
              id="product-cost-price"
              className="input"
              type="number"
              min={0}
              placeholder="Kosong = 60% dari harga jual"
              value={productForm.cost_price ?? ''}
              onChange={(e) =>
                setProductForm({
                  ...productForm,
                  cost_price: e.target.value === '' ? undefined : Number(e.target.value),
                })
              }
            />
            <p className="mt-1 text-xs text-zinc-500">
              Dipakai menghitung laba di Dashboard Profit. Kosong saat tambah produk = otomatis 60%
              dari harga; kosong saat ubah = HPP lama dipertahankan.
            </p>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label">SKU</label>
              <input
                className="input"
                value={productForm.sku}
                onChange={(e) => setProductForm({ ...productForm, sku: e.target.value })}
                required
                maxLength={50}
              />
            </div>
            <div>
              <label className="label">Berat (gram)</label>
              <input
                className="input"
                type="number"
                min={0}
                value={productForm.weight_gram}
                onChange={(e) => setProductForm({ ...productForm, weight_gram: Number(e.target.value) })}
              />
            </div>
          </div>
          <div>
            <label className="label">Deskripsi</label>
            <textarea
              className="input"
              rows={2}
              value={productForm.description}
              onChange={(e) => setProductForm({ ...productForm, description: e.target.value })}
            />
          </div>
          <div>
            <label className="label">URL Gambar</label>
            <input
              className="input"
              placeholder="https://... (kosong = tile kategori)"
              value={productForm.image_url}
              onChange={(e) => setProductForm({ ...productForm, image_url: e.target.value })}
            />
          </div>
          <label className="flex items-center gap-2 text-sm font-semibold">
            <input
              type="checkbox"
              className="h-4 w-4 accent-black"
              checked={productForm.is_active}
              onChange={(e) => setProductForm({ ...productForm, is_active: e.target.checked })}
            />
            Produk aktif (tampil di katalog)
          </label>
          <button type="submit" disabled={savingProduct} className="btn btn-accent w-full">
            {savingProduct ? 'Menyimpan...' : productModal.editing ? 'Simpan Perubahan' : 'Buat Produk'}
          </button>
        </form>
      </Modal>

      {/* Modal form kategori */}
      <Modal
        open={categoryModal.open}
        onClose={() => setCategoryModal({ open: false, editing: null })}
        title={categoryModal.editing ? 'Ubah Kategori' : 'Tambah Kategori'}
      >
        <form onSubmit={saveCategory} className="flex flex-col gap-3">
          <div>
            <label className="label">Nama Kategori</label>
            <input
              className="input"
              value={categoryForm.name}
              onChange={(e) => setCategoryForm({ ...categoryForm, name: e.target.value })}
              required
              minLength={2}
              maxLength={100}
            />
          </div>
          <div>
            <label className="label">Deskripsi</label>
            <textarea
              className="input"
              rows={2}
              value={categoryForm.description}
              onChange={(e) => setCategoryForm({ ...categoryForm, description: e.target.value })}
            />
          </div>
          <div>
            <label className="label">URL Gambar</label>
            <input
              className="input"
              value={categoryForm.image_url}
              onChange={(e) => setCategoryForm({ ...categoryForm, image_url: e.target.value })}
            />
          </div>
          <button type="submit" disabled={savingCategory} className="btn btn-accent w-full">
            {savingCategory ? 'Menyimpan...' : 'Simpan Kategori'}
          </button>
        </form>
      </Modal>
    </div>
  )
}
