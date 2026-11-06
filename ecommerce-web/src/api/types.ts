// Tipe data sesuai kontrak JSON backend Go (internal/domain/*.go).

export interface ApiResponse<T> {
  success: boolean
  message?: string
  data?: T
  error?: string
  meta?: PaginationMeta
}

export interface PaginationMeta {
  page: number
  limit: number
  total: number
  total_pages: number
}

export interface User {
  id: string
  name: string
  email: string
  phone: string
  role: 'admin' | 'customer'
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface Category {
  id: string
  name: string
  slug: string
  description: string
  image_url: string
  created_at: string
  updated_at: string
}

export interface Product {
  id: string
  category_id: string
  category?: Category
  name: string
  slug: string
  description: string
  price: number
  /** Harga modal / HPP. 0 di respons lama berarti belum diisi. */
  cost_price?: number
  stock: number
  sku: string
  image_url: string
  weight_gram: number
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface Address {
  id: string
  user_id: string
  label: string
  recipient_name: string
  phone: string
  street: string
  city: string
  province: string
  postal_code: string
  is_default: boolean
  created_at: string
  updated_at: string
}

export interface CartItem {
  id: string
  cart_id: string
  product_id: string
  product?: Product
  quantity: number
  price: number
  /** Catatan per item, hanya dipakai keranjang tamu (localStorage). */
  notes?: string
  created_at: string
  updated_at: string
}

export interface CartData {
  cart: {
    id: string
    user_id: string
    items: CartItem[]
  }
  subtotal: number
  total_items: number
}

export interface OrderItem {
  id: string
  order_id: string
  product_id: string
  product_name: string
  price: number
  quantity: number
  subtotal: number
  /** Catatan per item, mis. "less sugar / normal ice". */
  notes?: string
  created_at: string
  updated_at: string
}

export interface PaymentInfo {
  id?: string
  order_id?: string
  transaction_id: string
  gateway: string
  method: string
  amount: number
  status: string
  payment_url?: string
  qr_string?: string
  expires_at?: string
  paid_at?: string
  created_at?: string
  updated_at?: string
}

export type OrderStatus =
  | 'pending'
  | 'paid'
  | 'processing'
  | 'shipped'
  | 'completed'
  | 'cancelled'

export type OrderType = 'dine_in' | 'takeaway' | 'delivery'

export interface Order {
  id: string
  /** null untuk pesanan tamu (guest checkout). */
  user_id?: string | null
  order_number: string
  customer_name?: string
  customer_phone?: string
  order_type?: OrderType
  table_number?: string
  address_id?: string
  shipping_address?: Address | null
  shipping_name: string
  shipping_phone: string
  shipping_address_text: string
  status: OrderStatus
  subtotal: number
  shipping_cost: number
  total: number
  payment_method: string
  notes: string
  items: OrderItem[]
  payment?: PaymentInfo | null
  created_at: string
  updated_at: string
}

/** Item keranjang tamu yang disimpan di localStorage. */
export interface GuestCartItem {
  product_id: string
  quantity: number
  notes: string
  product: {
    id: string
    name: string
    price: number
    image_url: string
  }
}

/** Input POST /orders/guest-checkout (publik, tanpa JWT). */
export interface GuestCheckoutItemInput {
  product_id: string
  quantity: number
  notes?: string
}

export interface GuestDeliveryAddressInput {
  street: string
  city: string
  province: string
  postal_code: string
}

export interface GuestCheckoutInput {
  items: GuestCheckoutItemInput[]
  customer_name: string
  customer_phone?: string
  order_type: OrderType
  table_number?: string
  payment_method: 'qris' | 'va'
  notes?: string
  delivery_address?: GuestDeliveryAddressInput
}

/** Ringkasan order tamu yang disimpan di localStorage ("Pesanan Saya"). */
export interface GuestOrderSummary {
  order_number: string
  customer_name: string
  order_type: OrderType
  total: number
  status: OrderStatus
  created_at: string
}

export interface AuthPayload {
  access_token: string
  refresh_token: string
  user: User
}

// ---------- Analitik profit (GET /api/v1/admin/analytics/profit) ----------
export interface ProfitSummary {
  gross_revenue: number
  total_cogs: number
  net_profit: number
  profit_margin_percent: number
  total_paid_orders: number
  average_order_value: number
  pending_or_cancelled_count: number
}

export interface DailyProfitPoint {
  date: string
  revenue: number
  profit: number
  orders_count: number
}

export interface OrderTypeBreakdown {
  order_type: string
  revenue: number
  orders_count: number
}

export interface PaymentBreakdown {
  payment_method: string
  revenue: number
  orders_count: number
}

export interface TopProduct {
  product_id: string
  product_name: string
  quantity_sold: number
  revenue: number
  profit: number
}

export interface RecentTransaction {
  order_number: string
  customer_name: string
  order_type: string
  table_number?: string
  status: OrderStatus
  total: number
  created_at: string
}

export type ProfitPeriod = 'today' | '7d' | '30d' | 'all'

export interface ProfitAnalytics {
  period: string
  start_date: string
  end_date: string
  summary: ProfitSummary
  daily_series: DailyProfitPoint[]
  by_order_type: OrderTypeBreakdown[]
  by_payment_method: PaymentBreakdown[]
  top_products: TopProduct[]
  recent_transactions: RecentTransaction[]
}

export interface ProductInput {
  category_id: string
  name: string
  slug?: string
  description?: string
  price: number
  /** Harga modal / HPP. 0 atau kosong = backend default 60% dari harga (create) / tidak diubah (update). */
  cost_price?: number
  stock: number
  sku: string
  image_url?: string
  weight_gram?: number
  is_active?: boolean
}

export interface CategoryInput {
  name: string
  description?: string
  image_url?: string
}

export interface AddressInput {
  label: string
  recipient_name: string
  phone: string
  street: string
  city: string
  province: string
  postal_code: string
  is_default?: boolean
}
