package http

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"go.unistack.org/micro/v4/client"
)

// --------------------------------------------- HTTPClient option -----------------------------------------------------
type httpClientKey struct{}

func HTTPClient(c *http.Client) client.Option {
	return client.SetOption(httpClientKey{}, c)
}

func httpClientFromOpts(opts client.Options) (*http.Client, bool) {
	httpClient, ok := opts.Context.Value(httpClientKey{}).(*http.Client)
	return httpClient, ok
}

func defaultHTTPClient(
	dialer func(ctx context.Context, addr string) (net.Conn, error),
	tlsConfig *tls.Config,
) *http.Client {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer(ctx, addr)
		},
		ForceAttemptHTTP2:     true,
		MaxConnsPerHost:       100,
		MaxIdleConns:          20,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       tlsConfig,
	}
	return &http.Client{Transport: tr}
}

// --------------------------------------------- HTTPDialer option -----------------------------------------------------
type httpDialerKey struct{}

func HTTPDialer(d *net.Dialer) client.Option {
	return client.SetOption(httpDialerKey{}, d)
}

func httpDialerFromOpts(opts client.Options) (dialerFunc func(context.Context, string) (net.Conn, error), ok bool) {
	var d *net.Dialer

	if d, ok = opts.Context.Value(httpDialerKey{}).(*net.Dialer); ok {
		dialerFunc = func(ctx context.Context, addr string) (net.Conn, error) {
			return d.DialContext(ctx, "tcp", addr)
		}
	}

	if opts.ContextDialer != nil {
		dialerFunc, ok = opts.ContextDialer, true
	}

	return dialerFunc, ok
}

func defaultHTTPDialer() func(ctx context.Context, addr string) (net.Conn, error) {
	return func(ctx context.Context, addr string) (net.Conn, error) {
		d := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		return d.DialContext(ctx, "tcp", addr)
	}
}

// ----------------------------------------------- Method option -------------------------------------------------------
type methodKey struct{}

func Method(m string) client.CallOption {
	return client.SetCallOption(methodKey{}, m)
}

func methodFromOpts(opts client.CallOptions) (string, bool) {
	m, ok := opts.Context.Value(methodKey{}).(string)
	return m, ok
}

// ------------------------------------------------ Path option --------------------------------------------------------
type pathKey struct{}

func Path(p string) client.CallOption {
	return client.SetCallOption(pathKey{}, p)
}

func pathFromOpts(opts client.CallOptions) (string, bool) {
	p, ok := opts.Context.Value(pathKey{}).(string)
	return p, ok
}

// ------------------------------------------------ Body option --------------------------------------------------------
type bodyKey struct{}

func Body(b string) client.CallOption {
	return client.SetCallOption(bodyKey{}, b)
}

func bodyFromOpts(opts client.CallOptions) (string, bool) {
	b, ok := opts.Context.Value(bodyKey{}).(string)
	return b, ok
}

// ---------------------------------------------- ErrorMap option ------------------------------------------------------
type errorMapKey struct{}

func ErrorMap(m map[string]any) client.CallOption {
	return client.SetCallOption(errorMapKey{}, m)
}

func errorMapFromOpts(opts client.CallOptions) (map[string]any, bool) {
	errMap, ok := opts.Context.Value(errorMapKey{}).(map[string]any)
	return errMap, ok
}

// ------------------------------------------------ Cookie option ------------------------------------------------------
type cookieKey struct{}

func Cookie(cookies ...string) client.CallOption {
	return client.SetCallOption(cookieKey{}, cookies)
}

func cookieFromOpts(opts client.CallOptions) ([]string, bool) {
	c, ok := opts.Context.Value(cookieKey{}).([]string)
	return c, ok
}

// ------------------------------------------------ Header option ------------------------------------------------------
type headerKey struct{}

func Header(headers ...string) client.CallOption {
	return client.SetCallOption(headerKey{}, headers)
}

func headerFromOpts(opts client.CallOptions) ([]string, bool) {
	h, ok := opts.Context.Value(headerKey{}).([]string)
	return h, ok
}
