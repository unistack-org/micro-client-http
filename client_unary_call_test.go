package http_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	jsoncodec "go.unistack.org/micro-codec-json/v4"
	"go.unistack.org/micro/v4/client"
	microerr "go.unistack.org/micro/v4/errors"
	"go.unistack.org/micro/v4/metadata"
	"google.golang.org/protobuf/proto"

	httpcli "go.unistack.org/micro-client-http/v4"
	pb "go.unistack.org/micro-client-http/v4/builder/proto"
	"go.unistack.org/micro-client-http/v4/status"
)

func TestClient_Call_Get(t *testing.T) {
	type (
		request  = pb.Test_Client_Call_Request
		response = pb.Test_Client_Call_Response
	)

	tests := []struct {
		name        string
		method      string
		path        string
		req         *request
		options     []client.CallOption
		wantPath    string
		wantReqBody []byte
		wantRsp     *response
		wantErr     bool
	}{
		{
			name:        "GET request (query)",
			method:      http.MethodGet,
			path:        "/user/products",
			req:         &request{UserId: "123", OrderId: 456},
			wantPath:    "/user/products?order_id=456&user_id=123",
			wantReqBody: []byte{},
			wantRsp:     &response{Id: "product-id", Name: "product-name"},
			wantErr:     false,
		},
		{
			name:        "GET request (path)",
			method:      http.MethodGet,
			path:        "/user/{user_id}/order/{order_id}/products",
			req:         &request{UserId: "123", OrderId: 456},
			wantPath:    "/user/123/order/456/products",
			wantReqBody: []byte{},
			wantRsp:     &response{Id: "product-id", Name: "product-name"},
			wantErr:     false,
		},
		{
			name:        "GET request (path + query)",
			method:      http.MethodGet,
			path:        "/user/{user_id}/products",
			req:         &request{UserId: "123", OrderId: 456},
			wantPath:    "/user/123/products?order_id=456",
			wantReqBody: []byte{},
			wantRsp:     &response{Id: "product-id", Name: "product-name"},
			wantErr:     false,
		},
		{
			name:        "GET request (zero-value query)",
			method:      http.MethodGet,
			path:        "/user/products",
			req:         &request{},
			wantPath:    "/user/products",
			wantReqBody: []byte{},
			wantRsp:     &response{Id: "product-id", Name: "product-name"},
			wantErr:     false,
		},
		{
			name:    "GET request (zero-value path)",
			method:  http.MethodGet,
			path:    "/user/{user_id}/products",
			req:     &request{OrderId: 456},
			wantRsp: nil,
			wantErr: true,
		},
		{
			name:    "GET request (with body)",
			method:  http.MethodGet,
			path:    "/user/products",
			req:     &request{UserId: "123", OrderId: 456},
			options: []client.CallOption{httpcli.Body("*")},
			wantRsp: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, tt.method, r.Method)
				require.Equal(t, tt.wantPath, r.URL.RequestURI())

				require.Equal(t, "application/json", r.Header.Get("Content-Type"))
				require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
				require.Equal(t, "My-Header-Value", r.Header.Get("My-Header"))

				buf, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				defer r.Body.Close()
				require.Equal(t, tt.wantReqBody, buf)

				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("My-Header", "My-Header-Value")
				w.WriteHeader(http.StatusOK)

				if tt.wantRsp != nil {
					c := jsoncodec.NewCodec()
					buf, err = c.Marshal(tt.wantRsp)
					require.NoError(t, err)
					_, err = w.Write(buf)
					require.NoError(t, err)
				}
			}))
			defer server.Close()

			httpClient := httpcli.NewClient(
				client.Codec("application/json", jsoncodec.NewCodec()),
			)

			var (
				ctx = metadata.NewOutgoingContext(
					context.Background(),
					metadata.Pairs("Authorization", "Bearer token", "My-Header", "My-Header-Value"),
				)
				rsp          = &response{}
				respMetadata = metadata.Metadata{}
			)

			opts := []client.CallOption{
				client.WithAddress(server.URL),
				client.WithResponseMetadata(&respMetadata),
				httpcli.Method(tt.method),
				httpcli.Path(tt.path),
			}

			if len(tt.options) > 0 {
				opts = append(opts, tt.options...)
			}

			err := httpClient.Call(
				ctx,
				httpClient.NewRequest("test.service", "Test.Call", tt.req),
				rsp,
				opts...,
			)

			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, rsp)
			} else {
				require.NoError(t, err)
				require.True(t, proto.Equal(tt.wantRsp, rsp))
				require.Equal(t, "application/json", respMetadata.GetJoined("Content-Type"))
				require.Equal(t, "My-Header-Value", respMetadata.GetJoined("My-Header"))
			}
		})
	}
}

