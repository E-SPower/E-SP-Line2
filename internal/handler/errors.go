package handler

import (
	"errors"
	"net/http"

	"github.com/e-spl/e-sp-line2/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// respondError maps a service-layer error onto the single HTTP status that
// describes it, then writes the JSON error body. Every error branch of a
// handler that touches stored state goes through here so the same failure
// always produces the same status code:
//
//	404 — gorm.ErrRecordNotFound / service.ErrNotFound (missing resource)
//	401 — service.ErrUnauthorized (bad credentials, disabled account)
//	409 — service.ErrConflict (duplicate unique key, already-running start)
//	400 — service.ErrInvalid (well-formed request, wrong meaning)
//	500 — anything else (database, filesystem, unexpected)
//
// status overrides the mapping for domain-specific cases a caller knows
// better than the error does (e.g. a service that reports every failure as an
// opaque error but whose only 404-able branch is "id is unknown"); pass 0 to
// use the mapping.
func respondError(c *gin.Context, err error, status int) {
	if status == 0 {
		status = statusForError(err)
	}
	c.JSON(status, gin.H{"error": errorMessage(err, status)})
}

// statusForError classifies err as described on respondError.
func statusForError(err error) int {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound), errors.Is(err, service.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, service.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, service.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, service.ErrInvalid):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// errorMessage renders the response body for err. Sentinel wrappers are
// stripped so "resource conflict: platform code exists" becomes the readable
// "platform code exists"; a not-found result always says what was not found.
func errorMessage(err error, status int) string {
	if status == http.StatusNotFound {
		return "resource not found"
	}
	for _, sentinel := range []error{service.ErrUnauthorized, service.ErrConflict, service.ErrInvalid, service.ErrNotFound} {
		if errors.Is(err, sentinel) {
			msg := err.Error()
			// errors.Unwrap only strips one level, which is Sid's wrapping here.
			if unwrapped := errors.Unwrap(err); unwrapped != nil {
				msg = unwrapped.Error()
			}
			return msg
		}
	}
	return err.Error()
}
