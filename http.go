// Package http provides a http client
package http // import "go.unistack.org/micro-client-http/v3"

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.unistack.org/micro/v4/client"
	"go.unistack.org/micro/v4/codec"
	"go.unistack.org/micro/v4/errors"
	"go.unistack.org/micro/v4/logger"
	"go.unistack.org/micro/v4/metadata"
	"go.unistack.org/micro/v4/options"
	"go.unistack.org/micro/v4/selector"
	"go.unistack.org/micro/v4/semconv"
	"go.unistack.org/micro/v4/tracer"
	rutil "go.unistack.org/micro/v4/util/reflect"
)

var DefaultContentType = "application/json"

/*
func filterLabel(r []router.Route) []router.Route {
	//				selector.FilterLabel("protocol", "http")
	return r
}
*/

type httpClient struct {
	funcCall   client.FuncCall
	funcStream client.FuncStream
	httpcli    *http.Client
	opts       client.Options
	mu         sync.RWMutex
}

func newRequest(ctx context.Context, log logger.Logger, addr string, req client.Request, ct string, cf codec.Codec, msg interface{}, opts client.CallOptions) (*http.Request, error) {
	var tags []string
	var parameters map[string]map[string]string
	scheme := "http"
	method := http.MethodPost
	body := "*" // as like google api http annotation
	host := addr
	path := req.Endpoint()

	u, err := url.Parse(addr)
	if err == nil {
		scheme = u.Scheme
		path = u.Path
		host = u.Host
	} else {
		u = &url.URL{Scheme: scheme, Path: path, Host: host}
	}

	// nolint: nestif
	if opts.Context != nil {
		if m, ok := opts.Context.Value(methodKey{}).(string); ok {
			method = m
		}
		if p, ok := opts.Context.Value(pathKey{}).(string); ok {
			path += p
		}
		if b, ok := opts.Context.Value(bodyKey{}).(string); ok {
			body = b
		}
		if t, ok := opts.Context.Value(structTagsKey{}).([]string); ok && len(t) > 0 {
			tags = t
		}
		if k, ok := opts.Context.Value(headerKey{}).([]string); ok && len(k) > 0 {
			if parameters == nil {
				parameters = make(map[string]map[string]string)
			}
			m, ok := parameters["header"]
			if !ok {
				m = make(map[string]string)
				parameters["header"] = m
			}
			for idx := 0; idx < len(k)/2; idx += 2 {
				m[k[idx]] = k[idx+1]
			}
		}
		if k, ok := opts.Context.Value(cookieKey{}).([]string); ok && len(k) > 0 {
			if parameters == nil {
				parameters = make(map[string]map[string]string)
			}
			m, ok := parameters["cookie"]
			if !ok {
				m = make(map[string]string)
				parameters["cookie"] = m
			}
			for idx := 0; idx < len(k)/2; idx += 2 {
				m[k[idx]] = k[idx+1]
			}
		}
	}

	if len(tags) == 0 {
		switch ct {
		default:
			tags = append(tags, "json", "protobuf")
		case "text/xml":
			tags = append(tags, "xml")
		}
	}

	if path == "" {
		path = req.Endpoint()
	}

	u, err = u.Parse(path)
	if err != nil {
		return nil, errors.BadRequest("go.micro.client", "%+v", err)
	}

	var nmsg interface{}
	if len(u.Query()) > 0 {
		path, nmsg, err = newPathRequest(u.Path+"?"+u.RawQuery, method, body, msg, tags, parameters)
	} else {
		path, nmsg, err = newPathRequest(u.Path, method, body, msg, tags, parameters)
	}

	if err != nil {
		return nil, errors.BadRequest("go.micro.client", "%+v", err)
	}

	u, err = url.Parse(fmt.Sprintf("%s://%s%s", scheme, host, path))
	if err != nil {
		return nil, errors.BadRequest("go.micro.client", "%+v", err)
	}

	var cookies []*http.Cookie
	header := make(http.Header)
	if opts.Context != nil {
		if md, ok := opts.Context.Value(metadataKey{}).(metadata.Metadata); ok {
			for k, v := range md {
				header[k] = append(header[k], v...)
			}
		}
	}
	if opts.AuthToken != "" {
		header.Set(metadata.HeaderAuthorization, opts.AuthToken)
	}
	if opts.RequestMetadata != nil {
		for k, v := range opts.RequestMetadata {
			header[k] = append(header[k], v...)
		}
	}

	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		for k, v := range md {
			header[k] = append(header[k], v...)
		}
	}

	// set timeout in nanoseconds
	if opts.StreamTimeout > time.Duration(0) {
		header.Set(metadata.HeaderTimeout, fmt.Sprintf("%d", opts.StreamTimeout))
	}
	if opts.RequestTimeout > time.Duration(0) {
		header.Set(metadata.HeaderTimeout, fmt.Sprintf("%d", opts.RequestTimeout))
	}

	// set the content type for the request
	header.Set(metadata.HeaderContentType, ct)
	var v interface{}

	for km, vm := range parameters {
		for k, required := range vm {
			v, err = rutil.StructFieldByPath(msg, k)
			if err != nil {
				return nil, errors.BadRequest("go.micro.client", "%+v", err)
			}
			if rutil.IsZero(v) {
				if required == "true" {
					return nil, errors.BadRequest("go.micro.client", "required field %s not set", k)
				}
				continue
			}

			switch km {
			case "header":
				header.Set(k, fmt.Sprintf("%v", v))
			case "cookie":
				cookies = append(cookies, &http.Cookie{Name: k, Value: fmt.Sprintf("%v", v)})
			}
		}
	}

	b, err := cf.Marshal(nmsg)
	if err != nil {
		return nil, errors.BadRequest("go.micro.client", "%+v", err)
	}

	var hreq *http.Request
	if len(b) > 0 {
		hreq, err = http.NewRequestWithContext(ctx, method, u.String(), io.NopCloser(bytes.NewBuffer(b)))
		hreq.ContentLength = int64(len(b))
		header.Set("Content-Length", fmt.Sprintf("%d", hreq.ContentLength))
	} else {
		hreq, err = http.NewRequestWithContext(ctx, method, u.String(), nil)
	}

	if err != nil {
		return nil, errors.BadRequest("go.micro.client", "%+v", err)
	}

	hreq.Header = header
	for _, cookie := range cookies {
		hreq.AddCookie(cookie)
	}

	if log.V(logger.DebugLevel) {
		log.Debug(ctx, fmt.Sprintf("request %s to %s with headers %v body %s", method, u.String(), hreq.Header, b))
	}

	return hreq, nil
}

