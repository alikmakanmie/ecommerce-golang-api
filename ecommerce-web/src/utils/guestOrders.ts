import type { GuestOrderSummary, Order, OrderStatus } from '../api/types'

const KEY = 'snf_guest_orders'

export function loadGuestOrders(): GuestOrderSummary[] {
  try {
    const raw = localStorage.getItem(KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as GuestOrderSummary[]
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

export function saveGuestOrder(order: Order): void {
  try {
    const list = loadGuestOrders().filter((o) => o.order_number !== order.order_number)
    list.unshift({
      order_number: order.order_number,
      customer_name: order.customer_name ?? order.shipping_name ?? '',
      order_type: order.order_type ?? 'takeaway',
      total: order.total,
      status: order.status,
      created_at: order.created_at,
    })
    localStorage.setItem(KEY, JSON.stringify(list.slice(0, 50)))
  } catch {
    // localStorage penuh / tidak tersedia — riwayat tamu sekadar pelengkap.
  }
}

export function updateGuestOrderStatus(orderNumber: string, status: OrderStatus): void {
  try {
    const list = loadGuestOrders()
    const idx = list.findIndex((o) => o.order_number === orderNumber)
    if (idx >= 0 && list[idx].status !== status) {
      list[idx] = { ...list[idx], status }
      localStorage.setItem(KEY, JSON.stringify(list))
    }
  } catch {
    // Abaikan.
  }
}

export function clearGuestOrders(): void {
  try {
    localStorage.removeItem(KEY)
  } catch {
    // Abaikan.
  }
}
