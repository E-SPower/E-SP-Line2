package handler

import (
	"net/http"
	"strconv"

	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/gin-gonic/gin"
)

// ListRoutes lists all route rules
func ListRoutes(routeService *service.RouteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		routes, total, err := routeService.List(limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  routes,
			"total": total,
		})
	}
}

// GetRoute gets a route rule by ID
func GetRoute(routeService *service.RouteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		route, err := routeService.GetByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
			return
		}

		c.JSON(http.StatusOK, route)
	}
}

// CreateRoute creates a new route rule
func CreateRoute(routeService *service.RouteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req service.CreateRouteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		route, err := routeService.Create(&req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, route)
	}
}

// UpdateRoute updates a route rule
func UpdateRoute(routeService *service.RouteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req service.UpdateRouteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		route, err := routeService.Update(id, &req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, route)
	}
}

// DeleteRoute deletes a route rule
func DeleteRoute(routeService *service.RouteService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := routeService.Delete(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "route deleted"})
	}
}