func (c *httpClient) call(ctx context.Context, addr string, req client.Request, rsp interface{}, opts client.CallOptions) error {
	ct := req.ContentType()
	if len(opts.ContentType) > 0 {
		ct = opts.ContentType
	}

	cf, err := c.newCodec(ct)
	if err != nil {
		return errors.BadRequest("go.micro.client", "%+v", err)
	}
	hreq, err := newRequest(ctx, c.opts.Logger, addr, req, ct, cf, req.Body(), opts)
	if err != nil {
		return err
	}

	// make the request
	hrsp, err := c.httpcli.Do(hreq)
	if err != nil {
		switch err := err.(type) {
		case *url.Error:
			if err, ok := err.Err.(net.Error); ok && err.Timeout() {
				return errors.Timeout("go.micro.client", "%+v", err)
			}
		case net.Error:
			if err.Timeout() {
				return errors.Timeout("go.micro.client", "%+v", err)
			}
		}
		return errors.InternalServerError("go.micro.client", "%+v", err)
	}

	defer hrsp.Body.Close()

	return c.parseRsp(ctx, hrsp, rsp, opts)
}

func (c *httpClient) stream(ctx context.Context, addr string, req client.Request, opts client.CallOptions) (client.Stream, error) {
	ct := req.ContentType()
	if len(opts.ContentType) > 0 {
		ct = opts.ContentType
	}

	// get codec
	cf, err := c.newCodec(ct)
	if err != nil {
		return nil, errors.BadRequest("go.micro.client", "%+v", err)
	}

	cc, err := (c.httpcli.Transport).(*http.Transport).DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", "Error dialing: %v", err)
	}

	return &httpStream{
		address: addr,
		logger:  c.opts.Logger,
		context: ctx,
		closed:  make(chan bool),
		opts:    opts,
		conn:    cc,
		ct:      ct,
		cf:      cf,
		reader:  bufio.NewReader(cc),
		request: req,
	}, nil
}

