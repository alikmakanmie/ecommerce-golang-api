import { useEffect, useMemo, useState } from 'react'
import { Download } from 'lucide-react'
import { analyticsApi } from '../api/endpoints'
import { apiErrorMessage } from '../api/client'
import type { DailyProfitPoint, ProfitAnalytics, ProfitPeriod } from '../api/types'
import { formatDate, formatRupiah, orderTypeLabel } from '../utils/format'
import { Spinner } from '../components/ProtectedRoute'
import { StatusBadge } from '../components/StatusBadge'

const PERIODS: { value: ProfitPeriod; label: string }[] = [
  { value: 'today', label: 'Hari Ini' },
  { value: '7d', label: '7 Hari' },
  { value: '30d', label: '30 Hari' },
  { value: 'all', label: 'Semua' },
]

function Kpi({
  testid,
  label,
  value,
  sub,
}: {
  testid: string
  label: string
  value: string
  sub?: string
}) {
  return (
    <div className="card p-4" data-testid={testid}>
      <p className="font-display text-xs font-black uppercase tracking-wider text-zinc-500 dark:text-zinc-400">
        {label}
      </p>
      <p className="mt-2 font-display text-2xl font-black leading-none">{value}</p>
      {sub && <p className="mt-2 text-xs font-semibold text-zinc-500 dark:text-zinc-400">{sub}</p>}
    </div>
  )
}

/** Bila deret terlalu panjang (> 45 hari), agregasi per minggu agar grafik tetap terbaca. */
function aggregateSeries(series: DailyProfitPoint[]): DailyProfitPoint[] {
  if (series.length <= 45) return series
  const weeks: DailyProfitPoint[] = []
  for (let i = 0; i < series.length; i += 7) {
    const chunk = series.slice(i, i + 7)
    weeks.push({
      date: chunk[0].date,
      revenue: chunk.reduce((s, p) => s + p.revenue, 0),
      profit: chunk.reduce((s, p) => s + p.profit, 0),
      orders_count: chunk.reduce((s, p) => s + p.orders_count, 0),
    })
  }
  return weeks
}

function ProfitChart({ series }: { series: DailyProfitPoint[] }) {
  const points = useMemo(() => aggregateSeries(series), [series])
  const max = Math.max(1, ...points.map((p) => p.revenue))
  const slot = 18
  const width = Math.max(560, points.length * slot + 16)
  const height = 240
  const baseY = 210
  const plotH = 180
  const labelEvery = Math.max(1, Math.ceil(points.length / 8))

  return (
    <div className="overflow-x-auto" data-testid="profit-chart">
      <svg
        viewBox={`0 0 ${width} ${height}`}
        className="min-w-full"
        role="img"
        aria-label="Grafik omzet vs laba bersih per hari"
      >
        {/* Garis dasar */}
        <line x1="8" y1={baseY} x2={width - 8} y2={baseY} stroke="currentColor" strokeWidth="2" />
        {points.map((p, i) => {
          const x = 8 + i * slot
          const revH = Math.round((p.revenue / max) * plotH)
          const profH = Math.round((p.profit / max) * plotH)
          return (
            <g key={p.date}>
              <rect
                x={x}
                y={baseY - revH}
                width={7}
                height={revH}
                className="fill-black dark:fill-white"
                data-testid="chart-bar-revenue"
              >
                <title>{`${p.date}: Omzet ${formatRupiah(p.revenue)}`}</title>
              </rect>
              <rect
                x={x + 8}
                y={baseY - profH}
                width={7}
                height={profH}
                className="fill-yolk stroke-black dark:stroke-white"
                strokeWidth="1"
                data-testid="chart-bar-profit"
              >
                <title>{`${p.date}: Laba ${formatRupiah(p.profit)}`}</title>
              </rect>
              {i % labelEvery === 0 && (
                <text
                  x={x + 7}
                  y={baseY + 16}
                  textAnchor="middle"
                  className="fill-zinc-500 text-[9px] font-bold"
                >
                  {p.date.slice(5)}
                </text>
              )}
            </g>
          )
        })}
      </svg>
      <div className="mt-2 flex flex-wrap gap-4 text-xs font-bold">
        <span className="inline-flex items-center gap-1.5">
          <span className="inline-block h-3 w-3 border border-black bg-black dark:border-white dark:bg-white" /> Omzet
        </span>
        <span className="inline-flex items-center gap-1.5">
          <span className="inline-block h-3 w-3 border border-black bg-yolk" /> Laba Bersih
        </span>
        <span className="text-zinc-500">
          Puncak harian: {formatRupiah(max)} • {points.length} periode
        </span>
      </div>
    </div>
  )
}

