import { Coffee, Cookie, Gift, Package, ShoppingBag, UtensilsCrossed, type LucideIcon } from 'lucide-react'
import type { Product } from '../api/types'
import { API_URL } from '../api/client'

const CATEGORY_STYLE: Record<string, { icon: LucideIcon; bg: string; label: string }> = {
  'makanan-utama': { icon: UtensilsCrossed, bg: 'bg-yolk', label: 'Makanan Utama' },
  'camilan-roti': { icon: Cookie, bg: 'bg-orange-400', label: 'Camilan & Roti' },
  minuman: { icon: Coffee, bg: 'bg-sky-400', label: 'Minuman' },
  'paket-hampers': { icon: Gift, bg: 'bg-lime-400', label: 'Paket & Hampers' },
}

const FALLBACK = { icon: ShoppingBag, bg: 'bg-zinc-300', label: 'Slaras' }

function resolveImageUrl(url: string): string {
  if (!url) return ''
  if (url.startsWith('http://') || url.startsWith('https://')) return url
  if (url.startsWith('/')) {
    const origin = API_URL.replace(/\/api\/v1\/?$/, '')
    return `${origin}${url}`
  }
  return url
}

/** Visual produk: foto asli bila ada, bila tidak — tile neobrutalis per kategori. */
export function ProductVisual({ product, size = 'md' }: { product: Product; size?: 'sm' | 'md' | 'lg' }) {
  const style = CATEGORY_STYLE[product.category?.slug ?? ''] ?? FALLBACK
  const Icon = style.icon
  const img = resolveImageUrl(product.image_url)
  const textSize = size === 'lg' ? 'h-16 w-16' : size === 'sm' ? 'h-8 w-8' : 'h-12 w-12'

  if (img) {
    return (
      <div className="relative h-full w-full overflow-hidden bg-zinc-100 dark:bg-zinc-800">
        <img src={img} alt={product.name} className="h-full w-full object-cover" loading="lazy" />
      </div>
    )
  }
  return (
    <div className={`flex h-full w-full flex-col items-center justify-center gap-2 ${style.bg} p-4 text-black`}>
      <Icon className={textSize} strokeWidth={2.5} />
      <span className="text-center font-display text-[10px] font-black uppercase tracking-widest">
        {style.label}
      </span>
    </div>
  )
}

/** Ikon kecil kategori (untuk chip filter). */
export function CategoryIcon({ slug, className = 'h-4 w-4' }: { slug: string; className?: string }) {
  const style = CATEGORY_STYLE[slug]
  if (!style) return <Package className={className} />
  const Icon = style.icon
  return <Icon className={className} />
}
