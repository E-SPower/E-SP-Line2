package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/e-spl/e-sp-line2/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Logger returns a gin middleware for logging requests
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		if len(c.Errors) > 0 {
			logger.Error(c.Errors.String(),
				"status", status,
				"method", c.Request.Method,
				"path", path,
				"query", query,
				"ip", c.ClientIP(),
				"user-agent", c.Request.UserAgent(),
				"latency", latency.String(),
			)
		} else if status >= 500 {
			logger.Error("Server error",
				"status", status,
				"method", c.Request.Method,
				"path", path,
				"query", query,
				"ip", c.ClientIP(),
				"user-agent", c.Request.UserAgent(),
				"latency", latency.String(),
			)
		} else if status >= 400 {
			logger.Warn("Client error",
				"status", status,
				"method", c.Request.Method,
				"path", path,
				"query", query,
				"ip", c.ClientIP(),
				"user-agent", c.Request.UserAgent(),
				"latency", latency.String(),
			)
		} else {
			logger.Info("Request",
				"status", status,
				"method", c.Request.Method,
				"path", path,
				"query", query,
				"ip", c.ClientIP(),
				"user-agent", c.Request.UserAgent(),
				"latency", latency.String(),
			)
		}
	}
}

// CORS returns a gin middleware for CORS handling.
//
// The allowed Origin is echoed back instead of a fixed "*". That is what makes
// the `Access-Control-Allow-Credentials: true` header below valid at all: a
// wildcard Origin and credentials are mutually exclusive per the fetch spec, so
// a future switch to Cookie sessions would otherwise silently drop them. The
// allowlist is the same one the WebSocket upgrader checks (see OriginAllowed),
// so CORS and WS can never drift apart.
//
// Requests without an Origin header (curl, the Python bridges, health checks)
// are unaffected: they get no CORS headers and are not gated by this
// middleware. A non-allowlisted Origin still reaches the handler, but without
// an Access-Control-Allow-Origin header the browser blocks the response.
func CORS(allowedOrigin func(string) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			// The response varies by Origin, so caches must key on it.
			c.Writer.Header().Add("Vary", "Origin")
			if allowedOrigin == nil || allowedOrigin(origin) {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
				c.Header("Access-Control-Expose-Headers", "Content-Length")
			}
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// MaxBodyBytes limits how many bytes a request body may carry. It is installed
// globally so no individual handler can forget it; handlers that need a
// different cap can shrink it further via http.MaxBytesReader themselves.
//
// A declared Content-Length over the limit is refused up front with 413 instead
// of letting the handler read into the error: without that pre-check the bind
// failure surfaces as the handler's own 400 ("unexpected EOF" in the JSON
// decoder), which reads like a malformed body rather than an oversized one.
// A lying client is still capped by the MaxBytesReader.
func MaxBodyBytes(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limit <= 0 {
			c.Next()
			return
		}
		if c.Request.ContentLength > limit {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("request body too large: limit is %d bytes", limit),
			})
			return
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}