func TestClient_Call_Head(t *testing.T) {
	type (
		request  = pb.Test_Client_Call_Request
		response = pb.Test_Client_Call_Response
	)

	tests := []struct {
		name        string
		method      string
		path        string
		req         *request
		options     []client.CallOption
		wantPath    string
		wantReqBody []byte
		wantRsp     *response
		wantErr     bool
	}{
		{
			name:        "HEAD request (query)",
			method:      http.MethodHead,
			path:        "/user/products",
			req:         &request{UserId: "123", OrderId: 456},
			wantPath:    "/user/products?order_id=456&user_id=123",
			wantReqBody: []byte{},
			wantRsp:     &response{},
			wantErr:     false,
		},
		{
			name:        "HEAD request (path)",
			method:      http.MethodHead,
			path:        "/user/{user_id}/order/{order_id}/products",
			req:         &request{UserId: "123", OrderId: 456},
			wantPath:    "/user/123/order/456/products",
			wantReqBody: []byte{},
			wantRsp:     &response{},
			wantErr:     false,
		},
		{
			name:        "HEAD request (path + query)",
			method:      http.MethodHead,
			path:        "/user/{user_id}/products",
			req:         &request{UserId: "123", OrderId: 456},
			wantPath:    "/user/123/products?order_id=456",
			wantReqBody: []byte{},
			wantRsp:     &response{},
			wantErr:     false,
		},
		{
			name:        "HEAD request (zero-value query)",
			method:      http.MethodHead,
			path:        "/user/products",
			req:         &request{},
			wantPath:    "/user/products",
			wantReqBody: []byte{},
			wantRsp:     &response{},
			wantErr:     false,
		},
		{
			name:    "HEAD request (zero-value path)",
			method:  http.MethodHead,
			path:    "/user/{user_id}/products",
			req:     &request{OrderId: 456},
			wantRsp: nil,
			wantErr: true,
		},
		{
			name:    "HEAD request (with body)",
			method:  http.MethodHead,
			path:    "/user/products",
			req:     &request{UserId: "123", OrderId: 456},
			options: []client.CallOption{httpcli.Body("*")},
			wantRsp: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, tt.method, r.Method)
				require.Equal(t, tt.wantPath, r.URL.RequestURI())

				require.Equal(t, "application/json", r.Header.Get("Content-Type"))
				require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
				require.Equal(t, "My-Header-Value", r.Header.Get("My-Header"))

				buf, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				defer r.Body.Close()
				require.Equal(t, tt.wantReqBody, buf)

				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("My-Header", "My-Header-Value")
				w.WriteHeader(http.StatusOK)

				// used to verify that the HTTP client skips the response body for HEAD method
				c := jsoncodec.NewCodec()
				buf, err = c.Marshal(map[string]any{"id": "product-id", "name": "product-name"})
				require.NoError(t, err)
				_, err = w.Write(buf)
				require.NoError(t, err)
			}))
			defer server.Close()

			httpClient := httpcli.NewClient(
				client.Codec("application/json", jsoncodec.NewCodec()),
			)

			var (
				ctx = metadata.NewOutgoingContext(
					context.Background(),
					metadata.Pairs("Authorization", "Bearer token", "My-Header", "My-Header-Value"),
				)
				rsp          = &response{}
				respMetadata = metadata.Metadata{}
			)

			opts := []client.CallOption{
				client.WithAddress(server.URL),
				client.WithResponseMetadata(&respMetadata),
				httpcli.Method(tt.method),
				httpcli.Path(tt.path),
			}

			if len(tt.options) > 0 {
				opts = append(opts, tt.options...)
			}

			err := httpClient.Call(
				ctx,
				httpClient.NewRequest("test.service", "Test.Call", tt.req),
				rsp,
				opts...,
			)

			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, rsp)
			} else {
				require.NoError(t, err)
				require.True(t, proto.Equal(tt.wantRsp, rsp))
				require.Equal(t, "application/json", respMetadata.GetJoined("Content-Type"))
				require.Equal(t, "My-Header-Value", respMetadata.GetJoined("My-Header"))
			}
		})
	}
}

