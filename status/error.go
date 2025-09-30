package status

// Error is a thin wrapper around Status that implements the error interface.
type Error struct {
	s *Status
}

func (e *Error) Error() string {
	return e.s.String()
}

func (e *Error) HTTPStatus() *Status {
	return e.s
}
