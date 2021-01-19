package http

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/unistack-org/micro/v3/client"
	"github.com/unistack-org/micro/v3/codec"
	"github.com/unistack-org/micro/v3/errors"
)

// Implements the streamer interface
type httpStream struct {
	sync.RWMutex
	address string
	opts    client.CallOptions
	codec   codec.Codec
	context context.Context
	header  http.Header
	seq     uint64
	closed  chan bool
	err     error
	conn    net.Conn
	reader  *bufio.Reader
	request client.Request
}

var (
	errShutdown = fmt.Errorf("connection is shut down")
)

func (h *httpStream) isClosed() bool {
	select {
	case <-h.closed:
		return true
	default:
		return false
	}
}

func (h *httpStream) Context() context.Context {
	return h.context
}

func (h *httpStream) Request() client.Request {
	return h.request
}

func (h *httpStream) Response() client.Response {
	return nil
}

func (h *httpStream) Send(msg interface{}) error {
	h.Lock()
	defer h.Unlock()

	if h.isClosed() {
		h.err = errShutdown
		return errShutdown
	}

	hreq, err := newRequest(h.address, h.request, h.codec, msg, h.opts)
	if err != nil {
		return err
	}

	hreq.Header = h.header

	return hreq.Write(h.conn)
}

func (h *httpStream) Recv(msg interface{}) error {
	h.Lock()
	defer h.Unlock()

	if h.isClosed() {
		h.err = errShutdown
		return errShutdown
	}

	hrsp, err := http.ReadResponse(h.reader, new(http.Request))
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}
	defer hrsp.Body.Close()

	return parseRsp(h.context, hrsp, h.codec, msg, h.opts)
}

func (h *httpStream) Error() error {
	h.RLock()
	defer h.RUnlock()
	return h.err
}

func (h *httpStream) Close() error {
	select {
	case <-h.closed:
		return nil
	default:
		close(h.closed)
		return h.conn.Close()
	}
}
