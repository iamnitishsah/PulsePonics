// Package apperr defines the error type that services return for expected
// failures (bad input, missing records, auth problems). HTTP handlers map the
// Kind to a status code; anything that is not an *Error is treated as a 500.
package apperr

import "errors"

type Kind int

const (
	KindInvalid Kind = iota + 1
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
	KindTooManyRequests
)

type Error struct {
	Kind    Kind
	Message string
	Field   string // optional: the request field that caused a KindInvalid error
}

func (e *Error) Error() string { return e.Message }

func Invalid(field, message string) *Error {
	return &Error{Kind: KindInvalid, Field: field, Message: message}
}

func Unauthorized(message string) *Error {
	return &Error{Kind: KindUnauthorized, Message: message}
}

func Forbidden(message string) *Error {
	return &Error{Kind: KindForbidden, Message: message}
}

func NotFound(message string) *Error {
	return &Error{Kind: KindNotFound, Message: message}
}

func Conflict(message string) *Error {
	return &Error{Kind: KindConflict, Message: message}
}

func TooManyRequests(message string) *Error {
	return &Error{Kind: KindTooManyRequests, Message: message}
}

// As returns the *Error inside err, if there is one.
func As(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}

// IsKind reports whether err is an *Error of the given kind.
func IsKind(err error, kind Kind) bool {
	e, ok := As(err)
	return ok && e.Kind == kind
}
