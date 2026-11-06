package service

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"ecommerce-api/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Status order yang dihitung sebagai omzet (uang sudah/sedang diproses,
// bukan pending yang belum dibayar dan bukan yang dibatalkan).
var revenueStatuses = []domain.OrderStatus{
	domain.OrderPaid,
	domain.OrderProcessing,
	domain.OrderShipped,
	domain.OrderCompleted,
}

type ProfitSummary struct {
	GrossRevenue            int64   `json:"gross_revenue"`
	TotalCOGS               int64   `json:"total_cogs"`
	NetProfit               int64   `json:"net_profit"`
	ProfitMarginPercent     float64 `json:"profit_margin_percent"`
	TotalPaidOrders         int64   `json:"total_paid_orders"`
	AverageOrderValue       int64   `json:"average_order_value"`
	PendingOrCancelledCount int64   `json:"pending_or_cancelled_count"`
}

type DailyPoint struct {
	Date        string `json:"date"`
	Revenue     int64  `json:"revenue"`
	Profit      int64  `json:"profit"`
	OrdersCount int64  `json:"orders_count"`
}

type OrderTypeBreakdown struct {
	OrderType   string `json:"order_type"`
	Revenue     int64  `json:"revenue"`
	OrdersCount int64  `json:"orders_count"`
}

type PaymentBreakdown struct {
	PaymentMethod string `json:"payment_method"`
	Revenue       int64  `json:"revenue"`
	OrdersCount   int64  `json:"orders_count"`
}

type TopProduct struct {
	ProductID    uuid.UUID `json:"product_id"`
	ProductName  string    `json:"product_name"`
	QuantitySold int64     `json:"quantity_sold"`
	Revenue      int64     `json:"revenue"`
	Profit       int64     `json:"profit"`
}

type RecentTransaction struct {
	OrderNumber  string             `json:"order_number"`
	CustomerName string             `json:"customer_name"`
	OrderType    string             `json:"order_type"`
	TableNumber  string             `json:"table_number,omitempty"`
	Status       domain.OrderStatus `json:"status"`
	Total        int64              `json:"total"`
	CreatedAt    time.Time          `json:"created_at"`
}

type ProfitAnalytics struct {
	Period             string               `json:"period"`
	StartDate          string               `json:"start_date"`
	EndDate            string               `json:"end_date"`
	Summary            ProfitSummary        `json:"summary"`
	DailySeries        []DailyPoint         `json:"daily_series"`
	ByOrderType        []OrderTypeBreakdown `json:"by_order_type"`
	ByPaymentMethod    []PaymentBreakdown   `json:"by_payment_method"`
	TopProducts        []TopProduct         `json:"top_products"`
	RecentTransactions []RecentTransaction  `json:"recent_transactions"`
}

type AnalyticsService struct {
	db *gorm.DB
}

func NewAnalyticsService(db *gorm.DB) *AnalyticsService {
	return &AnalyticsService{db: db}
}

func wibLocation() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.UTC
}

