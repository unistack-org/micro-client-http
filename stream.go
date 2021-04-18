package http

import (
	"bufio"
	"context"
	"fmt"
	"io"
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
	ct      string
	cf      codec.Codec
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

	hreq, err := newRequest(h.address, h.request, h.ct, h.cf, msg, h.opts)
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

	return h.parseRsp(h.context, hrsp, h.cf, msg, h.opts)
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

func (h *httpStream) parseRsp(ctx context.Context, hrsp *http.Response, cf codec.Codec, rsp interface{}, opts client.CallOptions) error {
	var err error

	// fast path return
	if hrsp.StatusCode == http.StatusNoContent {
		return nil
	}

	if hrsp.StatusCode < 400 {
		if err = cf.ReadBody(hrsp.Body, rsp); err != nil {
			return errors.InternalServerError("go.micro.client", err.Error())
		}
		return nil
	}

	errmap, ok := opts.Context.Value(errorMapKey{}).(map[string]interface{})
	if ok && errmap != nil {
		if err, ok = errmap[fmt.Sprintf("%d", hrsp.StatusCode)].(error); !ok {
			err, ok = errmap["default"].(error)
		}
	}
	if err == nil {
		buf, err := io.ReadAll(hrsp.Body)
		if err != nil {
			errors.InternalServerError("go.micro.client", err.Error())
		}
		return errors.New("go.micro.client", string(buf), int32(hrsp.StatusCode))
	}

	if cerr := cf.ReadBody(hrsp.Body, err); cerr != nil {
		err = errors.InternalServerError("go.micro.client", cerr.Error())
	}

	return err
}
