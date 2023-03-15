package http

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"

	"go.unistack.org/micro/v3/client"
	"go.unistack.org/micro/v3/codec"
	"go.unistack.org/micro/v3/errors"
	"go.unistack.org/micro/v3/logger"
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

	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		// fast path return
		if hrsp.StatusCode == http.StatusNoContent {
			return nil
		}

		if hrsp.StatusCode < 400 {
			if log.V(logger.DebugLevel) {
				buf, rerr := io.ReadAll(hrsp.Body)
				log.Debugf(ctx, "response %s with %v", buf, hrsp.Header)
				if err != nil {
					return errors.InternalServerError("go.micro.client", rerr.Error())
				}
				hrsp.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
			}
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
		if !ok || err == nil {
			buf, cerr := io.ReadAll(hrsp.Body)
			if log.V(logger.DebugLevel) {
				log.Debugf(ctx, "response %s with %v", buf, hrsp.Header)
			}
			if cerr != nil {
				return errors.InternalServerError("go.micro.client", cerr.Error())
			}
			return errors.New("go.micro.client", string(buf), int32(hrsp.StatusCode))
		}

		if log.V(logger.DebugLevel) {
			buf, rerr := io.ReadAll(hrsp.Body)
			log.Debugf(ctx, "response %s with %v", buf, hrsp.Header)
			if err != nil {
				return errors.InternalServerError("go.micro.client", rerr.Error())
			}
			hrsp.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
		}

		if cerr := cf.ReadBody(hrsp.Body, err); cerr != nil {
			err = errors.InternalServerError("go.micro.client", cerr.Error())
		}
	}

	return err
}