// resolveRange mengubah period / start_date & end_date menjadi rentang
// [start, end) dalam UTC untuk query, plus label tanggal WIB untuk respons.
// start_date & end_date (YYYY-MM-DD, inklusif) mengalahkan period bila
// keduanya diisi. "today" = hari ini WIB; "7d"/"30d" = N hari terakhir
// termasuk hari ini; "all" = tanpa filter tanggal.
func resolveRange(period, startDate, endDate string) (start, end time.Time, hasRange bool, labelStart, labelEnd, labelPeriod string, err error) {
	loc := wibLocation()
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	period = strings.ToLower(strings.TrimSpace(period))
	if period == "" {
		period = "30d"
	}
	if strings.TrimSpace(startDate) != "" && strings.TrimSpace(endDate) != "" {
		s, e1 := time.ParseInLocation("2006-01-02", strings.TrimSpace(startDate), loc)
		e, e2 := time.ParseInLocation("2006-01-02", strings.TrimSpace(endDate), loc)
		if e1 != nil || e2 != nil {
			return start, end, false, "", "", "", errors.New("start_date/end_date harus format YYYY-MM-DD")
		}
		if e.Before(s) {
			return start, end, false, "", "", "", errors.New("end_date tidak boleh sebelum start_date")
		}
		start, end = s, e.AddDate(0, 0, 1)
		return start.UTC(), end.UTC(), true, s.Format("2006-01-02"), e.Format("2006-01-02"), "custom", nil
	}
	switch period {
	case "today":
		start, end = today, today.AddDate(0, 0, 1)
	case "7d":
		start, end = today.AddDate(0, 0, -6), today.AddDate(0, 0, 1)
	case "30d":
		start, end = today.AddDate(0, 0, -29), today.AddDate(0, 0, 1)
	case "all":
		return start, end, false, "", "", "all", nil
	default:
		return start, end, false, "", "", "", fmt.Errorf("period harus today, 7d, 30d, atau all")
	}
	return start.UTC(), end.UTC(), true,
		start.In(loc).Format("2006-01-02"), end.AddDate(0, 0, -1).In(loc).Format("2006-01-02"), period, nil
}

