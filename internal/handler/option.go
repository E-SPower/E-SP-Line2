package handler

import (
	"net/http"

	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/gin-gonic/gin"
)

// ListOptions returns all registered form option groups.
func ListOptions(optionService *service.OptionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		groups := optionService.AllGroups()
		c.JSON(http.StatusOK, gin.H{
			"data":  groups,
			"total": len(groups),
		})
	}
}

// GetOption returns a single option group identified by key.
func GetOption(optionService *service.OptionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("key")
		group, ok := optionService.Get(key)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "option group not found"})
			return
		}
		c.JSON(http.StatusOK, group)
	}
}