func TestClient_Call_Post(t *testing.T) {
	type (
		request  = pb.Test_Client_Call_Request
		response = pb.Test_Client_Call_Response
	)

	tests := []struct {
		name        string
		method      string
		path        string
		req         *request
		options     []client.CallOption
		wantPath    string
		wantReqBody []byte
		wantRsp     *response
		wantErr     bool
	}{
		{
			name:        "POST request (query)",
			method:      http.MethodPost,
			path:        "/user/products",
			req:         &request{UserId: "123", OrderId: 456},
			wantPath:    "/user/products?order_id=456&user_id=123",
			wantReqBody: []byte{},
			wantRsp:     &response{Id: "product-id", Name: "product-name"},
			wantErr:     false,
		},
		{
			name:        "POST request (path)",
			method:      http.MethodPost,
			path:        "/user/{user_id}/order/{order_id}/products",
			req:         &request{UserId: "123", OrderId: 456},
			wantPath:    "/user/123/order/456/products",
			wantReqBody: []byte{},
			wantRsp:     &response{Id: "product-id", Name: "product-name"},
			wantErr:     false,
		},
		{
			name:        "POST request (body)",
			method:      http.MethodPost,
			path:        "/user/products",
			req:         &request{UserId: "123", OrderId: 456},
			options:     []client.CallOption{httpcli.Body("*")},
			wantPath:    "/user/products",
			wantReqBody: []byte(`{"userId":"123","orderId":456}`),
			wantRsp:     &response{Id: "product-id", Name: "product-name"},
			wantErr:     false,
		},
		{
			name:        "POST request (path + query)",
			method:      http.MethodPost,
			path:        "/user/{user_id}/products",
			req:         &request{UserId: "123", OrderId: 456},
			wantPath:    "/user/123/products?order_id=456",
			wantReqBody: []byte{},
			wantRsp:     &response{Id: "product-id", Name: "product-name"},
			wantErr:     false,
		},
		{
			name:        "POST request (path + body)",
			method:      http.MethodPost,
			path:        "/user/{user_id}/products",
			req:         &request{UserId: "123", OrderId: 456},
			options:     []client.CallOption{httpcli.Body("*")},
			wantPath:    "/user/123/products",
			wantReqBody: []byte(`{"orderId":456}`),
			wantRsp:     &response{Id: "product-id", Name: "product-name"},
			wantErr:     false,
		},
		{
			name:        "POST request (query + body)",
			method:      http.MethodPost,
			path:        "/user/products",
			req:         &request{UserId: "123", OrderId: 456},
			options:     []client.CallOption{httpcli.Body("order_id")},
			wantPath:    "/user/products?user_id=123",
			wantReqBody: []byte(`{"orderId":456}`),
			wantRsp:     &response{Id: "product-id", Name: "product-name"},
			wantErr:     false,
		},
		{
			name:        "POST request (zero-value query)",
			method:      http.MethodPost,
			path:        "/user/products",
			req:         &request{},
			wantPath:    "/user/products",
			wantReqBody: []byte{},
			wantRsp:     &response{Id: "product-id", Name: "product-name"},
			wantErr:     false,
		},
		{
			name:        "POST request (zero-value body)",
			method:      http.MethodPost,
			path:        "/user/products",
			req:         &request{},
			options:     []client.CallOption{httpcli.Body("*")},
			wantPath:    "/user/products",
			wantReqBody: []byte{},
			wantRsp:     &response{Id: "product-id", Name: "product-name"},
			wantErr:     false,
		},
		{
			name:    "POST request (zero-value path)",
			method:  http.MethodPost,
			path:    "/user/{user_id}/products",
			req:     &request{OrderId: 456},
			wantRsp: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, tt.method, r.Method)
				require.Equal(t, tt.wantPath, r.URL.RequestURI())

				require.Equal(t, "application/json", r.Header.Get("Content-Type"))
				require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
				require.Equal(t, "My-Header-Value", r.Header.Get("My-Header"))

				buf, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				defer r.Body.Close()
				require.Equal(t, tt.wantReqBody, buf)

				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("My-Header", "My-Header-Value")
				w.WriteHeader(http.StatusOK)

				if tt.wantRsp != nil {
					c := jsoncodec.NewCodec()
					buf, err = c.Marshal(tt.wantRsp)
					require.NoError(t, err)
					_, err = w.Write(buf)
					require.NoError(t, err)
				}
			}))
			defer server.Close()

			httpClient := httpcli.NewClient(
				client.Codec("application/json", jsoncodec.NewCodec()),
			)

			var (
				ctx = metadata.NewOutgoingContext(
					context.Background(),
					metadata.Pairs("Authorization", "Bearer token", "My-Header", "My-Header-Value"),
				)
				rsp          = &response{}
				respMetadata = metadata.Metadata{}
			)

			opts := []client.CallOption{
				client.WithAddress(server.URL),
				client.WithResponseMetadata(&respMetadata),
				httpcli.Method(tt.method),
				httpcli.Path(tt.path),
			}

			if len(tt.options) > 0 {
				opts = append(opts, tt.options...)
			}

			err := httpClient.Call(
				ctx,
				httpClient.NewRequest("test.service", "Test.Call", tt.req),
				rsp,
				opts...,
			)

			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, rsp)
			} else {
				require.NoError(t, err)
				require.True(t, proto.Equal(tt.wantRsp, rsp))
				require.Equal(t, "application/json", respMetadata.GetJoined("Content-Type"))
				require.Equal(t, "My-Header-Value", respMetadata.GetJoined("My-Header"))
			}
		})
	}
}