func (s *AnalyticsService) GetProfitAnalytics(period, startDate, endDate string) (*ProfitAnalytics, error) {
	start, end, hasRange, labelStart, labelEnd, labelPeriod, err := resolveRange(period, startDate, endDate)
	if err != nil {
		return nil, err
	}
	// Kondisi tanggal dipakai di semua query agregat; "all" tanpa filter.
	dateCond := ""
	var args []interface{}
	if hasRange {
		dateCond = " AND created_at >= ? AND created_at < ?"
		args = append(args, start, end)
	}
	revIn := "status IN ('paid','processing','shipped','completed')"

	// --- Summary: omzet + jumlah transaksi lunas ---
	var sum struct {
		Revenue int64 `gorm:"column:revenue"`
		Count   int64 `gorm:"column:orders_count"`
	}
	q := fmt.Sprintf("SELECT COALESCE(SUM(total),0) AS revenue, COUNT(*) AS orders_count FROM orders WHERE %s%s AND deleted_at IS NULL", revIn, dateCond)
	if err := s.db.Raw(q, args...).Scan(&sum).Error; err != nil {
		return nil, err
	}
	// --- Total HPP: snapshot cost_price di order_items, fallback ke harga
	// modal produk saat ini bila snapshot 0 (order lama sebelum snapshot).
	var cogs struct {
		COGS int64 `gorm:"column:cogs"`
	}
	qc := "SELECT COALESCE(SUM(oi.quantity * COALESCE(NULLIF(oi.cost_price, 0), p.cost_price, 0)), 0) AS cogs " +
		"FROM order_items oi JOIN orders o ON o.id = oi.order_id JOIN products p ON p.id = oi.product_id " +
		"WHERE o.status IN ('paid','processing','shipped','completed')"
	if hasRange {
		qc += " AND o.created_at >= ? AND o.created_at < ?"
	}
	qc += " AND o.deleted_at IS NULL AND oi.deleted_at IS NULL"
	if err := s.db.Raw(qc, args...).Scan(&cogs).Error; err != nil {
		return nil, err
	}
	// --- Pending + cancelled dalam rentang ---
	var pc struct {
		Count int64 `gorm:"column:cnt"`
	}
	qp := fmt.Sprintf("SELECT COUNT(*) AS cnt FROM orders WHERE status IN ('pending','cancelled')%s AND deleted_at IS NULL", dateCond)
	if err := s.db.Raw(qp, args...).Scan(&pc).Error; err != nil {
		return nil, err
	}

	summary := ProfitSummary{
		GrossRevenue:            sum.Revenue,
		TotalCOGS:               cogs.COGS,
		NetProfit:               sum.Revenue - cogs.COGS,
		TotalPaidOrders:         sum.Count,
		PendingOrCancelledCount: pc.Count,
	}
	if sum.Revenue > 0 {
		summary.ProfitMarginPercent = math.Round(float64(summary.NetProfit)/float64(sum.Revenue)*10000) / 100
	}
	if sum.Count > 0 {
		summary.AverageOrderValue = sum.Revenue / sum.Count
	}

	// --- Time series harian (tanggal WIB via +7 jam) ---
	type dayRev struct {
		Day         string `gorm:"column:day"`
		Revenue     int64  `gorm:"column:revenue"`
		OrdersCount int64  `gorm:"column:orders_count"`
	}
	type dayCogs struct {
		Day  string `gorm:"column:day"`
		COGS int64  `gorm:"column:cogs"`
	}
	var revDays []dayRev
	qr := fmt.Sprintf("SELECT date(created_at, '+7 hours') AS day, COALESCE(SUM(total),0) AS revenue, COUNT(*) AS orders_count FROM orders WHERE %s%s AND deleted_at IS NULL GROUP BY day ORDER BY day", revIn, dateCond)
	if err := s.db.Raw(qr, args...).Scan(&revDays).Error; err != nil {
		return nil, err
	}
	var cogsDays []dayCogs
	qcd := "SELECT date(o.created_at, '+7 hours') AS day, COALESCE(SUM(oi.quantity * COALESCE(NULLIF(oi.cost_price, 0), p.cost_price, 0)), 0) AS cogs " +
		"FROM order_items oi JOIN orders o ON o.id = oi.order_id JOIN products p ON p.id = oi.product_id " +
		"WHERE o.status IN ('paid','processing','shipped','completed')"
	if hasRange {
		qcd += " AND o.created_at >= ? AND o.created_at < ?"
	}
	qcd += " AND o.deleted_at IS NULL AND oi.deleted_at IS NULL GROUP BY day ORDER BY day"
	if err := s.db.Raw(qcd, args...).Scan(&cogsDays).Error; err != nil {
		return nil, err
	}
	cogsByDay := map[string]int64{}
	for _, d := range cogsDays {
		cogsByDay[d.Day] = d.COGS
	}
	revByDay := map[string]dayRev{}
	var firstDay, lastDay string
	for _, d := range revDays {
		revByDay[d.Day] = d
		if firstDay == "" {
			firstDay = d.Day
		}
		lastDay = d.Day
	}
	// Isi hari kosong dengan nol agar grafik kontinu.
	series := []DailyPoint{}
	if hasRange {
		loc := wibLocation()
		for d := start.In(loc); d.Before(end.In(loc)); d = d.AddDate(0, 0, 1) {
			key := d.Format("2006-01-02")
			r := revByDay[key]
			c := cogsByDay[key]
			series = append(series, DailyPoint{Date: key, Revenue: r.Revenue, Profit: r.Revenue - c, OrdersCount: r.OrdersCount})
		}
		if labelStart == "" {
			labelStart = start.In(loc).Format("2006-01-02")
		}
		if labelEnd == "" {
			labelEnd = end.AddDate(0, 0, -1).In(loc).Format("2006-01-02")
		}
	} else {
		for _, d := range revDays {
			c := cogsByDay[d.Day]
			series = append(series, DailyPoint{Date: d.Day, Revenue: d.Revenue, Profit: d.Revenue - c, OrdersCount: d.OrdersCount})
		}
		labelStart, labelEnd = firstDay, lastDay
	}
	if series == nil {
		series = []DailyPoint{}
	}

	// --- Breakdown tipe pesanan (semua tipe tampil, 0 bila kosong) ---
	var typeRows []struct {
		OrderType   string `gorm:"column:order_type"`
		Revenue     int64  `gorm:"column:revenue"`
		OrdersCount int64  `gorm:"column:orders_count"`
	}
	qt := fmt.Sprintf("SELECT order_type, COALESCE(SUM(total),0) AS revenue, COUNT(*) AS orders_count FROM orders WHERE %s%s AND deleted_at IS NULL GROUP BY order_type", revIn, dateCond)
	if err := s.db.Raw(qt, args...).Scan(&typeRows).Error; err != nil {
		return nil, err
	}
	byType := []OrderTypeBreakdown{
		{OrderType: "dine_in"}, {OrderType: "takeaway"}, {OrderType: "delivery"},
	}
	for _, r := range typeRows {
		for i := range byType {
			if byType[i].OrderType == r.OrderType {
				byType[i].Revenue = r.Revenue
				byType[i].OrdersCount = r.OrdersCount
			}
		}
	}

	// --- Breakdown metode pembayaran ---
	var payRows []struct {
		Method      string `gorm:"column:payment_method"`
		Revenue     int64  `gorm:"column:revenue"`
		OrdersCount int64  `gorm:"column:orders_count"`
	}
	qm := fmt.Sprintf("SELECT payment_method, COALESCE(SUM(total),0) AS revenue, COUNT(*) AS orders_count FROM orders WHERE %s%s AND deleted_at IS NULL GROUP BY payment_method", revIn, dateCond)
	if err := s.db.Raw(qm, args...).Scan(&payRows).Error; err != nil {
		return nil, err
	}
	byPay := []PaymentBreakdown{
		{PaymentMethod: "qris"}, {PaymentMethod: "va"},
	}
	for _, r := range payRows {
		for i := range byPay {
			if byPay[i].PaymentMethod == r.Method {
				byPay[i].Revenue = r.Revenue
				byPay[i].OrdersCount = r.OrdersCount
			}
		}
	}

	// --- Top 5 produk (terlaris, beserta profit) ---
	var topRows []struct {
		ProductID    uuid.UUID `gorm:"column:product_id"`
		ProductName  string    `gorm:"column:product_name"`
		QuantitySold int64     `gorm:"column:quantity_sold"`
		Revenue      int64     `gorm:"column:revenue"`
		COGS         int64     `gorm:"column:cogs"`
	}
	qtop := "SELECT oi.product_id, oi.product_name, COALESCE(SUM(oi.quantity),0) AS quantity_sold, " +
		"COALESCE(SUM(oi.subtotal),0) AS revenue, " +
		"COALESCE(SUM(oi.quantity * COALESCE(NULLIF(oi.cost_price, 0), p.cost_price, 0)), 0) AS cogs " +
		"FROM order_items oi JOIN orders o ON o.id = oi.order_id JOIN products p ON p.id = oi.product_id " +
		"WHERE o.status IN ('paid','processing','shipped','completed')"
	if hasRange {
		qtop += " AND o.created_at >= ? AND o.created_at < ?"
	}
	qtop += " AND o.deleted_at IS NULL AND oi.deleted_at IS NULL GROUP BY oi.product_id, oi.product_name ORDER BY quantity_sold DESC, revenue DESC LIMIT 5"
	if err := s.db.Raw(qtop, args...).Scan(&topRows).Error; err != nil {
		return nil, err
	}
	top := []TopProduct{}
	for _, r := range topRows {
		top = append(top, TopProduct{
			ProductID: r.ProductID, ProductName: r.ProductName,
			QuantitySold: r.QuantitySold, Revenue: r.Revenue, Profit: r.Revenue - r.COGS,
		})
	}

	// --- 10 transaksi terbaru (semua status, dalam rentang bila ada) ---
	var orders []domain.Order
	recentQ := s.db.Model(&domain.Order{})
	if hasRange {
		recentQ = recentQ.Where("created_at >= ? AND created_at < ?", start, end)
	}
	if err := recentQ.Order("created_at DESC").Limit(10).Find(&orders).Error; err != nil {
		return nil, err
	}
	recent := []RecentTransaction{}
	for _, o := range orders {
		recent = append(recent, RecentTransaction{
			OrderNumber: o.OrderNumber, CustomerName: o.CustomerName,
			OrderType: o.OrderType, TableNumber: o.TableNumber,
			Status: o.Status, Total: o.Total, CreatedAt: o.CreatedAt,
		})
	}

	return &ProfitAnalytics{
		Period: labelPeriod, StartDate: labelStart, EndDate: labelEnd,
		Summary: summary, DailySeries: series,
		ByOrderType: byType, ByPaymentMethod: byPay,
		TopProducts: top, RecentTransactions: recent,
	}, nil
}
