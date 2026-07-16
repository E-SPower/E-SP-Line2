package handler

import (
	"net/http"
	"strconv"

	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/gin-gonic/gin"
)

// ListCommands lists all commands
func ListCommands(commandService *service.CommandService) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		commands, total, err := commandService.List(limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  commands,
			"total": total,
		})
	}
}

// GetCommand gets a command by ID
func GetCommand(commandService *service.CommandService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		command, err := commandService.GetByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "command not found"})
			return
		}

		c.JSON(http.StatusOK, command)
	}
}

// CreateCommand creates a new command
func CreateCommand(commandService *service.CommandService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req service.CreateCommandRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		command, err := commandService.Create(&req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, command)
	}
}
