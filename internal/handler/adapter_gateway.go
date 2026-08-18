package handler

import (
	"net/http"
	"strconv"

	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/gin-gonic/gin"
)

// ListGatewayAdapters lists all adapter (接入器) entities.
func ListGatewayAdapters(gateway *service.AdapterGatewayService) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		adapters, total, err := gateway.List(limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  adapters,
			"total": total,
		})
	}
}

// CreateGatewayAdapter creates a new adapter (接入器) entity.
func CreateGatewayAdapter(gateway *service.AdapterGatewayService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req service.CreateGatewayAdapterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userID, _ := c.Get("userID")
		createdBy, _ := userID.(string)

		adapter, err := gateway.Create(&req, createdBy)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, adapter)
	}
}

// GetGatewayAdapter gets an adapter by ID.
func GetGatewayAdapter(gateway *service.AdapterGatewayService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		adapter, err := gateway.GetByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "adapter not found"})
			return
		}
		c.JSON(http.StatusOK, adapter)
	}
}

// UpdateGatewayAdapter updates an adapter by ID.
func UpdateGatewayAdapter(gateway *service.AdapterGatewayService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req service.UpdateGatewayAdapterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		adapter, err := gateway.Update(id, &req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, adapter)
	}
}

// DeleteGatewayAdapter deletes an adapter by ID.
func DeleteGatewayAdapter(gateway *service.AdapterGatewayService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := gateway.Delete(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "adapter deleted"})
	}
}

// ListAdapterConnections lists all adapter connections.
func ListAdapterConnections(gateway *service.AdapterGatewayService) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		conns, total, err := gateway.ListConnections(limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  conns,
			"total": total,
		})
	}
}

// ListAdapterConnectionsByAdapter lists connections for a specific adapter.
func ListAdapterConnectionsByAdapter(gateway *service.AdapterGatewayService) gin.HandlerFunc {
	return func(c *gin.Context) {
		adapterID := c.Param("id")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		conns, err := gateway.ListConnectionsByAdapter(adapterID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  conns,
			"total": len(conns),
		})
	}
}
