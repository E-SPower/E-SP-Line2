package service

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// Sentinel errors describing *why* a service call failed. Handlers translate
// them into HTTP status codes (404 / 409 / 400 / 500) so the API stops
// collapsing "not found", "conflict" and "internal error" into one status.
//
// Wrap them, never compare by string: callers use errors.Is.
var (
	// ErrNotFound marks a lookup or write that matched no row.
	ErrNotFound = errors.New("resource not found")
	// ErrConflict marks a write that collides with existing state
	// (duplicate username, duplicate platform code, last-admin guards).
	ErrConflict = errors.New("resource conflict")
	// ErrValidation marks a well-formed request the service refused to apply.
	ErrValidation = errors.New("validation failed")
)

// wrapNotFound converts a gorm.ErrRecordNotFound into an ErrNotFound that
// carries the offending entity and id. Other errors pass through untouched.
func wrapNotFound(err error, format string, a ...any) error {
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return fmt.Errorf("%w: %s", ErrNotFound, fmt.Sprintf(format, a...))
}

// conflictf builds an ErrConflict.
func conflictf(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrConflict, fmt.Sprintf(format, a...))
}

// validationf builds an ErrValidation.
func validationf(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrValidation, fmt.Sprintf(format, a...))
}
