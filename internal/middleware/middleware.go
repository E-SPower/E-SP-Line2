package middleware

import (
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
// The allowed origin is echoed back instead of a blanket "*". That is what
// gives `Access-Control-Allow-Credentials: true` any effect — paired with "*"
// a browser ignores the credentials flag entirely, which is exactly the
// incoherence this replaced. With an allow-list the pair is consistent, so a
// future move to Cookie-carried sessions does not silently lose the Cookie.
//
// The session itself still travels in the Authorization header, not a Cookie,
// so a same-origin deployment does not need CORS at all — but an operator
// running the Vite dev server (or a separate UI host) does, and this is where
// that is decided. policy is shared with the WebSocket upgrader.
func CORS(policy *OriginPolicy) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := origin != "" && policy.AllowsRequest(c.Request)

		// Always Vary: Origin, otherwise a cache that served a response to one
		// origin keeps serving it to the next.
		c.Header("Vary", "Origin")

		switch {
		case allowed:
			// Echo, even under a "*" policy: the echoed form keeps the
			// credentials flag meaningful.
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		case origin == "" && policy.AllowsAny():
			// No Origin to echo (curl, a native client) — send the literal.
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
