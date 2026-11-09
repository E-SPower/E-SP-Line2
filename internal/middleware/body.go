package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// MaxBodyBytes caps the request body before it is read.
//
// Without it, `ShouldBindJSON` slurps whatever it is sent into memory, so a
// single client can drive the process out of memory. http.MaxBytesReader makes
// the underlying reader return *http.MaxBytesError once the limit is crossed,
// which the encoder renders as a 413.
//
// Only requests that can carry a body are wrapped: a GET with a bounded body is
// harmless, but wrapping also changes nothing for it, so the method check keeps
// the hot path untouched.
func MaxBodyBytes(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limit > 0 && methodHasBody(c.Request.Method) && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}

// methodHasBody reports whether an HTTP method normally carries a payload.
func methodHasBody(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// BodyTooLarge inspects a bind error and answers 413 itself when the cause was
// the size cap, reporting whether it handled the response.
//
// A handler that just returns 400 for every bind error reports "unexpected end
// of JSON input" for an over-size body, which sends the client hunting for a
// syntax problem instead of shrinking the payload.
func BodyTooLarge(c *gin.Context, err error) bool {
	var maxErr *http.MaxBytesError
	if !errors.As(err, &maxErr) {
		return false
	}
	c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
		"error": "request body too large",
	})
	return true
}
