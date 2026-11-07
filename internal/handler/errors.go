package handler

import (
	"errors"
	"net/http"

	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Error bodies are deliberately generic for 404/500: the old handlers echoed
// raw storage text ("record not found") as a 400 body, which told a client
// nothing about what to do. Service-level detail is preserved by the wrapped
// sentinel errors and shows up in the server log.
const (
	msgNotFound = "resource not found"
	msgConflict = "resource conflict"
	msgInternal = "internal server error"
)

// respondError maps a service error onto the HTTP status the API contract
// promises: 404 not found, 409 conflict, 400 invalid input, 500 anything else.
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": msgNotFound})
	case errors.Is(err, service.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": errorDetail(err, msgConflict)})
	case errors.Is(err, service.ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{"error": errorDetail(err, msgValidation)})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorDetail(err, msgInternal)})
	}
}

const msgValidation = "invalid request"

// errorDetail strips the sentinel prefix ("resource conflict: ") from a wrapped
// error so the client sees "username already exists" and not the marker.
func errorDetail(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	msg := err.Error()
	for _, prefix := range []string{
		service.ErrNotFound.Error() + ": ",
		service.ErrConflict.Error() + ": ",
		service.ErrValidation.Error() + ": ",
	} {
		if len(msg) > len(prefix) && msg[:len(prefix)] == prefix {
			return msg[len(prefix):]
		}
	}
	if msg == "" {
		return fallback
	}
	return msg
}
