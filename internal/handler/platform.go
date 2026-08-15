package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/e-spl/e-sp-line2/internal/service"
)

// ListPlatforms lists all platforms
func ListPlatforms(platformService *service.PlatformService) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		platforms, total, err := platformService.List(limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  platforms,
			"total": total,
		})
	}
}

// GetPlatform gets a platform by ID
func GetPlatform(platformService *service.PlatformService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		platform, err := platformService.GetByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "platform not found"})
			return
		}

		c.JSON(http.StatusOK, platform)
	}
}

// CreatePlatform creates a new platform
func CreatePlatform(platformService *service.PlatformService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req service.CreatePlatformRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		platform, err := platformService.Create(&req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, platform)
	}
}

// UpdatePlatform updates a platform
func UpdatePlatform(platformService *service.PlatformService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req service.UpdatePlatformRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		platform, err := platformService.Update(id, &req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, platform)
	}
}

// DeletePlatform deletes a platform
func DeletePlatform(platformService *service.PlatformService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := platformService.Delete(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "platform deleted"})
	}
}