func TestClient_Call_Delete(t *testing.T) {
	type (
		request  = pb.Test_Client_Call_Request
		response = pb.Test_Client_Call_Response
	)

	tests := []struct {
		name        string
		method      string
		path        string
		req         *request
		options     []client.CallOption
		wantPath    string
		wantReqBody []byte
		wantRsp     *response
		wantErr     bool
	}{
		{
			name:        "DELETE request (query)",
			method:      http.MethodDelete,
			path:        "/user/products",
			req:         &request{UserId: "123", OrderId: 456},
			wantPath:    "/user/products?order_id=456&user_id=123",
			wantReqBody: []byte{},
			wantRsp:     &response{Id: "product-id", Name: "product-name"},
			wantErr:     false,
		},
		{
			name:        "DELETE request (path)",
			method:      http.MethodDelete,
			path:        "/user/{user_id}/order/{order_id}/products",
			req:         &request{UserId: "123", OrderId: 456},
			wantPath:    "/user/123/order/456/products",
			wantReqBody: []byte{},
			wantRsp:     &response{Id: "product-id", Name: "product-name"},
			wantErr:     false,
		},
		{
			name:        "DELETE request (path + query)",
			method:      http.MethodDelete,
			path:        "/user/{user_id}/products",
			req:         &request{UserId: "123", OrderId: 456},
			wantPath:    "/user/123/products?order_id=456",
			wantReqBody: []byte{},
			wantRsp:     &response{Id: "product-id", Name: "product-name"},
			wantErr:     false,
		},
		{
			name:        "DELETE request (zero-value query)",
			method:      http.MethodDelete,
			path:        "/user/products",
			req:         &request{},
			wantPath:    "/user/products",
			wantReqBody: []byte{},
			wantRsp:     &response{Id: "product-id", Name: "product-name"},
			wantErr:     false,
		},
		{
			name:    "DELETE request (zero-value path)",
			method:  http.MethodDelete,
			path:    "/user/{user_id}/products",
			req:     &request{OrderId: 456},
			wantRsp: nil,
			wantErr: true,
		},
		{
			name:    "DELETE request (with body)",
			method:  http.MethodDelete,
			path:    "/user/products",
			req:     &request{UserId: "123", OrderId: 456},
			options: []client.CallOption{httpcli.Body("*")},
			wantRsp: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, tt.method, r.Method)
				require.Equal(t, tt.wantPath, r.URL.RequestURI())

				require.Equal(t, "application/json", r.Header.Get("Content-Type"))
				require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
				require.Equal(t, "My-Header-Value", r.Header.Get("My-Header"))

				buf, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				defer r.Body.Close()
				require.Equal(t, tt.wantReqBody, buf)

				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("My-Header", "My-Header-Value")
				w.WriteHeader(http.StatusOK)

				if tt.wantRsp != nil {
					c := jsoncodec.NewCodec()
					buf, err = c.Marshal(tt.wantRsp)
					require.NoError(t, err)
					_, err = w.Write(buf)
					require.NoError(t, err)
				}
			}))
			defer server.Close()

			httpClient := httpcli.NewClient(
				client.Codec("application/json", jsoncodec.NewCodec()),
			)

			var (
				ctx = metadata.NewOutgoingContext(
					context.Background(),
					metadata.Pairs("Authorization", "Bearer token", "My-Header", "My-Header-Value"),
				)
				rsp          = &response{}
				respMetadata = metadata.Metadata{}
			)

			opts := []client.CallOption{
				client.WithAddress(server.URL),
				client.WithResponseMetadata(&respMetadata),
				httpcli.Method(tt.method),
				httpcli.Path(tt.path),
			}

			if len(tt.options) > 0 {
				opts = append(opts, tt.options...)
			}

			err := httpClient.Call(
				ctx,
				httpClient.NewRequest("test.service", "Test.Call", tt.req),
				rsp,
				opts...,
			)

			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, rsp)
			} else {
				require.NoError(t, err)
				require.True(t, proto.Equal(tt.wantRsp, rsp))
				require.Equal(t, "application/json", respMetadata.GetJoined("Content-Type"))
				require.Equal(t, "My-Header-Value", respMetadata.GetJoined("My-Header"))
			}
		})
	}
}

func TestClient_Call_APIError_WithErrorsMap(t *testing.T) {
	type (
		request      = pb.Test_Client_Call_Request
		response     = pb.Test_Client_Call_Response
		defaultError = pb.Test_Client_Call_DefaultError
		specialError = pb.Test_Client_Call_SpecialError
	)

	tests := []struct {
		name           string
		serverMock     func() *httptest.Server
		expectedStatus *status.Status
	}{
		{
			name: "default error",
			serverMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Validate request
					require.Equal(t, "POST", r.Method)
					require.Equal(t, "/user/products", r.URL.RequestURI())

					require.Equal(t, "application/json", r.Header.Get("Content-Type"))
					require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
					require.Equal(t, "My-Header-Value", r.Header.Get("My-Header"))

					buf, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					defer r.Body.Close()

					c := jsoncodec.NewCodec()

					req := &request{}
					err = c.Unmarshal(buf, req)
					require.NoError(t, err)
					require.True(t, proto.Equal(&request{UserId: "123", OrderId: 456}, req))

					// Return response
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("My-Header", "My-Header-Value")
					w.WriteHeader(http.StatusBadRequest)

					resp := map[string]interface{}{
						"code": "default-error-code",
						"msg":  "default-error-message",
					}
					buf, err = c.Marshal(resp)
					require.NoError(t, err)
					_, err = w.Write(buf)
					require.NoError(t, err)
				}))
			},
			expectedStatus: status.New(http.StatusBadRequest).WithDetails(
				&defaultError{
					Code: "default-error-code",
					Msg:  "default-error-message",
				},
			),
		},
		{
			name: "special error",
			serverMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Validate request
					require.Equal(t, "POST", r.Method)
					require.Equal(t, "/user/products", r.URL.RequestURI())

					require.Equal(t, "application/json", r.Header.Get("Content-Type"))
					require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
					require.Equal(t, "My-Header-Value", r.Header.Get("My-Header"))

					buf, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					defer r.Body.Close()

					c := jsoncodec.NewCodec()

					req := &request{}
					err = c.Unmarshal(buf, req)
					require.NoError(t, err)
					require.True(t, proto.Equal(&request{UserId: "123", OrderId: 456}, req))

					// Return response
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("My-Header", "My-Header-Value")
					w.WriteHeader(http.StatusForbidden)

					resp := map[string]interface{}{
						"code":    "special-error-code",
						"msg":     "special-error-message",
						"warning": "special-error-warning",
					}
					buf, err = c.Marshal(resp)
					require.NoError(t, err)
					_, err = w.Write(buf)
					require.NoError(t, err)
				}))
			},
			expectedStatus: status.New(http.StatusForbidden).WithDetails(
				&specialError{
					Code:    "special-error-code",
					Msg:     "special-error-message",
					Warning: "special-error-warning",
				},
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.serverMock()
			defer server.Close()

			httpClient := httpcli.NewClient(
				client.Codec("application/json", jsoncodec.NewCodec()),
			)

			var (
				ctx = metadata.NewOutgoingContext(
					context.Background(),
					metadata.Pairs("Authorization", "Bearer token", "My-Header", "My-Header-Value"),
				)
				req = &request{UserId: "123", OrderId: 456}
				rsp = &response{}

				respMetadata = metadata.Metadata{}
			)

			opts := []client.CallOption{
				client.WithAddress(server.URL),
				client.WithResponseMetadata(&respMetadata),
				httpcli.Method(http.MethodPost),
				httpcli.Path("/user/products"),
				httpcli.Body("*"),
				httpcli.ErrorMap(map[string]any{
					"default": &defaultError{},
					"403":     &specialError{},
				}),
			}

			err := httpClient.Call(
				ctx,
				httpClient.NewRequest("test.service", "Test.Call", req),
				rsp,
				opts...,
			)

			require.Error(t, err)
			s, ok := status.FromError(err)
			require.True(t, ok)
			require.NotNil(t, s)
			require.Equal(t, tt.expectedStatus, s)
			require.Empty(t, rsp)

			require.Equal(t, "application/json", respMetadata.GetJoined("Content-Type"))
			require.Equal(t, "My-Header-Value", respMetadata.GetJoined("My-Header"))
		})
	}
}

