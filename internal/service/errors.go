package service

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// Sentinel errors that carry the HTTP semantics of a service failure to the
// handler layer. Handlers map them with respondError; without them the only
// signal available was the error string, which made every failure look like a
// 400 (or a 500) and made "record not found" leak into response bodies.
//
// Wrap them so the concrete explanation stays readable while errors.Is keeps
// working:
//
//	fmt.Errorf("%w: platform code already exists", ErrConflict)
var (
	// ErrNotFound means the addressed resource does not exist. Repository
	// gorm.ErrRecordNotFound values carry the same meaning and are mapped to
	// 404 as well.
	ErrNotFound = errors.New("resource not found")

	// ErrConflict means the request is well-formed but clashes with existing
	// state (duplicate unique key, second start of a running instance, ...).
	ErrConflict = errors.New("resource conflict")

	// ErrUnauthorized means the caller could not be authenticated (bad
	// credentials, disabled account).
	ErrUnauthorized = errors.New("unauthenticated")

	// ErrInvalid means the request body or parameters are semantically wrong.
	// Binding/JSON failures are handled by the handlers directly; this sentinel
	// is for checks that only the service layer can make.
	ErrInvalid = errors.New("invalid request")
)

// notFound converts a repository lookup failure into ErrNotFound-wrapped error
// text, leaving real database failures untouched (so they still map to 500).
func notFound(kind, id string, err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%w: %s %q", ErrNotFound, kind, id)
	}
	return err
}
