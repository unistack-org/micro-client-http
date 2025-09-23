package http

import "fmt"

// Error is used when items in the error map do not implement the error interface and need to be wrapped.
type Error struct {
	err any
}

func (e *Error) Error() string {
	return fmt.Sprintf("%+v", e.err)
}
