package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// normalizeOrigin menyeragamkan origin agar "http://localhost:5173/" dan
// "http://LOCALHOST:5173" cocok dengan header Origin yang dikirim browser.
func normalizeOrigin(origin string) string {
	return strings.ToLower(strings.TrimRight(strings.TrimSpace(origin), "/"))
}

// CORS mengizinkan browser dari origin yang terdaftar memanggil API ini.
//
// Presign upload dipanggil langsung dari React di origin Vite, bukan dari
// domain API, jadi preflight OPTIONS harus dijawab sebelum Gin mencari route.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if normalized := normalizeOrigin(origin); normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))

		if origin != "" {
			if _, ok := allowed[normalizeOrigin(origin)]; ok {
				header := c.Writer.Header()
				header.Set("Access-Control-Allow-Origin", origin)
				header.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				header.Set("Access-Control-Allow-Headers", "Content-Type")
				header.Set("Access-Control-Max-Age", "600")
				header.Add("Vary", "Origin")

				if c.Request.Method == http.MethodOptions {
					c.AbortWithStatus(http.StatusNoContent)
					return
				}
			}
		}

		c.Next()
	}
}
