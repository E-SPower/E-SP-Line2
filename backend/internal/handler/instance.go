package handler

import (
	"net/http"
	"strconv"

	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/gin-gonic/gin"
)

// ListInstances lists all instances
func ListInstances(instanceService *service.InstanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		instances, total, err := instanceService.List(limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  instances,
			"total": total,
		})
	}
}

// GetInstance gets an instance by ID
func GetInstance(instanceService *service.InstanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		instance, err := instanceService.GetByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
			return
		}

		c.JSON(http.StatusOK, instance)
	}
}

// CreateInstance creates a new instance
func CreateInstance(instanceService *service.InstanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req service.CreateInstanceRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get user ID from context
		userID, _ := c.Get("userID")
		req.UserID = userID.(string)

		instance, err := instanceService.Create(&req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, instance)
	}
}

// UpdateInstance updates an instance
func UpdateInstance(instanceService *service.InstanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req service.UpdateInstanceRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		instance, err := instanceService.Update(id, &req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, instance)
	}
}

// DeleteInstance deletes an instance
func DeleteInstance(instanceService *service.InstanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := instanceService.Delete(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "instance deleted"})
	}
}