func (c *httpClient) newCodec(ct string) (codec.Codec, error) {
	c.mu.RLock()

	if idx := strings.IndexRune(ct, ';'); idx >= 0 {
		ct = ct[:idx]
	}

	if cf, ok := c.opts.Codecs[ct]; ok {
		c.mu.RUnlock()
		return cf, nil
	}

	c.mu.RUnlock()
	return nil, codec.ErrUnknownContentType
}

func (c *httpClient) Init(opts ...client.Option) error {
	for _, o := range opts {
		o(&c.opts)
	}

	c.funcCall = c.fnCall
	c.funcStream = c.fnStream

	c.opts.Hooks.EachPrev(func(hook options.Hook) {
		switch h := hook.(type) {
		case client.HookCall:
			c.funcCall = h(c.funcCall)
		case client.HookStream:
			c.funcStream = h(c.funcStream)
		}
	})

	return nil
}

func (c *httpClient) Options() client.Options {
	return c.opts
}

func (c *httpClient) NewRequest(service, method string, req interface{}, opts ...client.RequestOption) client.Request {
	return newHTTPRequest(service, method, req, c.opts.ContentType, opts...)
}

func (c *httpClient) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	ts := time.Now()
	c.opts.Meter.Counter(semconv.ClientRequestInflight, "endpoint", req.Endpoint()).Inc()
	var sp tracer.Span
	ctx, sp = c.opts.Tracer.Start(ctx, req.Endpoint()+" rpc-client",
		tracer.WithSpanKind(tracer.SpanKindClient),
		tracer.WithSpanLabels("endpoint", req.Endpoint()),
	)
	err := c.funcCall(ctx, req, rsp, opts...)
	c.opts.Meter.Counter(semconv.ClientRequestInflight, "endpoint", req.Endpoint()).Dec()
	te := time.Since(ts)
	c.opts.Meter.Summary(semconv.ClientRequestLatencyMicroseconds, "endpoint", req.Endpoint()).Update(te.Seconds())
	c.opts.Meter.Histogram(semconv.ClientRequestDurationSeconds, "endpoint", req.Endpoint()).Update(te.Seconds())

	if me := errors.FromError(err); me == nil {
		sp.Finish()
		c.opts.Meter.Counter(semconv.ClientRequestTotal, "endpoint", req.Endpoint(), "status", "success", "code", strconv.Itoa(int(200))).Inc()
	} else {
		sp.SetStatus(tracer.SpanStatusError, err.Error())
		c.opts.Meter.Counter(semconv.ClientRequestTotal, "endpoint", req.Endpoint(), "status", "failure", "code", strconv.Itoa(int(me.Code))).Inc()
	}

	return err
}

