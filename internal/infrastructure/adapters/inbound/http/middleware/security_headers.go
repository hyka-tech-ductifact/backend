package middleware

import "github.com/gin-gonic/gin"

// SecurityHeadersMiddleware adds HTTP security headers to every response.
// These headers instruct browsers to enable built-in protections against
// common web attacks (XSS, clickjacking, MIME sniffing, etc.).
//
// This middleware should be registered early in the chain (after recovery)
// so that ALL responses — even error responses — include these headers.
//
// Headers set:
//   - Strict-Transport-Security (HSTS): forces HTTPS for 1 year
//   - X-Content-Type-Options: prevents MIME type sniffing
//   - X-Frame-Options: prevents clickjacking via iframes
//   - Content-Security-Policy (CSP): restricts resource loading origins
//   - Referrer-Policy: controls how much referrer info is sent
//   - X-XSS-Protection: legacy XSS filter (for older browsers)
//   - Permissions-Policy: disables access to sensitive browser APIs
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()

		// HSTS — tell browsers to ONLY use HTTPS for this domain.
		// max-age=31536000 = 1 year; includeSubDomains covers all subdomains.
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Prevent browsers from guessing (sniffing) the MIME type.
		// Forces the browser to respect the Content-Type header we send.
		h.Set("X-Content-Type-Options", "nosniff")

		// Prevent the page from being embedded in an iframe (clickjacking protection).
		// DENY = not even same-origin iframes are allowed.
		h.Set("X-Frame-Options", "DENY")

		// CSP — restrict where the browser can load resources from.
		// default-src 'self' = only allow resources from our own origin.
		// This is a strict baseline; loosen per-directive as needed
		// (e.g. allow a CDN for styles, or 'unsafe-inline' for Swagger UI).
		h.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")

		// Control how much URL info is sent in the Referer header.
		// strict-origin-when-cross-origin = send full path for same-origin,
		// only the origin (domain) for cross-origin requests.
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Legacy XSS filter for older browsers (Chrome < 78, IE).
		// Modern browsers ignore this, but it doesn't hurt to include it.
		h.Set("X-XSS-Protection", "1; mode=block")

		// Disable access to powerful browser APIs we don't use.
		// Prevents malicious scripts from accessing camera, mic, geolocation, etc.
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		c.Next()
	}
}
