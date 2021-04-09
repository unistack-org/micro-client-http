// Package http provides a http client
package http

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/unistack-org/micro/v3/broker"
	"github.com/unistack-org/micro/v3/client"
	"github.com/unistack-org/micro/v3/codec"
	"github.com/unistack-org/micro/v3/errors"
	"github.com/unistack-org/micro/v3/metadata"
	"github.com/unistack-org/micro/v3/router"
	rutil "github.com/unistack-org/micro/v3/util/reflect"
)

var (
	DefaultContentType = "application/json"
)

func filterLabel(r []router.Route) []router.Route {
	//				selector.FilterLabel("protocol", "http")
	return r
}

type httpClient struct {
	opts    client.Options
	dialer  *net.Dialer
	httpcli *http.Client
	init    bool
}

func newRequest(addr string, req client.Request, ct string, cf codec.Codec, msg interface{}, opts client.CallOptions) (*http.Request, error) {
	hreq := &http.Request{Method: http.MethodPost}
	body := "*" // as like google api http annotation

	var tags []string
	var scheme string
	u, err := url.Parse(addr)
	if err != nil {
		hreq.URL = &url.URL{
			Scheme: "http",
			Host:   addr,
			Path:   req.Endpoint(),
		}
		hreq.Host = addr
		scheme = "http"
	} else {
		ep := req.Endpoint()
		if opts.Context != nil {
			if m, ok := opts.Context.Value(methodKey{}).(string); ok {
				hreq.Method = m
			}
			if p, ok := opts.Context.Value(pathKey{}).(string); ok {
				ep = p
			}
			if b, ok := opts.Context.Value(bodyKey{}).(string); ok {
				body = b
			}
			if t, ok := opts.Context.Value(structTagsKey{}).([]string); ok && len(t) > 0 {
				tags = t
			}

		}
		hreq.URL, err = u.Parse(ep)
		if err != nil {
			return nil, errors.BadRequest("go.micro.client", err.Error())
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

	path, nmsg, err := newPathRequest(hreq.URL.Path, hreq.Method, body, msg, tags)
	if err != nil {
		return nil, errors.BadRequest("go.micro.client", err.Error())
	}

	if scheme != "" {
		hreq.URL, err = url.Parse(scheme + "://" + addr + path)
	} else {
		hreq.URL, err = url.Parse(addr + path)
	}
	if err != nil {
		return nil, errors.BadRequest("go.micro.client", err.Error())
	}

	var b []byte

	if nmsg != nil {
		if ct == "application/x-www-form-urlencoded" {
			data, err := rutil.StructURLValues(nmsg, "", tags)
			if err != nil {
				return nil, errors.BadRequest("go.micro.client", err.Error())
			}
			b = []byte(data.Encode())
		} else {
			b, err = cf.Marshal(nmsg)
			if err != nil {
				return nil, errors.BadRequest("go.micro.client", err.Error())
			}
		}
	}

	if len(b) > 0 {
		hreq.Body = ioutil.NopCloser(bytes.NewBuffer(b))
		hreq.ContentLength = int64(len(b))
	}

	return hreq, nil
}

func (h *httpClient) call(ctx context.Context, addr string, req client.Request, rsp interface{}, opts client.CallOptions) error {
	header := make(http.Header, 2)
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		for k, v := range md {
			header.Set(k, v)
		}
	}

	ct := req.ContentType()
	if len(opts.ContentType) > 0 {
		ct = opts.ContentType
	}

	// set timeout in nanoseconds
	header.Set("Timeout", fmt.Sprintf("%d", opts.RequestTimeout))
	// set the content type for the request
	header.Set("Content-Type", ct)

	var cf codec.Codec
	var err error
	// get codec
	switch ct {
	case "application/x-www-form-urlencoded":
		cf, err = h.newCodec(DefaultContentType)
	default:
		cf, err = h.newCodec(ct)
	}

	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	hreq, err := newRequest(addr, req, ct, cf, req.Body(), opts)
	if err != nil {
		return err
	}

	hreq.Header = header

	// make the request
	hrsp, err := h.httpcli.Do(hreq.WithContext(ctx))
	if err != nil {
		switch err := err.(type) {
		case net.Error:
			if err.Timeout() {
				return errors.Timeout("go.micro.client", err.Error())
			}
		case *url.Error:
			if err, ok := err.Err.(net.Error); ok && err.Timeout() {
				return errors.Timeout("go.micro.client", err.Error())
			}
		}
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	defer hrsp.Body.Close()

	return parseRsp(ctx, hrsp, cf, rsp, opts)
}

func (h *httpClient) stream(ctx context.Context, addr string, req client.Request, opts client.CallOptions) (client.Stream, error) {
	var header http.Header

	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		header = make(http.Header, len(md)+2)
		for k, v := range md {
			header.Set(k, v)
		}
	} else {
		header = make(http.Header, 2)
	}

	ct := req.ContentType()
	if len(opts.ContentType) > 0 {
		ct = opts.ContentType
	}

	// set timeout in nanoseconds
	header.Set("Timeout", fmt.Sprintf("%d", opts.RequestTimeout))
	// set the content type for the request
	header.Set("Content-Type", ct)

	// get codec
	cf, err := h.newCodec(req.ContentType())
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", err.Error())
	}

	dialAddr := addr
	u, err := url.Parse(dialAddr)
	if err == nil && u.Scheme != "" && u.Host != "" {
		dialAddr = u.Host
	}
	cc, err := (h.httpcli.Transport).(*http.Transport).DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", fmt.Sprintf("Error dialing: %v", err))
	}

	return &httpStream{
		address: addr,
		context: ctx,
		closed:  make(chan bool),
		opts:    opts,
		conn:    cc,
		ct:      ct,
		cf:      cf,
		header:  header,
		reader:  bufio.NewReader(cc),
		request: req,
	}, nil
}

