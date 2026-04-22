// Package metrics provides a lightweight Prometheus-format metrics endpoint
// for the Molecule AI platform. It requires no external dependencies — all
// serialization is done against the Prometheus text exposition format (v0.0.4)
// using the Go standard library.
//
// Exposed metrics:
//
//	molecule_http_requests_total{method,path,status}   - counter
//	molecule_http_request_duration_seconds{method,path} - counter (sum, for avg rate)
//	molecule_websocket_connections_active               - gauge
//	go_goroutines                                       - gauge
//	go_memstats_alloc_bytes                             - gauge
//	go_memstats_sys_bytes                               - gauge
//	go_memstats_heap_inuse_bytes                        - gauge
//	go_gc_duration_seconds_total                        - counter
package metrics

import (
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// reqKey indexes per-route request counts and latency sums.
type reqKey struct {
	method string
	path   string
	status int
}

var (
	mu            sync.RWMutex
	reqCounts     = map[reqKey]int64{}   // molecule_http_requests_total
	reqDurSums    = map[reqKey]float64{} // sum of durations (seconds)
	activeWSConns int64                  // molecule_websocket_connections_active
)

// Middleware records per-request counts and latency.
// Register this before route handlers in the Gin engine.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		duration := time.Since(start).Seconds()
		// Use the matched route pattern (e.g. "/workspaces/:id") so high-cardinality
		// workspace UUIDs don't explode the label space.
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}

		k := reqKey{
			method: c.Request.Method,
			path:   path,
			status: c.Writer.Status(),
		}

		mu.Lock()
		reqCounts[k]++
		reqDurSums[k] += duration
		mu.Unlock()
	}
}

// TrackWSConnect increments the active WebSocket connections gauge.
// Call from the WebSocket upgrade handler after a successful upgrade.
func TrackWSConnect() { atomic.AddInt64(&activeWSConns, 1) }

// TrackWSDisconnect decrements the active WebSocket connections gauge.
// Call from the WebSocket disconnect / cleanup path.
func TrackWSDisconnect() { atomic.AddInt64(&activeWSConns, -1) }

// Handler returns a Gin handler that serialises all collected metrics in
// Prometheus text exposition format (v0.0.4). Mount this at GET /metrics.
func Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)

		c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w := c.Writer
		w.WriteHeader(http.StatusOK)

		// ── Go runtime ─────────────────────────────────────────────────────
		writeln(w, "# HELP go_goroutines Number of goroutines currently running.")
		writeln(w, "# TYPE go_goroutines gauge")
		_, _ = fmt.Fprintf(w, "go_goroutines %d\n", runtime.NumGoroutine())

		writeln(w, "# HELP go_memstats_alloc_bytes Bytes of allocated heap objects.")
		writeln(w, "# TYPE go_memstats_alloc_bytes gauge")
		_, _ = fmt.Fprintf(w, "go_memstats_alloc_bytes %d\n", ms.Alloc)

		writeln(w, "# HELP go_memstats_sys_bytes Total bytes of memory obtained from the OS.")
		writeln(w, "# TYPE go_memstats_sys_bytes gauge")
		_, _ = fmt.Fprintf(w, "go_memstats_sys_bytes %d\n", ms.Sys)

		writeln(w, "# HELP go_memstats_heap_inuse_bytes Bytes in in-use heap spans.")
		writeln(w, "# TYPE go_memstats_heap_inuse_bytes gauge")
		fmt.Fprintf(w, "go_memstats_heap_inuse_bytes %d\n", ms.HeapInuse)

		writeln(w, "# HELP go_gc_duration_seconds_total Cumulative GC pause time.")
		writeln(w, "# TYPE go_gc_duration_seconds_total counter")
		fmt.Fprintf(w, "go_gc_duration_seconds_total %g\n", float64(ms.PauseTotalNs)/1e9)

		// ── Molecule AI HTTP ───────────────────────────────────────────────────
		writeln(w, "# HELP molecule_http_requests_total Total HTTP requests served, by method, path, and status.")
		writeln(w, "# TYPE molecule_http_requests_total counter")

		writeln(w, "# HELP molecule_http_request_duration_seconds_total Cumulative HTTP request duration in seconds.")
		writeln(w, "# TYPE molecule_http_request_duration_seconds_total counter")

		// Snapshot under lock, then write unlocked (avoids holding lock during slow HTTP writes)
		mu.RLock()
		countsCopy := make(map[reqKey]int64, len(reqCounts))
		for k, v := range reqCounts {
			countsCopy[k] = v
		}
		durCopy := make(map[reqKey]float64, len(reqDurSums))
		for k, v := range reqDurSums {
			durCopy[k] = v
		}
		mu.RUnlock()

		for k, count := range countsCopy {
			fmt.Fprintf(w,
				"molecule_http_requests_total{method=%q,path=%q,status=\"%d\"} %d\n",
				k.method, k.path, k.status, count,
			)
		}
		for k, sum := range durCopy {
			fmt.Fprintf(w,
				"molecule_http_request_duration_seconds_total{method=%q,path=%q,status=\"%d\"} %g\n",
				k.method, k.path, k.status, sum,
			)
		}

		// ── Molecule AI WebSocket ──────────────────────────────────────────────
		writeln(w, "# HELP molecule_websocket_connections_active Number of active WebSocket connections.")
		writeln(w, "# TYPE molecule_websocket_connections_active gauge")
		fmt.Fprintf(w, "molecule_websocket_connections_active %d\n", atomic.LoadInt64(&activeWSConns))
	}
}

func writeln(w http.ResponseWriter, s string) {
	_, _ = fmt.Fprintln(w, s)
}