func TestClient_Call_APIError_WithoutErrorsMap(t *testing.T) {
	type (
		request  = pb.Test_Client_Call_Request
		response = pb.Test_Client_Call_Response
	)

	serverMock := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Validate request
			require.Equal(t, "POST", r.Method)
			require.Equal(t, "/user/products", r.URL.RequestURI())

			require.Equal(t, "application/json", r.Header.Get("Content-Type"))
			require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
			require.Equal(t, "My-Header-Value", r.Header.Get("My-Header"))

			buf, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			defer r.Body.Close()

			c := jsoncodec.NewCodec()

			req := &request{}
			err = c.Unmarshal(buf, req)
			require.NoError(t, err)
			require.True(t, proto.Equal(&request{UserId: "123", OrderId: 456}, req))

			// Return response
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("My-Header", "My-Header-Value")
			w.WriteHeader(http.StatusConflict)

			resp := map[string]interface{}{"message": "not-mapped-error"}
			buf, err = c.Marshal(resp)
			require.NoError(t, err)
			_, err = w.Write(buf)
			require.NoError(t, err)
		}))
	}
	expectedStatus := status.New(http.StatusConflict)

	server := serverMock()
	defer server.Close()

	httpClient := httpcli.NewClient(
		client.Codec("application/json", jsoncodec.NewCodec()),
	)

	var (
		ctx = metadata.NewOutgoingContext(
			context.Background(),
			metadata.Pairs("Authorization", "Bearer token", "My-Header", "My-Header-Value"),
		)
		req = &request{UserId: "123", OrderId: 456}
		rsp = &response{}

		respMetadata = metadata.Metadata{}
	)

	opts := []client.CallOption{
		client.WithAddress(server.URL),
		client.WithResponseMetadata(&respMetadata),
		httpcli.Method(http.MethodPost),
		httpcli.Path("/user/products"),
		httpcli.Body("*"),
	}

	err := httpClient.Call(
		ctx,
		httpClient.NewRequest("test.service", "Test.Call", req),
		rsp,
		opts...,
	)

	require.Error(t, err)
	s, ok := status.FromError(err)
	require.True(t, ok)
	require.NotNil(t, s)
	require.Equal(t, expectedStatus, s)
	require.Empty(t, rsp)

	require.Equal(t, "application/json", respMetadata.GetJoined("Content-Type"))
	require.Equal(t, "My-Header-Value", respMetadata.GetJoined("My-Header"))
}