func (c *httpClient) fnCall(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	// make a copy of call opts
	callOpts := c.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// check if we already have a deadline
	d, ok := ctx.Deadline()
	if !ok {
		var cancel context.CancelFunc
		// no deadline so we create a new one
		ctx, cancel = context.WithTimeout(ctx, callOpts.RequestTimeout)
		defer cancel()
	} else {
		// got a deadline so no need to setup context
		// but we need to set the timeout we pass along
		opt := client.WithRequestTimeout(time.Until(d))
		opt(&callOpts)
	}

	// should we noop right here?
	select {
	case <-ctx.Done():
		return errors.New("go.micro.client", fmt.Sprintf("%v", ctx.Err()), 408)
	default:
	}

	// make copy of call method
	hcall := c.call

	// use the router passed as a call option, or fallback to the rpc clients router
	if callOpts.Router == nil {
		callOpts.Router = c.opts.Router
	}

	if callOpts.Selector == nil {
		callOpts.Selector = c.opts.Selector
	}

	// inject proxy address
	// TODO: don't even bother using Lookup/Select in this case
	if len(c.opts.Proxy) > 0 {
		callOpts.Address = []string{c.opts.Proxy}
	}

	var next selector.Next

	// return errors.New("go.micro.client", "request timeout", 408)
	call := func(i int) error {
		// call backoff first. Someone may want an initial start delay
		t, err := callOpts.Backoff(ctx, req, i)
		if err != nil {
			return errors.InternalServerError("go.micro.client", "%+v", err)
		}

		// only sleep if greater than 0
		if t.Seconds() > 0 {
			time.Sleep(t)
		}

		if next == nil {
			var routes []string
			// lookup the route to send the reques to
			// TODO apply any filtering here
			routes, err = c.opts.Lookup(ctx, req, callOpts)
			if err != nil {
				return errors.InternalServerError("go.micro.client", "%+v", err)
			}

			// balance the list of nodes
			next, err = callOpts.Selector.Select(routes)
			if err != nil {
				return err
			}
		}

		node := next()

		// make the call
		err = hcall(ctx, node, req, rsp, callOpts)
		// record the result of the call to inform future routing decisions
		if verr := c.opts.Selector.Record(node, err); verr != nil {
			return verr
		}

		// try and transform the error to a go-micro error
		if verr, ok := err.(*errors.Error); ok {
			return verr
		}

		return err
	}

	ch := make(chan error, callOpts.Retries)
	var gerr error

	for i := 0; i <= callOpts.Retries; i++ {
		go func() {
			ch <- call(i)
		}()

		select {
		case <-ctx.Done():
			return errors.New("go.micro.client", fmt.Sprintf("%v", ctx.Err()), 408)
		case err := <-ch:
			// if the call succeeded lets bail early
			if err == nil {
				return nil
			}

			retry, rerr := callOpts.Retry(ctx, req, i, err)
			if rerr != nil {
				return rerr
			}

			if !retry {
				return err
			}

			gerr = err
		}
	}

	return gerr
}

func (c *httpClient) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	ts := time.Now()
	c.opts.Meter.Counter(semconv.ClientRequestInflight, "endpoint", req.Endpoint()).Inc()
	var sp tracer.Span
	ctx, sp = c.opts.Tracer.Start(ctx, req.Endpoint()+" rpc-client",
		tracer.WithSpanKind(tracer.SpanKindClient),
		tracer.WithSpanLabels("endpoint", req.Endpoint()),
	)
	stream, err := c.funcStream(ctx, req, opts...)
	c.opts.Meter.Counter(semconv.ClientRequestInflight, "endpoint", req.Endpoint()).Dec()
	te := time.Since(ts)
	c.opts.Meter.Summary(semconv.ClientRequestLatencyMicroseconds, "endpoint", req.Endpoint()).Update(te.Seconds())
	c.opts.Meter.Histogram(semconv.ClientRequestDurationSeconds, "endpoint", req.Endpoint()).Update(te.Seconds())

	if me := errors.FromError(err); me == nil {
		sp.Finish()
		c.opts.Meter.Counter(semconv.ClientRequestTotal, "endpoint", req.Endpoint(), "status", "success", "code", strconv.Itoa(int(200))).Inc()
	} else {
		sp.SetStatus(tracer.SpanStatusError, err.Error())
		c.opts.Meter.Counter(semconv.ClientRequestTotal, "endpoint", req.Endpoint(), "status", "failure", "code", strconv.Itoa(int(me.Code))).Inc()
	}

	return stream, err
}

