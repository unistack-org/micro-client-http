package builder

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBodyOption_String(t *testing.T) {
	tests := []struct {
		name string
		opt  bodyOption
		want string
	}{
		{"empty", bodyOption(""), ""},
		{"star", bodyOption("*"), "*"},
		{"field", bodyOption("field"), "field"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.opt.String())
		})
	}
}

func TestBodyOption_isFullBody(t *testing.T) {
	tests := []struct {
		name string
		opt  bodyOption
		want bool
	}{
		{"empty", bodyOption(""), false},
		{"star", bodyOption("*"), true},
		{"field", bodyOption("field"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.opt.isFullBody())
		})
	}
}

func TestBodyOption_isWithoutBody(t *testing.T) {
	tests := []struct {
		name string
		opt  bodyOption
		want bool
	}{
		{"empty", bodyOption(""), true},
		{"star", bodyOption("*"), false},
		{"field", bodyOption("field"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.opt.isWithoutBody())
		})
	}
}

func TestBodyOption_isSingleField(t *testing.T) {
	tests := []struct {
		name string
		opt  bodyOption
		want bool
	}{
		{"empty", bodyOption(""), false},
		{"star", bodyOption("*"), false},
		{"field", bodyOption("field"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.opt.isSingleField())
		})
	}
}