func TestClient_Call_HeadersAndCookies(t *testing.T) {
	type (
		request  = pb.Test_Client_Call_Request
		response = pb.Test_Client_Call_Response
	)

	tests := []struct {
		name            string
		serverMock      func() *httptest.Server
		prepareMetadata func() metadata.Metadata
		headerOption    client.CallOption
		cookieOption    client.CallOption
		expectedRsp     *response
		wantErr         bool
	}{
		{
			name: "with required headers",
			serverMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Validate request
					require.Equal(t, "POST", r.Method)
					require.Equal(t, "/user/products", r.URL.RequestURI())

					require.Equal(t, "application/json", r.Header.Get("Content-Type"))
					require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
					require.Equal(t, "My-Header-Value", r.Header.Get("My-Header"))

					buf, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					defer r.Body.Close()

					c := jsoncodec.NewCodec()

					req := &request{}
					err = c.Unmarshal(buf, req)
					require.NoError(t, err)
					require.True(t, proto.Equal(&request{UserId: "123", OrderId: 456}, req))

					// Return response
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("My-Header", "My-Header-Value")
					w.WriteHeader(http.StatusOK)

					resp := map[string]interface{}{
						"id":   "product-id",
						"name": "product-name",
					}
					buf, err = c.Marshal(resp)
					require.NoError(t, err)
					_, err = w.Write(buf)
					require.NoError(t, err)
				}))
			},
			prepareMetadata: func() metadata.Metadata {
				return metadata.Pairs("Authorization", "Bearer token", "My-Header", "My-Header-Value")
			},
			headerOption: httpcli.Header("Authorization", "true", "My-Header", "true"),
			expectedRsp:  &response{Id: "product-id", Name: "product-name"},
		},
		{
			name: "without required headers",
			serverMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Validate request
					require.Equal(t, "POST", r.Method)
					require.Equal(t, "/user/products", r.URL.RequestURI())

					require.Equal(t, "application/json", r.Header.Get("Content-Type"))
					require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
					require.Equal(t, "My-Header-Value", r.Header.Get("My-Header"))

					buf, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					defer r.Body.Close()

					c := jsoncodec.NewCodec()

					req := &request{}
					err = c.Unmarshal(buf, req)
					require.NoError(t, err)
					require.True(t, proto.Equal(&request{UserId: "123", OrderId: 456}, req))

					// Return response
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("My-Header", "My-Header-Value")
					w.WriteHeader(http.StatusOK)

					resp := map[string]interface{}{
						"id":   "product-id",
						"name": "product-name",
					}
					buf, err = c.Marshal(resp)
					require.NoError(t, err)
					_, err = w.Write(buf)
					require.NoError(t, err)
				}))
			},
			prepareMetadata: func() metadata.Metadata {
				return metadata.Pairs("Authorization", "Bearer token")
			},
			headerOption: httpcli.Header("Authorization", "true", "My-Header", "true"),
			wantErr:      true,
		},
		{
			name: "with required cookies",
			serverMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Validate request
					require.Equal(t, "POST", r.Method)
					require.Equal(t, "/user/products", r.URL.RequestURI())

					require.Equal(t, "application/json", r.Header.Get("Content-Type"))
					require.Equal(t, "session_id=abc123; theme=dark", r.Header.Get("Cookie"))

					buf, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					defer r.Body.Close()

					c := jsoncodec.NewCodec()

					req := &request{}
					err = c.Unmarshal(buf, req)
					require.NoError(t, err)
					require.True(t, proto.Equal(&request{UserId: "123", OrderId: 456}, req))

					// Return response
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("My-Header", "My-Header-Value")
					w.WriteHeader(http.StatusOK)

					resp := map[string]interface{}{
						"id":   "product-id",
						"name": "product-name",
					}
					buf, err = c.Marshal(resp)
					require.NoError(t, err)
					_, err = w.Write(buf)
					require.NoError(t, err)
				}))
			},
			prepareMetadata: func() metadata.Metadata {
				return metadata.Pairs("Cookie", "session_id=abc123; theme=dark")
			},
			cookieOption: httpcli.Cookie("session_id", "true", "theme", "true"),
			expectedRsp:  &response{Id: "product-id", Name: "product-name"},
		},
		{
			name: "without required cookies",
			serverMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Validate request
					require.Equal(t, "POST", r.Method)
					require.Equal(t, "/user/products", r.URL.RequestURI())

					require.Equal(t, "application/json", r.Header.Get("Content-Type"))
					require.Equal(t, "session_id=abc123; theme=dark", r.Header.Get("Cookie"))

					buf, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					defer r.Body.Close()

					c := jsoncodec.NewCodec()

					req := &request{}
					err = c.Unmarshal(buf, req)
					require.NoError(t, err)
					require.True(t, proto.Equal(&request{UserId: "123", OrderId: 456}, req))

					// Return response
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("My-Header", "My-Header-Value")
					w.WriteHeader(http.StatusOK)

					resp := map[string]interface{}{
						"id":   "product-id",
						"name": "product-name",
					}
					buf, err = c.Marshal(resp)
					require.NoError(t, err)
					_, err = w.Write(buf)
					require.NoError(t, err)
				}))
			},
			prepareMetadata: func() metadata.Metadata {
				return metadata.Pairs("Cookie", "session_id=abc123")
			},
			cookieOption: httpcli.Cookie("session_id", "true", "theme", "true"),
			wantErr:      true,
		},
		{
			name: "with headers and cookies",
			serverMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Validate request
					require.Equal(t, "POST", r.Method)
					require.Equal(t, "/user/products", r.URL.RequestURI())

					require.Equal(t, "application/json", r.Header.Get("Content-Type"))
					require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
					require.Equal(t, "My-Header-Value", r.Header.Get("My-Header"))
					require.Equal(t, "session_id=abc123; theme=dark", r.Header.Get("Cookie"))

					buf, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					defer r.Body.Close()

					c := jsoncodec.NewCodec()

					req := &request{}
					err = c.Unmarshal(buf, req)
					require.NoError(t, err)
					require.True(t, proto.Equal(&request{UserId: "123", OrderId: 456}, req))

					// Return response
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("My-Header", "My-Header-Value")
					w.WriteHeader(http.StatusOK)

					resp := map[string]interface{}{
						"id":   "product-id",
						"name": "product-name",
					}
					buf, err = c.Marshal(resp)
					require.NoError(t, err)
					_, err = w.Write(buf)
					require.NoError(t, err)
				}))
			},
			prepareMetadata: func() metadata.Metadata {
				return metadata.Pairs(
					"Authorization", "Bearer token",
					"My-Header", "My-Header-Value",
					"Cookie", "session_id=abc123; theme=dark",
				)
			},
			headerOption: httpcli.Header("Authorization", "true", "My-Header", "true"),
			cookieOption: httpcli.Cookie("session_id", "true", "theme", "true"),
			expectedRsp:  &response{Id: "product-id", Name: "product-name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.serverMock()
			defer server.Close()

			httpClient := httpcli.NewClient(
				client.Codec("application/json", jsoncodec.NewCodec()),
			)

			var (
				ctx = metadata.NewOutgoingContext(context.Background(), tt.prepareMetadata())
				req = &request{UserId: "123", OrderId: 456}
				rsp = &response{}

				respMetadata = metadata.Metadata{}
			)

			opts := []client.CallOption{
				client.WithAddress(server.URL),
				client.WithResponseMetadata(&respMetadata),
				httpcli.Method(http.MethodPost),
				httpcli.Path("/user/products"),
				httpcli.Body("*"),
			}
			if tt.headerOption != nil {
				opts = append(opts, tt.headerOption)
			}
			if tt.cookieOption != nil {
				opts = append(opts, tt.cookieOption)
			}

			err := httpClient.Call(
				ctx,
				httpClient.NewRequest("test.service", "Test.Call", req),
				rsp,
				opts...,
			)

			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, rsp)
			} else {
				require.NoError(t, err)
				require.True(t, proto.Equal(tt.expectedRsp, rsp))
				require.Equal(t, "application/json", respMetadata.GetJoined("Content-Type"))
				require.Equal(t, "My-Header-Value", respMetadata.GetJoined("My-Header"))
			}
		})
	}
}

