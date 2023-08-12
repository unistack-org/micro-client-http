package http

import (
	"net"
	"net/http"

	"go.unistack.org/micro/v4/metadata"
	"go.unistack.org/micro/v4/options"
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
func PoolMaxStreams(n int) options.Option {
	return options.ContextOption(poolMaxStreams{}, n)
}

type poolMaxIdle struct{}

// PoolMaxIdle maximum idle conns of a pool
func PoolMaxIdle(d int) options.Option {
	return options.ContextOption(poolMaxIdle{}, d)
}

type maxRecvMsgSizeKey struct{}

// MaxRecvMsgSize set the maximum size of message that client can receive.
func MaxRecvMsgSize(s int) options.Option {
	return options.ContextOption(maxRecvMsgSizeKey{}, s)
}

type maxSendMsgSizeKey struct{}

// MaxSendMsgSize set the maximum size of message that client can send.
func MaxSendMsgSize(s int) options.Option {
	return options.ContextOption(maxSendMsgSizeKey{}, s)
}

type httpClientKey struct{}

// nolint: golint
// HTTPClient pass http.Client option to client Call
func HTTPClient(c *http.Client) options.Option {
	return options.ContextOption(httpClientKey{}, c)
}

type httpDialerKey struct{}

// nolint: golint
// HTTPDialer pass net.Dialer option to client
func HTTPDialer(d *net.Dialer) options.Option {
	return options.ContextOption(httpDialerKey{}, d)
}

type methodKey struct{}

// Method pass method option to client Call
func Method(m string) options.Option {
	return options.ContextOption(methodKey{}, m)
}

type pathKey struct{}

// Path spcecifies path option to client Call
func Path(p string) options.Option {
	return options.ContextOption(pathKey{}, p)
}

type bodyKey struct{}

// Body specifies body option to client Call
func Body(b string) options.Option {
	return options.ContextOption(bodyKey{}, b)
}

type errorMapKey struct{}

func ErrorMap(m map[string]interface{}) options.Option {
	return options.ContextOption(errorMapKey{}, m)
}

type structTagsKey struct{}

// StructTags pass tags slice option to client Call
func StructTags(tags []string) options.Option {
	return options.ContextOption(structTagsKey{}, tags)
}

type metadataKey struct{}

// Metadata pass metadata to client Call
func Metadata(md metadata.Metadata) options.Option {
	return options.ContextOption(metadataKey{}, md)
}

type cookieKey struct{}

// Cookie pass cookie to client Call
func Cookie(cookies ...string) options.Option {
	return options.ContextOption(cookieKey{}, cookies)
}

type headerKey struct{}

// Header pass cookie to client Call
func Header(headers ...string) options.Option {
	return options.ContextOption(headerKey{}, headers)
}
