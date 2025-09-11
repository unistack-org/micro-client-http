package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	jsoncodec "go.unistack.org/micro-codec-json/v4"
	"google.golang.org/protobuf/proto"

	pb "go.unistack.org/micro-client-http/v4/builder/proto"
)

func TestJoinURL(t *testing.T) {
	tests := []struct {
		name string
		addr string
		path string
		want string
	}{
		{
			name: "both without slash",
			addr: "http://example.com",
			path: "api/v1",
			want: "http://example.com/api/v1",
		},
		{
			name: "addr with slash, path without slash",
			addr: "http://example.com/",
			path: "api/v1",
			want: "http://example.com/api/v1",
		},
		{
			name: "addr without slash, path with slash",
			addr: "http://example.com",
			path: "/api/v1",
			want: "http://example.com/api/v1",
		},
		{
			name: "both with slash",
			addr: "http://example.com/",
			path: "/api/v1",
			want: "http://example.com/api/v1",
		},
		{
			name: "empty addr",
			addr: "",
			path: "/api/v1",
			want: "/api/v1",
		},
		{
			name: "empty path",
			addr: "http://example.com",
			path: "",
			want: "http://example.com",
		},
		{
			name: "both empty",
			addr: "",
			path: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, joinURL(tt.addr, tt.path))
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "host with port",
			input:   "localhost:8080",
			want:    "http://localhost:8080",
			wantErr: false,
		},
		{
			name:    "http with host",
			input:   "http://example.com",
			want:    "http://example.com",
			wantErr: false,
		},
		{
			name:    "http with no host",
			input:   "http://",
			want:    "",
			wantErr: true,
		},
		{
			name:    "https with host",
			input:   "https://example.com",
			want:    "https://example.com",
			wantErr: false,
		},
		{
			name:    "https with no host",
			input:   "https://",
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid scheme",
			input:   "ftp://example.com",
			want:    "",
			wantErr: true,
		},
		{
			name:    "IPv4 without scheme",
			input:   "127.0.0.1:9000",
			want:    "http://127.0.0.1:9000",
			wantErr: false,
		},
		{
			name:    "IPv4 with scheme",
			input:   "http://127.0.0.1:8080",
			want:    "http://127.0.0.1:8080",
			wantErr: false,
		},
		{
			name:    "IPv6 without scheme",
			input:   "[::1]:8080",
			want:    "http://[::1]:8080",
			wantErr: false,
		},
		{
			name:    "IPv6 with scheme",
			input:   "https://[::1]:443",
			want:    "https://[::1]:443",
			wantErr: false,
		},
		{
			name:    "hostname only",
			input:   "my-service",
			want:    "http://my-service",
			wantErr: false,
		},
		{
			name:    "hostname with path",
			input:   "service.local/api/v1",
			want:    "http://service.local/api/v1",
			wantErr: false,
		},
		{
			name:    "hostname with dash and port",
			input:   "api-service.local:8080",
			want:    "http://api-service.local:8080",
			wantErr: false,
		},
		{
			name:    "just path",
			input:   "/api/v1",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "http with query params",
			input:   "http://example.com?x=1&y=2",
			want:    "http://example.com?x=1&y=2",
			wantErr: false,
		},
		{
			name:    "http with fragment",
			input:   "http://example.com/path#section1",
			want:    "http://example.com/path#section1",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeURL(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, result.String())
			}
		})
	}
}

