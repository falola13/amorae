// Package apperr defines the error type every layer above the database uses.
// Handlers never see a raw pgx/sql error or send err.Error() to a client —
// they see a Kind, which httpx maps to exactly one HTTP status.
package apperr

import (
	"errors"
	"time"
)

// Kind classifies an error for the purpose of choosing an HTTP status. It
// deliberately has nothing to do with HTTP itself, so this package stays
// importable from services and repositories that must never import net/http.
type Kind int

const (
	// Zero value on purpose: an error that forgets to set a Kind fails closed
	// as "internal" (500), never something more permissive.
	KindInternal Kind = iota
	KindInvalid
	KindUnauthenticated
	KindForbidden
	KindNotFound
	KindConflict
	KindRateLimited
)

// Error is the application's error type. Code is machine-readable, Message
// is safe to show a user, Fields carries per-field validation messages, and
// Err is the underlying cause kept for logging/Unwrap but never rendered.
type Error struct {
	Kind       Kind
	Code       string
	Message    string
	Fields     map[string]string
	RetryAfter time.Duration
	Err        error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Code + ": " + e.Message + ": " + e.Err.Error()
	}
	return e.Code + ": " + e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

// As reports whether err is (or wraps) an *Error, mirroring the errors.As
// contract with a signature that doesn't force call sites to declare a
// target variable first.
func As(err error) (*Error, bool) {
	var target *Error
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

func Invalid(code, message string) *Error {
	return &Error{Kind: KindInvalid, Code: code, Message: message}
}

// Validation builds the one Invalid error every handler returns when field
// checks fail, so every 400 body in the API has the same code and message
// regardless of which fields tripped.
func Validation(fields map[string]string) *Error {
	return &Error{
		Kind:    KindInvalid,
		Code:    "validation_failed",
		Message: "Some fields are invalid.",
		Fields:  fields,
	}
}

func Unauthenticated(code, message string) *Error {
	return &Error{Kind: KindUnauthenticated, Code: code, Message: message}
}

func Forbidden(code, message string) *Error {
	return &Error{Kind: KindForbidden, Code: code, Message: message}
}

func NotFound(code, message string) *Error {
	return &Error{Kind: KindNotFound, Code: code, Message: message}
}

func Conflict(code, message string) *Error {
	return &Error{Kind: KindConflict, Code: code, Message: message}
}

// RateLimited means "too many attempts, wait retryAfter". The message is the
// same wherever it's raised, so a limit never reveals *which* limit tripped
// (e.g. whether an email address exists).
func RateLimited(retryAfter time.Duration) *Error {
	return &Error{
		Kind:       KindRateLimited,
		Code:       "rate_limited",
		Message:    "Too many attempts. Please wait a moment and try again.",
		RetryAfter: retryAfter,
	}
}

// Internal wraps an unexpected error. The wrapped error is never shown to a
// client (see httpx.Error) — it exists so the caller can log it.
func Internal(err error) *Error {
	return &Error{Kind: KindInternal, Code: "internal_error", Message: "Something went wrong. Please try again.", Err: err}
}
