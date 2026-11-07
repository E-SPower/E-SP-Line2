package handler

import (
	"net/http"
	"strconv"

	"github.com/e-spl/e-sp-line2/internal/message"
	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/e-spl/e-sp-line2/pkg/logger"
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

// CreateCommand creates a new command and immediately dispatches it to the
// adapter instance that owns the conversation.
//
// The row is written first (so a crash between the insert and the socket write
// leaves a retryable "created" row rather than a lost command), then handed to
// the hub. Success moves it to "sent"; a missing adapter connection or a failed
// write goes through MarkFailed, which retries up to MaxRetries before settling
// on "failed". The pending-command scanner picks up anything left in
// created/queued/retrying.
func CreateCommand(commandService *service.CommandService, hub *message.Hub) gin.HandlerFunc {
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

		if err := commandService.DispatchCommand(hub, command); err != nil {
			logger.Warn("Command dispatch failed",
				logger.String("command_id", command.ID),
				logger.String("instance_id", command.InstanceID),
				logger.String("error", err.Error()))
		}

		// Re-read so the response carries the status the dispatch produced.
		if updated, err := commandService.GetByID(command.ID); err == nil {
			command = updated
		}

		c.JSON(http.StatusCreated, command)
	}
}