func TestClient_Call_SetCookie(t *testing.T) {
	type (
		request  = pb.Test_Client_Call_Request
		response = pb.Test_Client_Call_Response
	)

	serverMock := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Validate request
			require.Equal(t, "POST", r.Method)
			require.Equal(t, "/user/products", r.URL.RequestURI())

			require.Equal(t, "application/json", r.Header.Get("Content-Type"))
			require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
			require.Equal(t, "My-Header-Value", r.Header.Get("My-Header"))

			buf, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			defer r.Body.Close()

			c := jsoncodec.NewCodec()

			req := &request{}
			err = c.Unmarshal(buf, req)
			require.NoError(t, err)
			require.True(t, proto.Equal(&request{UserId: "123", OrderId: 456}, req))

			// Return response
			cookieN1, err := http.ParseSetCookie("sessionid=abc123; Path=/; HttpOnly")
			require.NoError(t, err)
			http.SetCookie(w, cookieN1)

			cookieN2, err := http.ParseSetCookie("theme=dark; Path=/; Max-Age=3600")
			require.NoError(t, err)
			http.SetCookie(w, cookieN2)

			cookieN3, err := http.ParseSetCookie("lang=en-US; Path=/")
			require.NoError(t, err)
			http.SetCookie(w, cookieN3)

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("My-Header", "My-Header-Value")
			w.WriteHeader(http.StatusNoContent)
		}))
	}

	server := serverMock()
	defer server.Close()

	httpClient := httpcli.NewClient(
		client.Codec("application/json", jsoncodec.NewCodec()),
	)

	var (
		ctx = metadata.NewOutgoingContext(
			context.Background(),
			metadata.Pairs("Authorization", "Bearer token", "My-Header", "My-Header-Value"),
		)
		req = &request{UserId: "123", OrderId: 456}
		rsp = &response{}

		respMetadata = metadata.Metadata{}
	)

	opts := []client.CallOption{
		client.WithAddress(server.URL),
		client.WithResponseMetadata(&respMetadata),
		httpcli.Method(http.MethodPost),
		httpcli.Path("/user/products"),
		httpcli.Body("*"),
	}

	expectedSetCookie := []string{"sessionid=abc123; Path=/; HttpOnly", "theme=dark; Path=/; Max-Age=3600", "lang=en-US; Path=/"}

	err := httpClient.Call(
		ctx,
		httpClient.NewRequest("test.service", "Test.Call", req),
		rsp,
		opts...,
	)
	require.NoError(t, err)
	require.Empty(t, rsp)

	require.Equal(t, "application/json", respMetadata.GetJoined("Content-Type"))
	require.Equal(t, "My-Header-Value", respMetadata.GetJoined("My-Header"))
	for i, raw := range respMetadata.Get("Set-Cookie") {
		cookie, err := http.ParseSetCookie(raw)
		require.NoError(t, err)
		require.Equal(t, expectedSetCookie[i], cookie.String())
	}
}

