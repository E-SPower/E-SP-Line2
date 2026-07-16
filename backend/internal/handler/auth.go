package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/e-spl/e-sp-line2/internal/service"
)

// Register handles user registration
func Register(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req service.RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := authService.Register(&req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, resp)
	}
}

// Login handles user login
func Login(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req service.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := authService.Login(&req)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

// GetCurrentUser gets the current authenticated user
func GetCurrentUser(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		user, err := authService.GetUserByID(userID.(string))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}
