// Package errorsx defines shared sentinel errors and wrapping helpers.
package errorsx

import "errors"

var (
	// ErrNotFound indicates a requested resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrInvalidArgument indicates bad client input.
	ErrInvalidArgument = errors.New("invalid argument")
	// ErrUnauthenticated indicates missing/invalid credentials.
	ErrUnauthenticated = errors.New("unauthenticated")
	// ErrPermissionDenied indicates authorization failure.
	ErrPermissionDenied = errors.New("permission denied")
	// ErrConflict indicates a state conflict (e.g. duplicate).
	ErrConflict = errors.New("conflict")
	// ErrInternal indicates an unexpected internal failure.
	ErrInternal = errors.New("internal error")
)