function csvEscape(v: string | number): string {
  const s = String(v)
  return /[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}

export default function ProfitDashboard() {
  const [period, setPeriod] = useState<ProfitPeriod>('30d')
  const [data, setData] = useState<ProfitAnalytics | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let alive = true
    setLoading(true)
    setError('')
    analyticsApi
      .getProfit(period)
      .then((d) => {
        if (alive) setData(d)
      })
      .catch((err) => {
        if (alive) setError(apiErrorMessage(err, 'Gagal memuat analitik profit'))
      })
      .finally(() => {
        if (alive) setLoading(false)
      })
    return () => {
      alive = false
    }
  }, [period])

  const exportCsv = () => {
    if (!data) return
    const s = data.summary
    const today = new Date().toLocaleDateString('en-CA')
    const lines: string[] = []
    lines.push('SLARAS NUSA FOODLAB - LAPORAN PROFIT')
    lines.push(`Periode,${PERIODS.find((p) => p.value === period)?.label ?? period} (${data.start_date} s/d ${data.end_date})`)
    lines.push('')
    lines.push('RINGKASAN')
    lines.push('Metrik,Nilai (Rp)')
    lines.push(`Total Omzet (Gross Revenue),${s.gross_revenue}`)
    lines.push(`Total Modal (COGS),${s.total_cogs}`)
    lines.push(`Laba Bersih (Net Profit),${s.net_profit}`)
    lines.push(`Margin Profit (%),${s.profit_margin_percent}`)
    lines.push(`Total Pesanan Lunas,${s.total_paid_orders}`)
    lines.push(`Rata-rata Transaksi (AOV),${s.average_order_value}`)
    lines.push(`Pending / Dibatalkan,${s.pending_or_cancelled_count}`)
    lines.push('')
    lines.push('MENU PALING MENGUNTUNGKAN')
    lines.push('Menu,Terjual,Omzet (Rp),Profit (Rp)')
    for (const p of data.top_products) {
      lines.push([p.product_name, p.quantity_sold, p.revenue, p.profit].map(csvEscape).join(','))
    }
    lines.push('')
    lines.push('TRANSAKSI TERBARU')
    lines.push('Nomor Order,Nama Pemesan,Tipe Pesanan,Nomor Meja,Status,Total (Rp),Tanggal')
    for (const t of data.recent_transactions) {
      lines.push(
        [
          t.order_number,
          t.customer_name,
          t.order_type,
          t.table_number ?? '',
          t.status,
          t.total,
          t.created_at,
        ]
          .map(csvEscape)
          .join(','),
      )
    }
    const blob = new Blob([lines.join('\n')], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `laporan-profit-${today}.csv`
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  }

  if (loading) return <Spinner label="Memuat dashboard profit..." />

  return (
    <section className="mt-6" data-testid="profit-dashboard">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap gap-2" role="tablist" aria-label="Filter periode">
          {PERIODS.map((p) => (
            <button
              key={p.value}
              type="button"
              role="tab"
              aria-selected={period === p.value}
              onClick={() => setPeriod(p.value)}
              className={`btn btn-sm ${period === p.value ? 'btn-primary' : 'btn-ghost'}`}
            >
              {p.label.toUpperCase()}
            </button>
          ))}
        </div>
        <button
          onClick={exportCsv}
          disabled={!data}
          className="btn btn-accent btn-sm"
          data-testid="export-csv"
        >
          <Download className="h-4 w-4" /> Export Laporan CSV
        </button>
      </div>

      {error && (
        <p className="mt-4 border-2 border-red-500 bg-red-50 p-3 text-sm font-semibold text-red-700 dark:bg-red-950 dark:text-red-300">
          {error}
        </p>
      )}

      {data && (
        <>
          <p className="mt-4 text-xs font-bold text-zinc-500">
            Periode {data.start_date} s/d {data.end_date} • Mata uang Rupiah (IDR)
          </p>

          <div className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <Kpi
              testid="kpi-gross-revenue"
              label="Total Omzet (Gross Revenue)"
              value={formatRupiah(data.summary.gross_revenue)}
              sub={`${data.summary.total_paid_orders} transaksi lunas`}
            />
            <Kpi
              testid="kpi-net-profit"
              label="Laba Bersih (Net Profit)"
              value={formatRupiah(data.summary.net_profit)}
              sub={`Modal (COGS) ${formatRupiah(data.summary.total_cogs)}`}
            />
            <Kpi
              testid="kpi-margin"
              label="Margin Profit"
              value={`${data.summary.profit_margin_percent.toFixed(2)}%`}
              sub="Laba bersih / omzet"
            />
            <Kpi
              testid="kpi-paid-orders"
              label="Total Pesanan Lunas"
              value={String(data.summary.total_paid_orders)}
              sub={`${data.summary.pending_or_cancelled_count} pending / dibatalkan`}
            />
            <Kpi
              testid="kpi-aov"
              label="Rata-rata Transaksi (AOV)"
              value={formatRupiah(data.summary.average_order_value)}
              sub="Omzet / transaksi lunas"
            />
            <Kpi
              testid="kpi-cogs"
              label="Total Modal (COGS)"
              value={formatRupiah(data.summary.total_cogs)}
              sub="Harga modal seluruh item terjual"
            />
          </div>

          <div className="card mt-6 p-5">
            <h3 className="font-display text-lg font-black uppercase">Omzet vs Laba Bersih Harian</h3>
            <div className="mt-4">
              <ProfitChart series={data.daily_series} />
            </div>
          </div>

          <div className="mt-6 grid gap-6 lg:grid-cols-2">
            <div className="card p-5">
              <h3 className="font-display text-lg font-black uppercase">Menu Paling Menguntungkan</h3>
              {data.top_products.length === 0 ? (
                <p className="mt-3 text-sm text-zinc-500">Belum ada penjualan pada periode ini.</p>
              ) : (
                <table className="mt-4 w-full text-left text-sm" data-testid="top-products-table">
                  <thead>
                    <tr className="border-b-2 border-black font-display text-xs uppercase dark:border-white">
                      <th className="py-2 pr-2">Menu</th>
                      <th className="py-2 pr-2 text-right">Terjual</th>
                      <th className="py-2 pr-2 text-right">Omzet</th>
                      <th className="py-2 text-right">Profit</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.top_products.map((p) => {
                      const share =
                        data.summary.net_profit > 0
                          ? (p.profit / data.summary.net_profit) * 100
                          : 0
                      return (
                        <tr key={p.product_id} className="border-b border-zinc-200 dark:border-zinc-800">
                          <td className="py-2 pr-2">
                            <span className="block font-bold">{p.product_name}</span>
                            <span className="text-xs text-zinc-500">
                              Kontribusi {share.toFixed(1)}% dari laba
                            </span>
                          </td>
                          <td className="py-2 pr-2 text-right font-bold">{p.quantity_sold}</td>
                          <td className="py-2 pr-2 text-right">{formatRupiah(p.revenue)}</td>
                          <td className="py-2 text-right font-black text-green-700 dark:text-green-400">
                            {formatRupiah(p.profit)}
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              )}
            </div>

            <div className="flex flex-col gap-6">
              <div className="card p-5">
                <h3 className="font-display text-lg font-black uppercase">Breakdown Tipe Pesanan</h3>
                <ul className="mt-3 flex flex-col gap-2 text-sm">
                  {data.by_order_type.map((b) => (
                    <li key={b.order_type} className="flex items-center justify-between gap-2">
                      <span className="font-bold">{orderTypeLabel(b.order_type)}</span>
                      <span>
                        {b.orders_count} order •{' '}
                        <span className="font-black">{formatRupiah(b.revenue)}</span>
                      </span>
                    </li>
                  ))}
                  {data.by_order_type.length === 0 && (
                    <li className="text-zinc-500">Belum ada data.</li>
                  )}
                </ul>
              </div>
              <div className="card p-5">
                <h3 className="font-display text-lg font-black uppercase">Breakdown Pembayaran</h3>
                <ul className="mt-3 flex flex-col gap-2 text-sm">
                  {data.by_payment_method.map((b) => (
                    <li key={b.payment_method} className="flex items-center justify-between gap-2">
                      <span className="font-bold">{b.payment_method.toUpperCase()}</span>
                      <span>
                        {b.orders_count} order •{' '}
                        <span className="font-black">{formatRupiah(b.revenue)}</span>
                      </span>
                    </li>
                  ))}
                  {data.by_payment_method.length === 0 && (
                    <li className="text-zinc-500">Belum ada data.</li>
                  )}
                </ul>
              </div>
            </div>
          </div>

          <div className="card mt-6 p-5">
            <h3 className="font-display text-lg font-black uppercase">10 Transaksi Terbaru</h3>
            {data.recent_transactions.length === 0 ? (
              <p className="mt-3 text-sm text-zinc-500">Belum ada transaksi.</p>
            ) : (
              <div className="mt-4 overflow-x-auto">
                <table className="w-full text-left text-sm" data-testid="recent-transactions-table">
                  <thead>
                    <tr className="border-b-2 border-black font-display text-xs uppercase dark:border-white">
                      <th className="py-2 pr-2">Nomor Order</th>
                      <th className="py-2 pr-2">Pemesan</th>
                      <th className="py-2 pr-2">Tipe / Meja</th>
                      <th className="py-2 pr-2">Status</th>
                      <th className="py-2 text-right">Total</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.recent_transactions.map((t) => (
                      <tr
                        key={t.order_number}
                        className="border-b border-zinc-200 dark:border-zinc-800"
                      >
                        <td className="py-2 pr-2 font-display text-xs font-black">{t.order_number}</td>
                        <td className="py-2 pr-2 font-bold">{t.customer_name}</td>
                        <td className="py-2 pr-2">
                          {orderTypeLabel(t.order_type)}
                          {t.table_number ? ` • ${t.table_number}` : ''}
                          <span className="block text-xs text-zinc-500">{formatDate(t.created_at)}</span>
                        </td>
                        <td className="py-2 pr-2">
                          <StatusBadge status={t.status} />
                        </td>
                        <td className="py-2 text-right font-black">{formatRupiah(t.total)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </>
      )}
    </section>
  )
}