func (c *httpClient) fnStream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	var err error

	// make a copy of call opts
	callOpts := c.opts.CallOptions
	for _, o := range opts {
		o(&callOpts)
	}

	// check if we already have a deadline
	d, ok := ctx.Deadline()
	if !ok && callOpts.StreamTimeout > time.Duration(0) {
		var cancel context.CancelFunc
		// no deadline so we create a new one
		ctx, cancel = context.WithTimeout(ctx, callOpts.StreamTimeout)
		defer cancel()
	} else {
		// got a deadline so no need to setup context
		// but we need to set the timeout we pass along
		o := client.WithStreamTimeout(time.Until(d))
		o(&callOpts)
	}

	// should we noop right here?
	select {
	case <-ctx.Done():
		return nil, errors.New("go.micro.client", fmt.Sprintf("%v", ctx.Err()), 408)
	default:
	}

	/*
		// make copy of call method
		hstream := h.stream
		// wrap the call in reverse
		for i := len(callOpts.CallWrappers); i > 0; i-- {
			hstream = callOpts.CallWrappers[i-1](hstream)
		}
	*/

	// use the router passed as a call option, or fallback to the rpc clients router
	if callOpts.Router == nil {
		callOpts.Router = c.opts.Router
	}

	if callOpts.Selector == nil {
		callOpts.Selector = c.opts.Selector
	}

	// inject proxy address
	// TODO: don't even bother using Lookup/Select in this case
	if len(c.opts.Proxy) > 0 {
		callOpts.Address = []string{c.opts.Proxy}
	}

	var next selector.Next

	call := func(i int) (client.Stream, error) {
		// call backoff first. Someone may want an initial start delay
		t, cerr := callOpts.Backoff(ctx, req, i)
		if cerr != nil {
			return nil, errors.InternalServerError("go.micro.client", "%+v", cerr)
		}

		// only sleep if greater than 0
		if t.Seconds() > 0 {
			time.Sleep(t)
		}

		if next == nil {
			var routes []string
			// lookup the route to send the reques to
			// TODO apply any filtering here
			routes, err = c.opts.Lookup(ctx, req, callOpts)
			if err != nil {
				return nil, errors.InternalServerError("go.micro.client", "%+v", err)
			}

			// balance the list of nodes
			next, err = callOpts.Selector.Select(routes)
			if err != nil {
				return nil, err
			}
		}

		node := next()

		stream, cerr := c.stream(ctx, node, req, callOpts)

		// record the result of the call to inform future routing decisions
		if verr := c.opts.Selector.Record(node, cerr); verr != nil {
			return nil, verr
		}

		// try and transform the error to a go-micro error
		if verr, ok := cerr.(*errors.Error); ok {
			return nil, verr
		}

		return stream, cerr
	}

	type response struct {
		stream client.Stream
		err    error
	}

	ch := make(chan response, callOpts.Retries)
	var grr error

	for i := 0; i <= callOpts.Retries; i++ {
		go func() {
			s, cerr := call(i)
			ch <- response{s, cerr}
		}()

		select {
		case <-ctx.Done():
			return nil, errors.New("go.micro.client", fmt.Sprintf("%v", ctx.Err()), 408)
		case rsp := <-ch:
			// if the call succeeded lets bail early
			if rsp.err == nil {
				return rsp.stream, nil
			}

			retry, rerr := callOpts.Retry(ctx, req, i, err)
			if rerr != nil {
				return nil, rerr
			}

			if !retry {
				return nil, rsp.err
			}

			grr = rsp.err
		}
	}

	return nil, grr
}

func (c *httpClient) String() string {
	return "http"
}

func (c *httpClient) Name() string {
	return c.opts.Name
}

func NewClient(opts ...client.Option) *httpClient {
	options := client.NewOptions(opts...)

	if len(options.ContentType) == 0 {
		options.ContentType = DefaultContentType
	}

	c := &httpClient{
		opts: options,
	}

	var dialer func(context.Context, string) (net.Conn, error)
	if v, ok := options.Context.Value(httpDialerKey{}).(*net.Dialer); ok {
		dialer = func(ctx context.Context, addr string) (net.Conn, error) {
			return v.DialContext(ctx, "tcp", addr)
		}
	}
	if options.ContextDialer != nil {
		dialer = options.ContextDialer
	}
	if dialer == nil {
		dialer = func(ctx context.Context, addr string) (net.Conn, error) {
			return (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext(ctx, "tcp", addr)
		}
	}

	if httpcli, ok := options.Context.Value(httpClientKey{}).(*http.Client); ok {
		c.httpcli = httpcli
	} else {
		// TODO customTransport := http.DefaultTransport.(*http.Transport).Clone()
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
			TLSClientConfig:       options.TLSConfig,
		}
		c.httpcli = &http.Client{Transport: tr}
	}

	c.funcCall = c.fnCall
	c.funcStream = c.fnStream

	return c
}
