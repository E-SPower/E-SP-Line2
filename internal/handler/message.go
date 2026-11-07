package handler

import (
	"net/http"
	"strconv"

	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/gin-gonic/gin"
)

// ListMessages lists all messages
func ListMessages(messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		messages, total, err := messageService.List(limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  messages,
			"total": total,
		})
	}
}

// GetMessage gets a message by ID
func GetMessage(messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		message, err := messageService.GetByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
			return
		}

		c.JSON(http.StatusOK, message)
	}
}

// AckMessage acknowledges a message
func AckMessage(messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := messageService.Ack(id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "message acknowledged"})
	}
}

// CreateMessage ingests an inbound message over HTTP.
//
// This is the non-WebSocket entry point: the body is bound to
// CreateMessageRequest, persisted, and handed to the hub for routing. It answers
// 201 for a new message and 200 with the existing row when the idempotency key
// was seen before (so a retrying caller can tell the cases apart).
func CreateMessage(messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req service.CreateMessageRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// A caller-supplied key that already exists must be answered with the
		// original row, not a 500 from the unique index.
		if req.IdempotencyKey != "" {
			if existing, err := messageService.FindByIdempotencyKey(req.IdempotencyKey); err == nil && existing != nil {
				c.JSON(http.StatusOK, existing)
				return
			}
		}

		message, err := messageService.Create(&req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, message)
	}
}