func (h *httpClient) newCodec(ct string) (codec.Codec, error) {
	if c, ok := h.opts.Codecs[ct]; ok {
		return c, nil
	}

	return nil, codec.ErrUnknownContentType
}

func (h *httpClient) Init(opts ...client.Option) error {
	if len(opts) == 0 && h.init {
		return nil
	}
	for _, o := range opts {
		o(&h.opts)
	}

	if err := h.opts.Broker.Init(); err != nil {
		return err
	}
	if err := h.opts.Tracer.Init(); err != nil {
		return err
	}
	if err := h.opts.Router.Init(); err != nil {
		return err
	}
	if err := h.opts.Logger.Init(); err != nil {
		return err
	}
	if err := h.opts.Meter.Init(); err != nil {
		return err
	}
	if err := h.opts.Transport.Init(); err != nil {
		return err
	}

	return nil
}

func (h *httpClient) Options() client.Options {
	return h.opts
}

func (h *httpClient) NewMessage(topic string, msg interface{}, opts ...client.MessageOption) client.Message {
	return newHTTPMessage(topic, msg, h.opts.ContentType, opts...)
}

func (h *httpClient) NewRequest(service, method string, req interface{}, opts ...client.RequestOption) client.Request {
	return newHTTPRequest(service, method, req, h.opts.ContentType, opts...)
}

