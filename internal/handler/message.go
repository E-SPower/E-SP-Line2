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
			respondError(c, err)
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
			respondError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "message acknowledged"})
	}
}

// CreateMessage is the message ingest entry point (B3).
//
// It binds a CreateMessageRequest, persists the event and then routes it through
// the hub. Idempotency is the service's job: a repeated idempotency_key returns
// the stored event with 200 instead of inserting a second row, and a missing key
// is replaced with a generated one (B4) so the UNIQUE index never sees "" twice.
func CreateMessage(messageService *service.MessageService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req service.CreateMessageRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		event, existed, err := messageService.CreateAndRouteIdempotent(&req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		status := http.StatusCreated
		if existed {
			// A replayed key is not a new resource; 200 tells the sender the
			// earlier attempt already landed, and keeps it from retrying.
			status = http.StatusOK
		}
		c.JSON(status, event)
	}
}
