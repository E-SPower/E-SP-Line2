package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthRequired returns a gin middleware that requires authentication
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header is required",
			})
			c.Abort()
			return
		}

		// Check Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization format",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// TODO: Validate token using auth service
		// For now, just extract user info from token (placeholder)
		_ = token // Use token to avoid unused variable error

		// Set user info in context (placeholder)
		c.Set("userID", "user-123")
		c.Set("username", "testuser")
		c.Set("role", "user")

		c.Next()
	}
}

// AdminRequired returns a gin middleware that requires admin role
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "admin access required",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
