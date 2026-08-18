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

// GetInstanceLogs returns the instance's adapter process log.
// Query params:
//   - lines: number of trailing lines to return; "all" or missing negative
//     returns the full log (default: all).
//   - level: filter by level (debug/info/warning/error). Empty = all.
//   - keyword: optional substring filter (case-insensitive); space-separated
//     words are ANDed.
//   - from / to: optional time-range filters (RFC3339 or "2006-01-02 15:04:05").
func GetInstanceLogs(instanceService *service.InstanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		linesParam := c.DefaultQuery("lines", "all")
		maxLines := 0 // 0 = full log
		if linesParam != "" && linesParam != "all" {
			if n, err := strconv.Atoi(linesParam); err == nil && n > 0 {
				maxLines = n
			}
		}
		level := c.Query("level")
		keyword := c.Query("keyword")
		from := c.Query("from")
		to := c.Query("to")
		logs, err := instanceService.ReadLog(id, maxLines, level, keyword, from, to)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": logs})
	}
}

// GetInstanceLogHeatmap returns per-hour log level counts for an instance.
func GetInstanceLogHeatmap(instanceService *service.InstanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		level := c.Query("level")
		heatmap, err := instanceService.LogHeatmap(id, level)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": heatmap})
	}
}

// ClearInstanceLogs truncates the instance's log file.
func ClearInstanceLogs(instanceService *service.InstanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		removed, err := instanceService.ClearLog(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "logs cleared", "removed_bytes": removed})
	}
}

// StartInstance starts the instance's Python adapter process.
func StartInstance(instanceService *service.InstanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := instanceService.Start(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "instance started"})
	}
}

// StopInstance stops the instance's Python adapter process.
func StopInstance(instanceService *service.InstanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := instanceService.Stop(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "instance stopped"})
	}
}

// GetInstanceInitStatus returns the dependency installation (initialization)
// status of an instance. The WebUI polls this while the instance is
// "initializing" so it can show progress.
func GetInstanceInitStatus(instanceService *service.InstanceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		status := instanceService.InitStatus(id)
		c.JSON(http.StatusOK, gin.H{"data": status})
	}
}
