package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/e-spl/e-sp-line2/internal/service"
)

// ListAdapterCatalog returns the available adapters discovered by scanning
// adapters/*/adapter.yaml (hidden adapters excluded).
func ListAdapterCatalog(catalog *service.AdapterCatalog) gin.HandlerFunc {
	return func(c *gin.Context) {
		adapters := catalog.All(false)
		c.JSON(http.StatusOK, gin.H{
			"data":  adapters,
			"total": len(adapters),
		})
	}
}

// ListAdapters lists all adapters
func ListAdapters(adapterService *service.AdapterService) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		adapters, total, err := adapterService.List(limit, offset)
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

// GetAdapter gets an adapter by ID
func GetAdapter(adapterService *service.AdapterService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		adapter, err := adapterService.GetByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "adapter not found"})
			return
		}

		c.JSON(http.StatusOK, adapter)
	}
}

// CreateAdapter creates a new adapter
func CreateAdapter(adapterService *service.AdapterService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req service.CreateAdapterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		adapter, err := adapterService.Create(&req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, adapter)
	}
}

// UpdateAdapter updates an adapter
func UpdateAdapter(adapterService *service.AdapterService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req service.UpdateAdapterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		adapter, err := adapterService.Update(id, &req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, adapter)
	}
}

// DeleteAdapter deletes an adapter
func DeleteAdapter(adapterService *service.AdapterService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := adapterService.Delete(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "adapter deleted"})
	}
}

// StartAdapter starts an adapter
func StartAdapter(adapterService *service.AdapterService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := adapterService.StartAdapter(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "adapter started"})
	}
}

// StopAdapter stops an adapter
func StopAdapter(adapterService *service.AdapterService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := adapterService.StopAdapter(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "adapter stopped"})
	}
}
