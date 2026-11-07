// Command bench: load generator untuk ekosistem SLARAS NUSA FOODLAB.
//
// Mode:
//
//	read     GET berulang (katalog / website) — ukur RPS + latensi.
//	checkout POST /api/v1/orders/guest-checkout serentak — ukur TPS + error.
//	upload   POST /api/v1/upload multipart serentak — ukur throughput upload.
//
// Pemakaian (dijalankan di VPS):
//
//	bench -mode read -url http://127.0.0.1:8080/api/v1/products -c 500 -d 30s
//	bench -mode checkout -url http://127.0.0.1:8080/api/v1 -c 200 -d 30s
//	bench -mode upload -url http://127.0.0.1:8080/api/v1 -file /tmp/bench/dummy.jpg -c 50 -d 30s
//
// Keluaran: satu baris JSON ringkasan ke stdout + baris manusiawi ke stderr.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type summary struct {
	Mode         string         `json:"mode"`
	Concurrency  int            `json:"concurrency"`
	DurationSec  float64        `json:"duration_sec"`
	Total        int64          `json:"total"`
	OK           int64          `json:"ok"`
	Errors       int64          `json:"errors"`
	StatusCodes  map[int]int64  `json:"status_codes"`
	OtherErrors  map[string]int `json:"other_errors"`
	RPS          float64        `json:"rps"`
	AvgMs        float64        `json:"avg_ms"`
	P50Ms        float64        `json:"p50_ms"`
	P95Ms        float64        `json:"p95_ms"`
	P99Ms        float64        `json:"p99_ms"`
	MaxMs        float64        `json:"max_ms"`
	BytesWritten int64          `json:"bytes_written,omitempty"`
}

var (
	mode        = flag.String("mode", "read", "read|checkout|upload")
	urlFlag     = flag.String("url", "", "base URL (read: URL lengkap; checkout/upload: base API, mis. http://127.0.0.1:8080/api/v1)")
	concurrency = flag.Int("c", 100, "jumlah worker serentak")
	duration    = flag.Duration("d", 30*time.Second, "durasi tiap tahap")
	fileFlag    = flag.String("file", "", "file gambar untuk mode upload")
	tokenFlag   = flag.String("token", "", "Bearer token admin (mode upload)")
	emailFlag   = flag.String("email", "admin@ecommerce.local", "email admin (login otomatis bila token kosong)")
	passFlag    = flag.String("pass", "Admin123!", "password admin")
)

var recGlobal *recorder

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(p / 100 * float64(len(sorted)))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func newClient(concurrency int) *http.Client {
	tr := &http.Transport{
		MaxIdleConns:        concurrency * 2,
		MaxIdleConnsPerHost: concurrency * 2,
		MaxConnsPerHost:     concurrency * 2,
		IdleConnTimeout:     90 * time.Second,
	}
	return &http.Client{Transport: tr, Timeout: 60 * time.Second}
}

type recorder struct {
	latencies []float64
	status    map[int]int64
	otherErr  map[string]int
	ok        int64
	err       int64
	written   int64
	mu        sync.Mutex
}

func (r *recorder) add(d time.Duration, status int, err error, written int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.latencies = append(r.latencies, float64(d.Microseconds())/1000)
	if err != nil {
		r.err++
		msg := err.Error()
		if len(msg) > 60 {
			msg = msg[:60]
		}
		r.otherErr[msg]++
		return
	}
	r.status[status]++
	if status >= 200 && status < 300 {
		r.ok++
	} else {
		r.err++
	}
	r.written += written
}

func (r *recorder) summary(mode string, c int, d time.Duration) summary {
	sort.Float64s(r.latencies)
	var sum float64
	for _, v := range r.latencies {
		sum += v
	}
	total := int64(len(r.latencies))
	avg := 0.0
	if total > 0 {
		avg = sum / float64(total)
	}
	maxMs := 0.0
	if total > 0 {
		maxMs = r.latencies[total-1]
	}
	return summary{
		Mode:         mode,
		Concurrency:  c,
		DurationSec:  d.Seconds(),
		Total:        total,
		OK:           r.ok,
		Errors:       r.err,
		StatusCodes:  r.status,
		OtherErrors:  r.otherErr,
		RPS:          float64(total) / d.Seconds(),
		AvgMs:        avg,
		P50Ms:        percentile(r.latencies, 50),
		P95Ms:        percentile(r.latencies, 95),
		P99Ms:        percentile(r.latencies, 99),
		MaxMs:        maxMs,
		BytesWritten: r.written,
	}
}