func (h *httpClient) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	// make a copy of call opts
	callOpts := h.opts.CallOptions
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
		opt := client.WithRequestTimeout(d.Sub(time.Now()))
		opt(&callOpts)
	}

	// should we noop right here?
	select {
	case <-ctx.Done():
		return errors.New("go.micro.client", fmt.Sprintf("%v", ctx.Err()), 408)
	default:
	}

	// make copy of call method
	hcall := h.call

	// wrap the call in reverse
	for i := len(callOpts.CallWrappers); i > 0; i-- {
		hcall = callOpts.CallWrappers[i-1](hcall)
	}

	// use the router passed as a call option, or fallback to the rpc clients router
	if callOpts.Router == nil {
		callOpts.Router = h.opts.Router
	}

	if callOpts.Selector == nil {
		callOpts.Selector = h.opts.Selector
	}

	// inject proxy address
	// TODO: don't even bother using Lookup/Select in this case
	if len(h.opts.Proxy) > 0 {
		callOpts.Address = []string{h.opts.Proxy}
	}

	// lookup the route to send the reques to
	// TODO apply any filtering here
	routes, err := h.opts.Lookup(ctx, req, callOpts)
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	// balance the list of nodes
	next, err := callOpts.Selector.Select(routes)
	if err != nil {
		return err
	}

	// return errors.New("go.micro.client", "request timeout", 408)
	call := func(i int) error {
		// call backoff first. Someone may want an initial start delay
		t, err := callOpts.Backoff(ctx, req, i)
		if err != nil {
			return errors.InternalServerError("go.micro.client", err.Error())
		}

		// only sleep if greater than 0
		if t.Seconds() > 0 {
			time.Sleep(t)
		}

		node := next()

		// make the call
		err = hcall(ctx, node, req, rsp, callOpts)
		// record the result of the call to inform future routing decisions
		if verr := h.opts.Selector.Record(node, err); verr != nil {
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

func (h *httpClient) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	// make a copy of call opts
	callOpts := h.opts.CallOptions
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
		opt := client.WithRequestTimeout(d.Sub(time.Now()))
		opt(&callOpts)
	}

	// should we noop right here?
	select {
	case <-ctx.Done():
		return nil, errors.New("go.micro.client", fmt.Sprintf("%v", ctx.Err()), 408)
	default:
	}

	/*
		// make copy of call method
		hstream, err := h.stream()
		if err != nil {
			return nil, err
		}
		// wrap the call in reverse
		for i := len(callOpts.CallWrappers); i > 0; i-- {
			hstream = callOpts.CallWrappers[i-1](hstream)
		}
	*/

	// use the router passed as a call option, or fallback to the rpc clients router
	if callOpts.Router == nil {
		callOpts.Router = h.opts.Router
	}

	if callOpts.Selector == nil {
		callOpts.Selector = h.opts.Selector
	}

	// inject proxy address
	// TODO: don't even bother using Lookup/Select in this case
	if len(h.opts.Proxy) > 0 {
		callOpts.Address = []string{h.opts.Proxy}
	}

	// lookup the route to send the reques to
	// TODO apply any filtering here
	routes, err := h.opts.Lookup(ctx, req, callOpts)
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", err.Error())
	}

	// balance the list of nodes
	next, err := callOpts.Selector.Select(routes)
	if err != nil {
		return nil, err
	}

	call := func(i int) (client.Stream, error) {
		// call backoff first. Someone may want an initial start delay
		t, err := callOpts.Backoff(ctx, req, i)
		if err != nil {
			return nil, errors.InternalServerError("go.micro.client", err.Error())
		}

		// only sleep if greater than 0
		if t.Seconds() > 0 {
			time.Sleep(t)
		}

		node := next()

		stream, err := h.stream(ctx, node, req, callOpts)

		// record the result of the call to inform future routing decisions
		if verr := h.opts.Selector.Record(node, err); verr != nil {
			return nil, verr
		}

		// try and transform the error to a go-micro error
		if verr, ok := err.(*errors.Error); ok {
			return nil, verr
		}

		return stream, err
	}

	type response struct {
		stream client.Stream
		err    error
	}

	ch := make(chan response, callOpts.Retries)
	var grr error

	for i := 0; i <= callOpts.Retries; i++ {
		go func() {
			s, err := call(i)
			ch <- response{s, err}
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

func (h *httpClient) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	options := client.NewPublishOptions(opts...)

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(2)
	}
	md["Content-Type"] = p.ContentType()
	md["Micro-Topic"] = p.Topic()

	cf, err := h.newCodec(p.ContentType())
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	var body []byte

	// passed in raw data
	if d, ok := p.Payload().(*codec.Frame); ok {
		body = d.Data
	} else {
		b := bytes.NewBuffer(nil)
		if err := cf.Write(b, &codec.Message{Type: codec.Event}, p.Payload()); err != nil {
			return errors.InternalServerError("go.micro.client", err.Error())
		}
		body = b.Bytes()
	}

	topic := p.Topic()

	// get proxy
	if prx := os.Getenv("MICRO_PROXY"); len(prx) > 0 {
		options.Exchange = prx
	}

	// get the exchange
	if len(options.Exchange) > 0 {
		topic = options.Exchange
	}

	return h.opts.Broker.Publish(ctx, topic, &broker.Message{
		Header: md,
		Body:   body,
	}, broker.PublishContext(ctx))
}

func (h *httpClient) String() string {
	return "http"
}

func (h *httpClient) Name() string {
	return h.opts.Name
}

func NewClient(opts ...client.Option) client.Client {
	options := client.NewOptions(opts...)

	if len(options.ContentType) == 0 {
		options.ContentType = DefaultContentType
	}

	rc := &httpClient{
		opts: options,
	}

	dialer, ok := options.Context.Value(httpDialerKey{}).(*net.Dialer)
	if !ok {
		dialer = &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}
	}

	if httpcli, ok := options.Context.Value(httpClientKey{}).(*http.Client); ok {
		rc.httpcli = httpcli
	} else {
		// TODO customTransport := http.DefaultTransport.(*http.Transport).Clone()
		tr := &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialer.DialContext,
			ForceAttemptHTTP2:     true,
			MaxConnsPerHost:       100,
			MaxIdleConns:          20,
			IdleConnTimeout:       60 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       options.TLSConfig,
		}
		rc.httpcli = &http.Client{Transport: tr}
	}
	c := client.Client(rc)

	// wrap in reverse
	for i := len(options.Wrappers); i > 0; i-- {
		c = options.Wrappers[i-1](c)
	}

	return c
}
