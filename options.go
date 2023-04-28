package http

import (
	"net"
	"net/http"

	"go.unistack.org/micro/v4/client"
	"go.unistack.org/micro/v4/metadata"
)

var (
	// DefaultPoolMaxStreams maximum streams on a connectioin
	// (20)
	DefaultPoolMaxStreams = 20

	// DefaultPoolMaxIdle maximum idle conns of a pool
	// (50)
	DefaultPoolMaxIdle = 50

	// DefaultMaxRecvMsgSize maximum message that client can receive
	// (4 MB).
	DefaultMaxRecvMsgSize = 1024 * 1024 * 4

	// DefaultMaxSendMsgSize maximum message that client can send
	// (4 MB).
	DefaultMaxSendMsgSize = 1024 * 1024 * 4
)

type poolMaxStreams struct{}

// PoolMaxStreams maximum streams on a connectioin
func PoolMaxStreams(n int) client.Option {
	return client.SetOption(poolMaxStreams{}, n)
}

type poolMaxIdle struct{}

// PoolMaxIdle maximum idle conns of a pool
func PoolMaxIdle(d int) client.Option {
	return client.SetOption(poolMaxIdle{}, d)
}

type maxRecvMsgSizeKey struct{}

// MaxRecvMsgSize set the maximum size of message that client can receive.
func MaxRecvMsgSize(s int) client.Option {
	return client.SetOption(maxRecvMsgSizeKey{}, s)
}

type maxSendMsgSizeKey struct{}

// MaxSendMsgSize set the maximum size of message that client can send.
func MaxSendMsgSize(s int) client.Option {
	return client.SetOption(maxSendMsgSizeKey{}, s)
}

type httpClientKey struct{}

// nolint: golint
// HTTPClient pass http.Client option to client Call
func HTTPClient(c *http.Client) client.Option {
	return client.SetOption(httpClientKey{}, c)
}

type httpDialerKey struct{}

// nolint: golint
// HTTPDialer pass net.Dialer option to client
func HTTPDialer(d *net.Dialer) client.Option {
	return client.SetOption(httpDialerKey{}, d)
}

type methodKey struct{}

// Method pass method option to client Call
func Method(m string) client.CallOption {
	return client.SetCallOption(methodKey{}, m)
}

type pathKey struct{}

// Path spcecifies path option to client Call
func Path(p string) client.CallOption {
	return client.SetCallOption(pathKey{}, p)
}

type bodyKey struct{}

// Body specifies body option to client Call
func Body(b string) client.CallOption {
	return client.SetCallOption(bodyKey{}, b)
}

type errorMapKey struct{}

func ErrorMap(m map[string]interface{}) client.CallOption {
	return client.SetCallOption(errorMapKey{}, m)
}

type structTagsKey struct{}

// StructTags pass tags slice option to client Call
func StructTags(tags []string) client.CallOption {
	return client.SetCallOption(structTagsKey{}, tags)
}

type metadataKey struct{}

// Metadata pass metadata to client Call
func Metadata(md metadata.Metadata) client.CallOption {
	return client.SetCallOption(metadataKey{}, md)
}

type cookieKey struct{}

// Cookie pass cookie to client Call
func Cookie(cookies ...string) client.CallOption {
	return client.SetCallOption(cookieKey{}, cookies)
}

type headerKey struct{}

// Header pass cookie to client Call
func Header(headers ...string) client.CallOption {
	return client.SetCallOption(headerKey{}, headers)
}
