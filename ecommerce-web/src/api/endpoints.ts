import { api } from './client'
import type {
  Address,
  AddressInput,
  ApiResponse,
  AuthPayload,
  CartData,
  CartItem,
  Category,
  CategoryInput,
  GuestCheckoutInput,
  Order,
  OrderStatus,
  Product,
  ProductInput,
  ProfitAnalytics,
  ProfitPeriod,
  User,
} from './types'

export interface ProductFilters {
  search?: string
  category_id?: string
  min_price?: number
  max_price?: number
  page?: number
  limit?: number
}

// ---------- Auth ----------
export const authApi = {
  async login(email: string, password: string): Promise<AuthPayload> {
    const res = await api.post<ApiResponse<AuthPayload>>('/auth/login', { email, password })
    return res.data.data as AuthPayload
  },
  async register(name: string, email: string, password: string, phone?: string): Promise<User> {
    const res = await api.post<ApiResponse<{ user: User }>>('/auth/register', {
      name,
      email,
      password,
      phone: phone || undefined,
    })
    return (res.data.data as { user: User }).user
  },
  async logout(refreshToken: string): Promise<void> {
    await api.post('/auth/logout', { refresh_token: refreshToken })
  },
  async getProfile(): Promise<User> {
    const res = await api.get<ApiResponse<User>>('/users/me')
    return res.data.data as User
  },
  async updateProfile(name: string, phone: string): Promise<User> {
    const res = await api.put<ApiResponse<User>>('/users/me', { name, phone })
    return res.data.data as User
  },
}

// ---------- Katalog ----------
export const catalogApi = {
  async listCategories(): Promise<Category[]> {
    const res = await api.get<ApiResponse<Category[]>>('/categories')
    return res.data.data ?? []
  },
  async createCategory(input: CategoryInput): Promise<Category> {
    const res = await api.post<ApiResponse<Category>>('/categories', input)
    return res.data.data as Category
  },
  async updateCategory(id: string, input: CategoryInput): Promise<Category> {
    const res = await api.put<ApiResponse<Category>>(`/categories/${id}`, input)
    return res.data.data as Category
  },
  async deleteCategory(id: string): Promise<void> {
    await api.delete(`/categories/${id}`)
  },
  async listProducts(filters: ProductFilters = {}): Promise<{ products: Product[]; total: number; totalPages: number }> {
    const params: Record<string, string | number> = {}
    if (filters.search) params.search = filters.search
    if (filters.category_id) params.category_id = filters.category_id
    if (filters.min_price !== undefined) params.min_price = filters.min_price
    if (filters.max_price !== undefined) params.max_price = filters.max_price
    params.page = filters.page ?? 1
    params.limit = filters.limit ?? 20
    const res = await api.get<ApiResponse<Product[]>>('/products', { params })
    return {
      products: res.data.data ?? [],
      total: res.data.meta?.total ?? 0,
      totalPages: res.data.meta?.total_pages ?? 1,
    }
  },
  async getProduct(id: string): Promise<Product> {
    const res = await api.get<ApiResponse<Product>>(`/products/${id}`)
    return res.data.data as Product
  },
  async createProduct(input: ProductInput): Promise<Product> {
    const res = await api.post<ApiResponse<Product>>('/products', input)
    return res.data.data as Product
  },
  async updateProduct(id: string, input: ProductInput): Promise<Product> {
    const res = await api.put<ApiResponse<Product>>(`/products/${id}`, input)
    return res.data.data as Product
  },
  async deleteProduct(id: string): Promise<void> {
    await api.delete(`/products/${id}`)
  },
}

// ---------- Alamat ----------
export const addressApi = {
  async list(): Promise<Address[]> {
    const res = await api.get<ApiResponse<Address[]>>('/users/me/addresses')
    return res.data.data ?? []
  },
  async create(input: AddressInput): Promise<Address> {
    const res = await api.post<ApiResponse<Address>>('/users/me/addresses', input)
    return res.data.data as Address
  },
  async remove(id: string): Promise<void> {
    await api.delete(`/users/me/addresses/${id}`)
  },
  async setDefault(id: string): Promise<Address> {
    const res = await api.put<ApiResponse<Address>>(`/users/me/addresses/${id}/default`)
    return res.data.data as Address
  },
}

// ---------- Keranjang ----------
export const cartApi = {
  async get(): Promise<CartData> {
    const res = await api.get<ApiResponse<CartData>>('/cart')
    return res.data.data as CartData
  },
  async add(productId: string, quantity: number): Promise<CartItem> {
    const res = await api.post<ApiResponse<CartItem>>('/cart/items', {
      product_id: productId,
      quantity,
    })
    return res.data.data as CartItem
  },
  async updateQty(itemId: string, quantity: number): Promise<void> {
    await api.put(`/cart/items/${itemId}`, { quantity })
  },
  async remove(itemId: string): Promise<void> {
    await api.delete(`/cart/items/${itemId}`)
  },
}

// ---------- Order ----------
export const orderApi = {
  async checkout(addressId: string, paymentMethod: string, notes?: string): Promise<Order> {
    const res = await api.post<ApiResponse<Order>>('/orders/checkout', {
      address_id: addressId,
      payment_method: paymentMethod,
      notes: notes || undefined,
    })
    return res.data.data as Order
  },
  /** Guest checkout publik (tanpa JWT): dine-in / takeaway / delivery. */
  async guestCheckout(input: GuestCheckoutInput): Promise<Order> {
    const res = await api.post<ApiResponse<Order>>('/orders/guest-checkout', input)
    return res.data.data as Order
  },
  /** Lacak order publik (tanpa JWT) berdasarkan nomor order. */
  async track(orderNumber: string): Promise<Order> {
    const res = await api.get<ApiResponse<Order>>(`/orders/track/${encodeURIComponent(orderNumber)}`)
    return res.data.data as Order
  },
  async list(page = 1, limit = 20): Promise<{ orders: Order[]; total: number }> {
    const res = await api.get<ApiResponse<Order[]>>('/orders', { params: { page, limit } })
    return { orders: res.data.data ?? [], total: res.data.meta?.total ?? 0 }
  },
  async get(id: string): Promise<Order> {
    const res = await api.get<ApiResponse<Order>>(`/orders/${id}`)
    return res.data.data as Order
  },
  async cancel(id: string): Promise<Order> {
    const res = await api.post<ApiResponse<Order>>(`/orders/${id}/cancel`)
    return res.data.data as Order
  },
  async listAll(page = 1, limit = 50): Promise<{ orders: Order[]; total: number }> {
    const res = await api.get<ApiResponse<Order[]>>('/admin/orders', { params: { page, limit } })
    return { orders: res.data.data ?? [], total: res.data.meta?.total ?? 0 }
  },
  async updateStatus(id: string, status: OrderStatus): Promise<Order> {
    const res = await api.patch<ApiResponse<Order>>(`/admin/orders/${id}/status`, { status })
    return res.data.data as Order
  },
}

// ---------- Analitik (admin) ----------
export const analyticsApi = {
  async getProfit(period: ProfitPeriod = '30d'): Promise<ProfitAnalytics> {
    const res = await api.get<ApiResponse<ProfitAnalytics>>('/admin/analytics/profit', {
      params: { period },
    })
    return res.data.data as ProfitAnalytics
  },
}
