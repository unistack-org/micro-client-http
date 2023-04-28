package http

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"go.unistack.org/micro/v4/client"
	"go.unistack.org/micro/v4/codec"
	"go.unistack.org/micro/v4/errors"
	"go.unistack.org/micro/v4/logger"
)

// Implements the streamer interface
type httpStream struct {
	err     error
	conn    net.Conn
	cf      codec.Codec
	context context.Context
	logger  logger.Logger
	request client.Request
	closed  chan bool
	reader  *bufio.Reader
	address string
	ct      string
	opts    client.CallOptions
	sync.RWMutex
}

var errShutdown = fmt.Errorf("connection is shut down")

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

func (h *httpStream) SendMsg(msg interface{}) error {
	return h.Send(msg)
}

func (h *httpStream) Send(msg interface{}) error {
	h.Lock()
	defer h.Unlock()

	if h.isClosed() {
		h.err = errShutdown
		return errShutdown
	}

	hreq, err := newRequest(h.context, h.logger, h.address, h.request, h.ct, h.cf, msg, h.opts)
	if err != nil {
		return err
	}

	return hreq.Write(h.conn)
}

func (h *httpStream) RecvMsg(msg interface{}) error {
	return h.Recv(msg)
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

	return h.parseRsp(h.context, h.logger, hrsp, h.cf, msg, h.opts)
}

func (h *httpStream) Error() error {
	h.RLock()
	defer h.RUnlock()
	return h.err
}

func (h *httpStream) CloseSend() error {
	return h.Close()
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

func (h *httpStream) parseRsp(ctx context.Context, log logger.Logger, hrsp *http.Response, cf codec.Codec, rsp interface{}, opts client.CallOptions) error {
	var err error
	var buf []byte

	// fast path return
	if hrsp.StatusCode == http.StatusNoContent {
		return nil
	}

	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		if hrsp.Body != nil {
			buf, err = io.ReadAll(hrsp.Body)
			if err != nil {
				if log.V(logger.ErrorLevel) {
					log.Errorf(ctx, "failed to read body: %v", err)
				}
				return errors.InternalServerError("go.micro.client", string(buf))
			}
		}

		if log.V(logger.DebugLevel) {
			log.Debugf(ctx, "response %s with %v", buf, hrsp.Header)
		}

		if hrsp.StatusCode < 400 {
			if err = cf.Unmarshal(buf, rsp); err != nil {
				return errors.InternalServerError("go.micro.client", err.Error())
			}
			return nil
		}

		var rerr interface{}
		errmap, ok := opts.Context.Value(errorMapKey{}).(map[string]interface{})
		if ok && errmap != nil {
			if rerr, ok = errmap[fmt.Sprintf("%d", hrsp.StatusCode)].(error); !ok {
				rerr, ok = errmap["default"].(error)
			}
		}
		if !ok || rerr == nil {
			return errors.New("go.micro.client", string(buf), int32(hrsp.StatusCode))
		}

		if cerr := cf.Unmarshal(buf, rerr); cerr != nil {
			return errors.InternalServerError("go.micro.client", cerr.Error())
		}

		if err, ok = rerr.(error); !ok {
			err = &Error{rerr}
		}
	}

	return err
}
