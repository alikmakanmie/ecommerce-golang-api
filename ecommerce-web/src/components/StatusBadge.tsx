import type { OrderStatus } from '../api/types'

const STATUS_LABEL: Record<OrderStatus, string> = {
  pending: 'Menunggu Bayar',
  paid: 'Dibayar',
  processing: 'Diproses',
  shipped: 'Dikirim',
  completed: 'Selesai',
  cancelled: 'Dibatalkan',
}

const STATUS_STYLE: Record<OrderStatus, string> = {
  pending: 'bg-yolk text-black',
  paid: 'bg-green-500 text-white',
  processing: 'bg-sky-400 text-black',
  shipped: 'bg-purple-400 text-black',
  completed: 'bg-black text-white dark:bg-white dark:text-black',
  cancelled: 'bg-red-500 text-white',
}

export function StatusBadge({ status }: { status: OrderStatus }) {
  return (
    <span className={`chip ${STATUS_STYLE[status]}`}>{STATUS_LABEL[status] ?? status}</span>
  )
}

export const ORDER_STATUSES: OrderStatus[] = [
  'pending',
  'paid',
  'processing',
  'shipped',
  'completed',
  'cancelled',
]

export const NEXT_STATUS: Record<OrderStatus, OrderStatus[]> = {
  pending: ['paid', 'cancelled'],
  paid: ['processing', 'cancelled'],
  processing: ['shipped', 'cancelled'],
  shipped: ['completed'],
  completed: [],
  cancelled: [],
}
