package errs

import "errors"

var (
	// ErrNotImplemented indicates that the functionality is pending implementation.
	ErrNotImplemented = errors.New("not implemented")
)
