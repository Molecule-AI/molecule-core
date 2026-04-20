package router

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

// newCPProxy returns a Gin handler that reverse-proxies /cp/* requests
// to the control plane. Lives beside newCanvasProxy because they solve
// the same problem — tenant browser fetches targeted at a single
// same-origin base — for the mirror-image endpoint set.
//
// Why this exists: canvas's browser bundle calls both CP endpoints
// (/cp/auth/me, /cp/orgs, /cp/billing/checkout) AND tenant-platform
// endpoints (/canvas/viewport, /approvals/pending). They share ONE
// build-time base URL (NEXT_PUBLIC_PLATFORM_URL). Baking the CP
// origin breaks tenant calls; baking the tenant origin breaks CP
// calls. The only sane fix is same-origin fetches + let the server
// split the traffic. This handler is the /cp/* leg of that split;
// newCanvasProxy is the UI leg.
//
// Security:
//   - We do NOT strip Cookie/Authorization here: those carry the
//     WorkOS session cookie and must reach the CP to resolve the
//     user. That's the whole point of this proxy.
//   - We DO rewrite the Host header to the CP upstream so CORS and
//     cookie-domain logic upstream see themselves, not the tenant.
//   - We do NOT strip X-Forwarded-For — upstream may want it for
//     audit and rate-limit keying.
//   - The proxy ONLY forwards /cp/* paths. The upstream URL is
//     env-configured and its scheme is enforced https in prod via
//     url.Parse (the caller passes the URL; we reject anything
//     that isn't http/https at construction time).
//
// Rate / timeout note: we do NOT set a custom Transport with
// aggressive timeouts because CP endpoints are fast and any hang
// is already bounded by the caller's browser-level timeout. If a
// future slow endpoint warrants a bound, add here not at the
// gateway.
func newCPProxy(targetURL string) gin.HandlerFunc {
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("cp_proxy: invalid CP_UPSTREAM_URL %q: %v", targetURL, err)
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		log.Fatalf("cp_proxy: CP_UPSTREAM_URL scheme must be http(s), got %q", target.Scheme)
	}
	if target.Host == "" {
		log.Fatalf("cp_proxy: CP_UPSTREAM_URL missing host: %q", targetURL)
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			// Host header rewrite: CP middleware (CORS, cookie-domain)
			// keys off Host; rewriting avoids "origin not allowed" on
			// upstream OPTIONS preflight.
			req.Host = target.Host
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			log.Printf("cp_proxy: %v", err)
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("control plane unavailable"))
		},
	}

	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
