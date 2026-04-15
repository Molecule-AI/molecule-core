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
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		// #282: these two were documented in CLAUDE.md but missing from
		// the middleware. Referrer-Policy prevents browsers from leaking
		// the full Referer URL to cross-origin resources (which can
		// expose internal paths/queries). Permissions-Policy denies
		// sensor access by default — especially relevant because the
		// canvas embeds iframes for Langfuse traces.
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Next()
	}
}
