// Package domain holds the types shared by every layer: roles, the acting user
// and the error vocabulary that services return and the HTTP layer renders.
package domain

import (
	"errors"
	"fmt"
)

// Kind classifies an error so the HTTP layer can pick a status code.
type Kind int

const (
	KindInternal Kind = iota
	KindValidation
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
	KindWipLimit
	KindColumnNotEmpty
	KindRateLimited
	KindFileTooLarge
	KindUnsupportedMedia
	KindPayloadTooLarge
)

// Code is the stable machine-readable identifier sent to API clients.
func (k Kind) Code() string {
	switch k {
	case KindValidation:
		return "VALIDATION_ERROR"
	case KindUnauthorized:
		return "UNAUTHORIZED"
	case KindForbidden:
		return "FORBIDDEN"
	case KindNotFound:
		return "NOT_FOUND"
	case KindConflict:
		return "CONFLICT"
	case KindWipLimit:
		return "WIP_LIMIT_EXCEEDED"
	case KindColumnNotEmpty:
		return "COLUMN_NOT_EMPTY"
	case KindRateLimited:
		return "RATE_LIMITED"
	case KindFileTooLarge:
		return "FILE_TOO_LARGE"
	case KindUnsupportedMedia:
		return "UNSUPPORTED_MEDIA_TYPE"
	case KindPayloadTooLarge:
		return "PAYLOAD_TOO_LARGE"
	default:
		return "INTERNAL"
	}
}

// FieldError points at one invalid input field.
type FieldError struct {
	Field   string
	Message string
}

// Error is an expected failure with a message that is safe to show to the user.
type Error struct {
	Kind    Kind
	Message string
	Fields  []FieldError
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Kind.Code(), e.Message)
}

// E builds an Error of the given kind.
func E(kind Kind, message string) *Error {
	return &Error{Kind: kind, Message: message}
}

// Invalid builds a validation error for a single field.
func Invalid(field, message string) *Error {
	return &Error{
		Kind:    KindValidation,
		Message: "Некорректные данные",
		Fields:  []FieldError{{Field: field, Message: message}},
	}
}

// NotFound builds a not-found error naming the missing object.
func NotFound(what string) *Error {
	return E(KindNotFound, what+": объект не найден")
}

// ErrForbidden is returned when the actor lacks rights for the operation.
var ErrForbidden = E(KindForbidden, "Недостаточно прав")

// ErrUnauthorized is returned when credentials are missing or invalid.
var ErrUnauthorized = E(KindUnauthorized, "Требуется аутентификация")

// KindOf reports the kind of err, or KindInternal for unexpected errors.
func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return KindInternal
}
