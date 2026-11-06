export function formatRupiah(value: number): string {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value)
}

export function formatDate(value: string): string {
  return new Date(value).toLocaleDateString('id-ID', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/** Ongkir dummy backend: Rp10.000 bila subtotal < Rp500.000, gratis bila >= Rp500.000. */
export function shippingCost(subtotal: number): number {
  return subtotal >= 500000 ? 0 : 10000
}

export function orderTypeLabel(orderType?: string): string {
  switch (orderType) {
    case 'dine_in':
      return 'Dine-In (Makan di Tempat)'
    case 'takeaway':
      return 'Takeaway (Bungkus)'
    case 'delivery':
      return 'Delivery'
    default:
      return orderType ?? '-'
  }
}
