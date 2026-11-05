// Command bench is a dependency-free HTTP load generator (stdlib only).
// Usage: go run ./benchmark/bench.go -url http://localhost:8080 -path /api/v1/products -concurrency 50 -duration 15s [-token X] [-method POST -body ...] [-xff-prefix 203.0.113]
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	url := flag.String("url", "http://localhost:8080", "base URL")
	path := flag.String("path", "/api/v1/products", "request path")
	concurrency := flag.Int("concurrency", 50, "concurrent workers")
	duration := flag.Duration("duration", 15*time.Second, "test duration")
	token := flag.String("token", "", "bearer token (optional)")
	method := flag.String("method", "GET", "HTTP method")
	body := flag.String("body", "", "request body")
	xffPrefix := flag.String("xff-prefix", "", "X-Forwarded-For prefix, e.g. 203.0.113 (worker i gets <prefix>.<i>)")
	flag.Parse()

	client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: *concurrency * 2, MaxConnsPerHost: *concurrency * 2}}
	var total, ok int64
	statuses := map[int]int64{}
	var mu sync.Mutex
	lat := make([]time.Duration, 0, 100000)
	var latMu sync.Mutex

	deadline := time.Now().Add(*duration)
	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < *concurrency; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for time.Now().Before(deadline) {
				var rdr io.Reader
				if *body != "" {
					rdr = bytes.NewBufferString(*body)
				}
				req, err := http.NewRequest(*method, *url+*path, rdr)
				if err != nil {
					return
				}
				if *body != "" {
					req.Header.Set("Content-Type", "application/json")
				}
				if *token != "" {
					req.Header.Set("Authorization", "Bearer "+*token)
				}
				if *xffPrefix != "" {
					req.Header.Set("X-Forwarded-For", fmt.Sprintf("%s.%d", *xffPrefix, worker))
				}
				t0 := time.Now()
				resp, err := client.Do(req)
				d := time.Since(t0)
				latMu.Lock()
				lat = append(lat, d)
				latMu.Unlock()
				if err != nil {
					mu.Lock()
					statuses[0]++
					mu.Unlock()
					atomic.AddInt64(&total, 1)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				mu.Lock()
				statuses[resp.StatusCode]++
				mu.Unlock()
				atomic.AddInt64(&total, 1)
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					atomic.AddInt64(&ok, 1)
				}
			}
		}(w)
	}
	wg.Wait()
	elapsed := time.Since(start)
	sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
	pct := func(p float64) time.Duration {
		if len(lat) == 0 {
			return 0
		}
		i := int(float64(len(lat))*p) - 1
		if i < 0 {
			i = 0
		}
		if i >= len(lat) {
			i = len(lat) - 1
		}
		return lat[i]
	}
	var sum time.Duration
	for _, d := range lat {
		sum += d
	}
	avg := time.Duration(0)
	if len(lat) > 0 {
		avg = sum / time.Duration(len(lat))
	}
	errPct := 0.0
	if total > 0 {
		errPct = float64(total-ok) / float64(total) * 100
	}
	fmt.Printf("url=%s%s method=%s concurrency=%d duration=%s\n", *url, *path, *method, *concurrency, elapsed.Round(time.Millisecond))
	fmt.Printf("total=%d ok(2xx)=%d errors=%d error_pct=%.2f%%\n", total, ok, total-ok, errPct)
	fmt.Printf("status breakdown: %v\n", statuses)
	fmt.Printf("rps=%.1f avg=%.2fms p50=%.2fms p95=%.2fms p99=%.2fms\n",
		float64(total)/elapsed.Seconds(),
		float64(avg.Microseconds())/1000,
		float64(pct(0.50).Microseconds())/1000,
		float64(pct(0.95).Microseconds())/1000,
		float64(pct(0.99).Microseconds())/1000)
	_ = os.Stdout
}
