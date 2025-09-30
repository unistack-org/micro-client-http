package status

import (
	"errors"
	"fmt"
	"net/http"
)

// Status represents the outcome of an HTTP request in a style similar to gRPC status.
type Status struct {
	code    int    // HTTP status code
	message string // HTTP status text
	details any    // parsed error object
}

func New(statusCode int) *Status {
	return &Status{code: statusCode, message: http.StatusText(statusCode)}
}

func FromError(err error) (*Status, bool) {
	if err == nil {
		return nil, false
	}
	var e *Error
	if errors.As(err, &e) {
		return e.HTTPStatus(), true
	}
	return nil, false
}

func (s *Status) Code() int {
	return s.code
}

func (s *Status) Message() string {
	return s.message
}

func (s *Status) Details() any {
	return s.details
}

func (s *Status) WithDetails(details any) *Status {
	s.details = details
	return s
}

func (s *Status) String() string {
	return fmt.Sprintf("http error: code = %d desc = %s", s.Code(), s.Message())
}

func (s *Status) Err() error {
	return &Error{s: s}
}
