import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react'
import { cartApi, catalogApi } from '../api/endpoints'
import { apiErrorMessage } from '../api/client'
import { useAuth } from './AuthContext'
import type { CartData, CartItem, GuestCartItem, Product } from '../api/types'

const GUEST_CART_KEY = 'snf_guest_cart'

function loadGuestItems(): GuestCartItem[] {
  try {
    const raw = localStorage.getItem(GUEST_CART_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as GuestCartItem[]
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function persistGuestItems(items: GuestCartItem[]): void {
  try {
    localStorage.setItem(GUEST_CART_KEY, JSON.stringify(items))
  } catch {
    // localStorage tidak tersedia — keranjang tamu sekadar tidak tersimpan.
  }
}

/** Bentuk CartData yang sama dengan API, disintesis dari keranjang tamu lokal. */
function guestItemsToCartData(items: GuestCartItem[]): CartData {
  const cartItems: CartItem[] = items.map((it) => ({
    id: `guest-${it.product_id}`,
    cart_id: 'guest',
    product_id: it.product_id,
    product: {
      id: it.product.id,
      category_id: '',
      name: it.product.name,
      slug: '',
      description: '',
      price: it.product.price,
      stock: 0,
      sku: '',
      image_url: it.product.image_url,
      weight_gram: 0,
      is_active: true,
      created_at: '',
      updated_at: '',
    },
    quantity: it.quantity,
    price: it.product.price,
    notes: it.notes || undefined,
    created_at: '',
    updated_at: '',
  }))
  return {
    cart: { id: 'guest', user_id: '', items: cartItems },
    subtotal: cartItems.reduce((s, i) => s + i.price * i.quantity, 0),
    total_items: cartItems.reduce((s, i) => s + i.quantity, 0),
  }
}

interface CartCtx {
  cart: CartData | null
  loading: boolean
  busy: boolean
  /** true bila keranjang saat ini adalah keranjang tamu (localStorage). */
  isGuestCart: boolean
  refresh: () => Promise<void>
  addItem: (productId: string, quantity?: number, product?: Product) => Promise<void>
  updateQty: (itemId: string, quantity: number) => Promise<void>
  removeItem: (itemId: string) => Promise<void>
  /** Catatan per item — hanya berlaku untuk keranjang tamu. */
  setItemNotes: (itemId: string, notes: string) => void
  clearLocal: () => void
}

const Ctx = createContext<CartCtx>({
  cart: null,
  loading: false,
  busy: false,
  isGuestCart: true,
  refresh: async () => {},
  addItem: async () => {},
  updateQty: async () => {},
  removeItem: async () => {},
  setItemNotes: () => {},
  clearLocal: () => {},
})

export function CartProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth()
  const [cart, setCart] = useState<CartData | null>(null)
  const [loading, setLoading] = useState(false)
  const [busy, setBusy] = useState(false)
  const mergedForUserRef = useRef<string | null>(null)

  const refresh = useCallback(async () => {
    if (!user) {
      setCart(guestItemsToCartData(loadGuestItems()))
      return
    }
    setLoading(true)
    try {
      const data = await cartApi.get()
      setCart(data)
    } catch {
      setCart(null)
    } finally {
      setLoading(false)
    }
  }, [user])

  useEffect(() => {
    refresh()
  }, [refresh])

  // Saat user login, pindahkan isi keranjang tamu (localStorage) ke /cart API.
  // Item yang gagal dipindah (mis. stok habis) tetap tersimpan sebagai tamu.
  useEffect(() => {
    if (!user) {
      mergedForUserRef.current = null
      return
    }
    if (mergedForUserRef.current === user.id) return
    mergedForUserRef.current = user.id
    const pending = loadGuestItems()
    if (pending.length === 0) return
    ;(async () => {
      const remaining: GuestCartItem[] = []
      for (const it of pending) {
        try {
          await cartApi.add(it.product_id, it.quantity)
        } catch {
          remaining.push(it)
        }
      }
      persistGuestItems(remaining)
      await refresh()
    })()
  }, [user, refresh])

  const addItem = useCallback(
    async (productId: string, quantity = 1, product?: Product) => {
      setBusy(true)
      try {
        if (user) {
          await cartApi.add(productId, quantity)
          await refresh()
          return
        }
        const snap = product ?? (await catalogApi.getProduct(productId))
        const items = loadGuestItems()
        const idx = items.findIndex((i) => i.product_id === productId)
        if (idx >= 0) {
          items[idx] = { ...items[idx], quantity: Math.min(items[idx].quantity + quantity, 99) }
        } else {
          items.push({
            product_id: productId,
            quantity,
            notes: '',
            product: {
              id: snap.id,
              name: snap.name,
              price: snap.price,
              image_url: snap.image_url,
            },
          })
        }
        persistGuestItems(items)
        setCart(guestItemsToCartData(items))
      } catch (err) {
        throw new Error(apiErrorMessage(err, 'Gagal menambah ke keranjang'))
      } finally {
        setBusy(false)
      }
    },
    [user, refresh],
  )

  const updateQty = useCallback(
    async (itemId: string, quantity: number) => {
      setBusy(true)
      try {
        if (user) {
          await cartApi.updateQty(itemId, quantity)
          await refresh()
          return
        }
        const productId = itemId.replace(/^guest-/, '')
        const items = loadGuestItems()
        const idx = items.findIndex((i) => i.product_id === productId)
        if (idx < 0) return
        if (quantity < 1) {
          items.splice(idx, 1)
        } else {
          items[idx] = { ...items[idx], quantity }
        }
        persistGuestItems(items)
        setCart(guestItemsToCartData(items))
      } catch (err) {
        throw new Error(apiErrorMessage(err, 'Gagal mengubah kuantitas'))
      } finally {
        setBusy(false)
      }
    },
    [user, refresh],
  )

  const removeItem = useCallback(
    async (itemId: string) => {
      setBusy(true)
      try {
        if (user) {
          await cartApi.remove(itemId)
          await refresh()
          return
        }
        const productId = itemId.replace(/^guest-/, '')
        const items = loadGuestItems().filter((i) => i.product_id !== productId)
        persistGuestItems(items)
        setCart(guestItemsToCartData(items))
      } catch (err) {
        throw new Error(apiErrorMessage(err, 'Gagal menghapus item'))
      } finally {
        setBusy(false)
      }
    },
    [user, refresh],
  )

  const setItemNotes = useCallback((itemId: string, notes: string) => {
    const productId = itemId.replace(/^guest-/, '')
    const items = loadGuestItems()
    const idx = items.findIndex((i) => i.product_id === productId)
    if (idx < 0) return
    items[idx] = { ...items[idx], notes }
    persistGuestItems(items)
    setCart(guestItemsToCartData(items))
  }, [])

  const clearLocal = useCallback(() => {
    persistGuestItems([])
    if (!user) setCart(guestItemsToCartData([]))
    else setCart(null)
  }, [user])

  return (
    <Ctx.Provider
      value={{
        cart,
        loading,
        busy,
        isGuestCart: !user,
        refresh,
        addItem,
        updateQty,
        removeItem,
        setItemNotes,
        clearLocal,
      }}
    >
      {children}
    </Ctx.Provider>
  )
}

export const useCart = () => useContext(Ctx)