func runWithRec(workers int, d time.Duration, work func(w int, seq int64)) summary {
	var seq int64
	// time.After hanya mengirim SATU nilai (satu worker saja yang berhenti);
	// pakai channel yang di-close agar SEMUA worker berhenti bersamaan.
	done := make(chan struct{})
	time.AfterFunc(d, func() { close(done) })
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
				}
				n := atomic.AddInt64(&seq, 1)
				work(w, n)
			}
		}(w)
	}
	wg.Wait()
	return recGlobal.summary(*mode, workers, d)
}

func main() {
	flag.Parse()
	if *urlFlag == "" {
		fmt.Fprintln(os.Stderr, "flag -url wajib diisi")
		os.Exit(2)
	}
	client := newClient(*concurrency)
	recGlobal = &recorder{status: map[int]int64{}, otherErr: map[string]int{}}

	var s summary
	switch *mode {
	case "read":
		urls := strings.Split(*urlFlag, ",")
		s = runWithRec(*concurrency, *duration, func(w int, seq int64) {
			u := urls[int(seq)%len(urls)]
			start := time.Now()
			resp, err := client.Get(u)
			if err != nil {
				recGlobal.add(time.Since(start), 0, err, 0)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			recGlobal.add(time.Since(start), resp.StatusCode, nil, 0)
		})
	case "checkout":
		products := fetchProducts(client, *urlFlag)
		if len(products) == 0 {
			fmt.Fprintln(os.Stderr, "tidak ada produk untuk checkout")
			os.Exit(1)
		}
		s = runWithRec(*concurrency, *duration, func(w int, seq int64) {
			pid := products[int(seq)%len(products)]
			body := fmt.Sprintf(`{"customer_name":"Bench %d","customer_phone":"081200000000","order_type":"takeaway","items":[{"product_id":%q,"quantity":1}]}`, seq, pid)
			start := time.Now()
			resp, err := client.Post(*urlFlag+"/orders/guest-checkout", "application/json", strings.NewReader(body))
			if err != nil {
				recGlobal.add(time.Since(start), 0, err, 0)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			recGlobal.add(time.Since(start), resp.StatusCode, nil, 0)
		})
	case "upload":
		data, err := os.ReadFile(*fileFlag)
		if err != nil {
			fmt.Fprintln(os.Stderr, "baca file dummy:", err)
			os.Exit(1)
		}
		token := *tokenFlag
		if token == "" {
			token = login(client, *urlFlag, *emailFlag, *passFlag)
			if token == "" {
				fmt.Fprintln(os.Stderr, "login admin gagal")
				os.Exit(1)
			}
		}
		s = runWithRec(*concurrency, *duration, func(w int, seq int64) {
			var buf bytes.Buffer
			mw := multipart.NewWriter(&buf)
			fw, err := mw.CreateFormFile("file", fmt.Sprintf("bench-%d-%d.jpg", w, seq))
			if err != nil {
				return
			}
			fw.Write(data)
			mw.Close()
			req, _ := http.NewRequest(http.MethodPost, *urlFlag+"/upload", &buf)
			req.Header.Set("Content-Type", mw.FormDataContentType())
			req.Header.Set("Authorization", "Bearer "+token)
			start := time.Now()
			resp, err := client.Do(req)
			if err != nil {
				recGlobal.add(time.Since(start), 0, err, 0)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			recGlobal.add(time.Since(start), resp.StatusCode, nil, int64(len(data)))
		})
	default:
		fmt.Fprintln(os.Stderr, "mode tidak dikenal:", *mode)
		os.Exit(2)
	}

	out, _ := json.Marshal(s)
	fmt.Println(string(out))
	fmt.Fprintf(os.Stderr, "[%s c=%d] total=%d ok=%d err=%d rps=%.1f avg=%.1fms p50=%.1f p95=%.1f p99=%.1f max=%.1fms status=%v\n",
		s.Mode, s.Concurrency, s.Total, s.OK, s.Errors, s.RPS, s.AvgMs, s.P50Ms, s.P95Ms, s.P99Ms, s.MaxMs, s.StatusCodes)
}

func fetchProducts(client *http.Client, base string) []string {
	resp, err := client.Get(base + "/products?limit=100")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var env struct {
		Data struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil
	}
	ids := make([]string, 0, len(env.Data.Items))
	for _, it := range env.Data.Items {
		ids = append(ids, it.ID)
	}
	return ids
}

func login(client *http.Client, base, email, pass string) string {
	body := fmt.Sprintf(`{"email":%q,"password":%q}`, email, pass)
	resp, err := client.Post(base+"/auth/login", "application/json", strings.NewReader(body))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var env struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return ""
	}
	return env.Data.AccessToken
}
