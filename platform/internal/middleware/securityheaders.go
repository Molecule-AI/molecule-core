package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders returns a Gin middleware that sets standard HTTP security
// headers on every response to mitigate common web-application attacks:
//
//   - X-Content-Type-Options: nosniff                        — prevents MIME-type sniffing
//   - X-Frame-Options: DENY                                  — blocks iframe embedding (clickjacking)
//   - Content-Security-Policy: default-src 'self'            — restricts resource loading to same origin
//   - Strict-Transport-Security: max-age=31536000; includeSubDomains — enforces HTTPS for 1 year
//   - Referrer-Policy: strict-origin-when-cross-origin       — avoids leaking full paths/queries in Referer
//   - Permissions-Policy: camera=(), microphone=(), geolocation=() — denies sensor access for embedded content
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// #282: these two were documented in CLAUDE.md but missing from
		// the middleware. Referrer-Policy prevents browsers from leaking
		// the full Referer URL to cross-origin resources (which can
		// expose internal paths/queries). Permissions-Policy denies
		// sensor access by default — especially relevant because the
		// canvas embeds iframes for Langfuse traces.
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// CSP: only apply to API responses. Canvas-proxied routes
		// (NoRoute → reverse-proxy to Next.js) serve HTML with inline
		// scripts + styles that `default-src 'self'` blocks. Next.js
		// sets its own CSP via <meta> tags. The Go middleware should
		// not override it for proxied HTML responses.
		//
		// Detection: API routes are registered explicitly in the router;
		// canvas-proxied routes hit NoRoute. We can't detect NoRoute
		// before c.Next() fires, so instead we check the response
		// Content-Type after Next() — but that's too late for headers.
		//
		// Simpler: apply a permissive CSP that allows Next.js to work.
		// 'unsafe-inline' is needed for Next.js standalone builds that
		// inject inline scripts for hydration. 'unsafe-eval' was dropped
		// after confirming production canvas renders without it.
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: blob:; "+
				"connect-src 'self' ws: wss:; "+
				"font-src 'self' data:")
		c.Next()
	}
}
