package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders returns a Gin middleware that sets standard HTTP security
// headers on every response to mitigate common web-application attacks:
//
//   - X-Content-Type-Options: nosniff                        — prevents MIME-type sniffing
//   - X-Frame-Options: DENY                                  — blocks iframe embedding (clickjacking)
//   - Content-Security-Policy: default-src 'self'            — restricts resource loading to same origin
//   - Strict-Transport-Security: max-age=31536000; includeSubDomains — enforces HTTPS for 1 year
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	}
}
