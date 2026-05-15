package status_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"go.unistack.org/micro-client-http/v5/status"
)

type fakeError struct{ s *status.Status }

func (fe *fakeError) Error() string {
	return fe.s.String()
}

func TestNew(t *testing.T) {
	s := status.New(http.StatusNotFound)
	require.Equal(t, http.StatusNotFound, s.Code())
	require.Equal(t, "Not Found", s.Message())
}

func TestFromError(t *testing.T) {
	tests := []struct {
		name       string
		input      error
		wantStatus *status.Status
		wantOK     bool
	}{
		{
			name:       "nil error",
			input:      nil,
			wantStatus: nil,
			wantOK:     false,
		},
		{
			name:       "simple error",
			input:      errors.New("some error"),
			wantStatus: nil,
			wantOK:     false,
		},
		{
			name: "unexpected type of error",
			input: func() error {
				return &fakeError{s: status.New(http.StatusNotFound)}
			}(),
			wantStatus: nil,
			wantOK:     false,
		},
		{
			name:       "expected type of error",
			input:      status.New(http.StatusNotFound).Err(),
			wantStatus: status.New(http.StatusNotFound),
			wantOK:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := status.FromError(tt.input)
			require.Equal(t, tt.wantStatus, result)
			require.Equal(t, tt.wantOK, ok)
		})
	}
}

func TestStatus_Code(t *testing.T) {
	s := status.New(http.StatusNotFound)
	require.Equal(t, http.StatusNotFound, s.Code())
}

func TestStatus_Message(t *testing.T) {
	s := status.New(http.StatusNotFound)
	require.Equal(t, "Not Found", s.Message())
}

func TestStatus_WithDetails(t *testing.T) {
	s := status.New(http.StatusNotFound).WithDetails(errors.New("some error"))
	require.Equal(t, errors.New("some error"), s.Details())
}

func TestStatus_String(t *testing.T) {
	s := status.New(http.StatusInternalServerError)
	expected := fmt.Sprintf("http error: code = %d desc = %s", 500, "Internal Server Error")
	require.Equal(t, expected, s.String())
}

func TestStatus_Err(t *testing.T) {
	var e *status.Error
	s := status.New(http.StatusForbidden)
	require.Error(t, s.Err())
	require.ErrorAs(t, s.Err(), &e)
	require.Equal(t, status.New(http.StatusForbidden), e.HTTPStatus())
}