func TestClient_Call_NoContent(t *testing.T) {
	type (
		request  = pb.Test_Client_Call_Request
		response = pb.Test_Client_Call_Response
	)

	serverMock := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Validate request
			require.Equal(t, "POST", r.Method)
			require.Equal(t, "/user/products", r.URL.RequestURI())

			require.Equal(t, "application/json", r.Header.Get("Content-Type"))
			require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
			require.Equal(t, "My-Header-Value", r.Header.Get("My-Header"))

			buf, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			defer r.Body.Close()

			c := jsoncodec.NewCodec()

			req := &request{}
			err = c.Unmarshal(buf, req)
			require.NoError(t, err)
			require.True(t, proto.Equal(&request{UserId: "123", OrderId: 456}, req))

			// Return response
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("My-Header", "My-Header-Value")
			w.WriteHeader(http.StatusNoContent)
		}))
	}

	server := serverMock()
	defer server.Close()

	httpClient := httpcli.NewClient(
		client.Codec("application/json", jsoncodec.NewCodec()),
	)

	var (
		ctx = metadata.NewOutgoingContext(
			context.Background(),
			metadata.Pairs("Authorization", "Bearer token", "My-Header", "My-Header-Value"),
		)
		req = &request{UserId: "123", OrderId: 456}
		rsp = &response{}

		respMetadata = metadata.Metadata{}
	)

	opts := []client.CallOption{
		client.WithAddress(server.URL),
		client.WithResponseMetadata(&respMetadata),
		httpcli.Method(http.MethodPost),
		httpcli.Path("/user/products"),
		httpcli.Body("*"),
	}

	err := httpClient.Call(
		ctx,
		httpClient.NewRequest("test.service", "Test.Call", req),
		rsp,
		opts...,
	)
	require.NoError(t, err)
	require.Empty(t, rsp)

	require.Equal(t, "application/json", respMetadata.GetJoined("Content-Type"))
	require.Equal(t, "My-Header-Value", respMetadata.GetJoined("My-Header"))
}

func TestClient_Call_RequestTimeoutError(t *testing.T) {
	type (
		request  = pb.Test_Client_Call_Request
		response = pb.Test_Client_Call_Response
	)

	serverMock := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Millisecond)
		}))
	}

	server := serverMock()
	defer server.Close()

	httpClient := httpcli.NewClient(
		client.Codec("application/json", jsoncodec.NewCodec()),
	)

	var (
		ctx = context.Background()
		req = &request{UserId: "123", OrderId: 456}
		rsp = &response{}
	)

	opts := []client.CallOption{
		client.WithAddress(server.URL),
		client.WithRequestTimeout(time.Millisecond),
		httpcli.Method(http.MethodPost),
		httpcli.Path("/user/products"),
		httpcli.Body("*"),
	}

	err := httpClient.Call(
		ctx,
		httpClient.NewRequest("test.service", "Test.Call", req),
		rsp,
		opts...,
	)
	require.Error(t, err)
	require.Equal(t, err, &microerr.Error{
		ID:     "go.micro.client",
		Detail: "context deadline exceeded",
		Status: "Request Timeout",
		Code:   http.StatusRequestTimeout,
	})
}

func TestClient_Call_ContextDeadlineError(t *testing.T) {
	type (
		request  = pb.Test_Client_Call_Request
		response = pb.Test_Client_Call_Response
	)

	serverMock := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Millisecond)
		}))
	}

	server := serverMock()
	defer server.Close()

	httpClient := httpcli.NewClient(
		client.Codec("application/json", jsoncodec.NewCodec()),
	)

	var (
		ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond))
		req         = &request{UserId: "123", OrderId: 456}
		rsp         = &response{}
	)
	defer cancel()

	opts := []client.CallOption{
		client.WithAddress(server.URL),
		httpcli.Method(http.MethodPost),
		httpcli.Path("/user/products"),
		httpcli.Body("*"),
	}

	err := httpClient.Call(
		ctx,
		httpClient.NewRequest("test.service", "Test.Call", req),
		rsp,
		opts...,
	)
	require.Error(t, err)
	require.Equal(t, err, &microerr.Error{
		ID:     "go.micro.client",
		Detail: "context deadline exceeded",
		Status: "Request Timeout",
		Code:   http.StatusRequestTimeout,
	})
}

func TestClient_Call_ContextCanceled(t *testing.T) {
	type (
		request  = pb.Test_Client_Call_Request
		response = pb.Test_Client_Call_Response
	)

	serverMock := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Millisecond)
		}))
	}

	server := serverMock()
	defer server.Close()

	httpClient := httpcli.NewClient(
		client.Codec("application/json", jsoncodec.NewCodec()),
	)

	var (
		ctx, cancel = context.WithCancel(context.Background())
		req         = &request{UserId: "123", OrderId: 456}
		rsp         = &response{}
	)
	cancel()

	opts := []client.CallOption{
		client.WithAddress(server.URL),
		httpcli.Method(http.MethodPost),
		httpcli.Path("/user/products"),
		httpcli.Body("*"),
	}

	err := httpClient.Call(
		ctx,
		httpClient.NewRequest("test.service", "Test.Call", req),
		rsp,
		opts...,
	)
	require.Error(t, err)
	require.Equal(t, err, &microerr.Error{
		ID:     "go.micro.client",
		Detail: "context canceled",
		Status: "Request Timeout",
		Code:   http.StatusRequestTimeout,
	})
}
