package router

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

// newCanvasProxy returns a Gin handler that reverse-proxies all unmatched
// routes to the canvas Next.js server. Used in the combined tenant image
// (Dockerfile.tenant) where Go platform (:8080) and canvas (:3000) run in
// the same container.
//
// The proxy forwards the request path, query, and headers as-is. The Host
// header is rewritten to the canvas upstream so Next.js doesn't reject it
// (Next.js checks Host in dev mode). Response headers from canvas flow back
// to the client unchanged.
//
// Why NoRoute + proxy instead of nginx: one fewer process, one fewer config
// file, and the Go router already knows which routes are API routes. Any
// path not registered as an API route is a canvas page by elimination.
func newCanvasProxy(targetURL string) gin.HandlerFunc {
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("canvas_proxy: invalid CANVAS_PROXY_URL %q: %v", targetURL, err)
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host
			// Strip credentials that belong to the platform API — the canvas
			// Next.js server is internal and should never receive workspace
			// bearer tokens or session cookies (N2 / issue #451).
			req.Header.Del("Authorization")
			req.Header.Del("Cookie")
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			log.Printf("canvas_proxy: %v", err)
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("canvas unavailable"))
		},
	}

	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
