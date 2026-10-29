package httpapi

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if len(id) < 8 || len(id) > 128 {
			var b [16]byte
			if _, err := rand.Read(b[:]); err == nil {
				id = hex.EncodeToString(b[:])
			} else {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}
func SecurityHeaders(http3 bool, address string) gin.HandlerFunc {
	_, port, _ := net.SplitHostPort(address)
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if http3 && port != "" {
			c.Header("Alt-Svc", `h3=":`+port+`"; ma=86400`)
		}
		c.Next()
	}
}
func Bearer(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token == "" {
			c.Next()
			return
		}
		got := c.GetHeader("Authorization")
		want := "Bearer " + token
		if subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
			c.Header("WWW-Authenticate", `Bearer realm="helix-admin"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized", "request_id": c.GetString("request_id")})
			return
		}
		c.Next()
	}
}
func Basic(username, password string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if username == "" {
			c.Next()
			return
		}
		u, p, ok := c.Request.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(u), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(p), []byte(password)) != 1 {
			c.Header("WWW-Authenticate", `Basic realm="Helix OPDS", charset="UTF-8"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}
func Error(c *gin.Context, status int, public string) {
	c.AbortWithStatusJSON(status, gin.H{"error": public, "request_id": c.GetString("request_id")})
}