func TestMarshallMsg(t *testing.T) {
	type request = pb.Test_Client_Call_Request

	tests := []struct {
		name     string
		msg      proto.Message
		expected string
	}{
		{
			name:     "empty",
			msg:      &request{},
			expected: "",
		},
		{
			name:     "nil",
			msg:      nil,
			expected: "",
		},
		{
			name:     "valid",
			msg:      &request{UserId: "123", OrderId: 456},
			expected: `{"userId":"123","orderId":456}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := marshallMsg(jsoncodec.NewCodec(), tt.msg)
			require.NoError(t, err)
			require.Equal(t, tt.expected, string(result))
		})
	}
}

func TestApplyCookies(t *testing.T) {
	tests := []struct {
		name       string
		rawCookies []string
		want       []*http.Cookie
	}{
		{
			name:       "empty",
			rawCookies: []string{},
			want:       []*http.Cookie{},
		},
		{
			name:       "single cookie",
			rawCookies: []string{"session=abc123"},
			want: []*http.Cookie{
				{Name: "session", Value: "abc123"},
			},
		},
		{
			name:       "multiple cookies separate items",
			rawCookies: []string{"session=abc123", "user=john"},
			want: []*http.Cookie{
				{Name: "session", Value: "abc123"},
				{Name: "user", Value: "john"},
			},
		},
		{
			name:       "multiple cookies in one item",
			rawCookies: []string{"a=1; b=2"},
			want: []*http.Cookie{
				{Name: "a", Value: "1"},
				{Name: "b", Value: "2"},
			},
		},
		{
			name:       "mix of combined and separate cookies",
			rawCookies: []string{"a=1; b=2", "c=3"},
			want: []*http.Cookie{
				{Name: "a", Value: "1"},
				{Name: "b", Value: "2"},
				{Name: "c", Value: "3"},
			},
		},
		{
			name:       "duplicate cookies",
			rawCookies: []string{"session=abc123", "session=xyz"},
			want: []*http.Cookie{
				{Name: "session", Value: "abc123"},
				{Name: "session", Value: "xyz"},
			},
		},
		{
			name:       "cookie with spaces",
			rawCookies: []string{"token=abc 123"},
			want: []*http.Cookie{
				{Name: "token", Value: "abc 123", Quoted: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			applyCookies(req, tt.rawCookies)
			require.Equal(t, tt.want, req.Cookies())
		})
	}
}

func TestValidateHeadersAndCookies(t *testing.T) {
	tests := []struct {
		name           string
		prepareRequest func() *http.Request
		parameters     map[string]map[string]string
		wantErr        bool
	}{
		{
			name: "all required headers and cookies present",
			prepareRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("My-Header-1", "Header-Value-1")
				req.Header.Set("My-Header-2", "Header-Value-2")
				req.AddCookie(&http.Cookie{Name: "session-1", Value: "abc-1"})
				req.AddCookie(&http.Cookie{Name: "session-2", Value: "abc-2"})
				return req
			},
			parameters: map[string]map[string]string{
				"header": {"My-Header-1": "true", "My-Header-2": "true"},
				"cookie": {"session-1": "true", "session-2": "true"},
			},
			wantErr: false,
		},
		{
			name: "missing required header",
			prepareRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("My-Header-1", "Header-Value-1")
				req.AddCookie(&http.Cookie{Name: "session-1", Value: "abc-1"})
				req.AddCookie(&http.Cookie{Name: "session-2", Value: "abc-2"})
				return req
			},
			parameters: map[string]map[string]string{
				"header": {"My-Header-1": "true", "My-Header-2": "true"},
				"cookie": {"session-1": "true", "session-2": "true"},
			},
			wantErr: true,
		},
		{
			name: "missing required cookie",
			prepareRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("My-Header-1", "Header-Value-1")
				req.Header.Set("My-Header-2", "Header-Value-2")
				req.AddCookie(&http.Cookie{Name: "session-1", Value: "abc-1"})
				return req
			},
			parameters: map[string]map[string]string{
				"header": {"My-Header-1": "true", "My-Header-2": "true"},
				"cookie": {"session-1": "true", "session-2": "true"},
			},
			wantErr: true,
		},
		{
			name: "optional header and cookie not provided partially",
			prepareRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("My-Header-1", "Header-Value-1")
				req.AddCookie(&http.Cookie{Name: "session-1", Value: "abc-1"})
				return req
			},
			parameters: map[string]map[string]string{
				"header": {"My-Header-1": "true", "My-Header-2": "false"},
				"cookie": {"session-1": "true", "session-2": "false"},
			},
			wantErr: false,
		},
		{
			name: "optional header and cookie not provided",
			prepareRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				return req
			},
			parameters: map[string]map[string]string{
				"header": {"My-Header-1": "false", "My-Header-2": "false"},
				"cookie": {"session-1": "false", "session-2": "false"},
			},
			wantErr: false,
		},
		{
			name: "no headers or cookies required",
			prepareRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				return req
			},
			parameters: map[string]map[string]string{},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHeadersAndCookies(tt.prepareRequest(), tt.parameters)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
