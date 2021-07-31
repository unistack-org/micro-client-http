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
	"strings"
	"sync"
	"time"

	"github.com/unistack-org/micro/v3/broker"
	"github.com/unistack-org/micro/v3/client"
	"github.com/unistack-org/micro/v3/codec"
	"github.com/unistack-org/micro/v3/errors"
	"github.com/unistack-org/micro/v3/metadata"
)

var DefaultContentType = "application/json"

/*
func filterLabel(r []router.Route) []router.Route {
	//				selector.FilterLabel("protocol", "http")
	return r
}
*/

type httpClient struct {
	httpcli *http.Client
	opts    client.Options
	sync.RWMutex
	init bool
}

func newRequest(ctx context.Context, addr string, req client.Request, ct string, cf codec.Codec, msg interface{}, opts client.CallOptions) (*http.Request, error) {
	var tags []string
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
		return nil, errors.BadRequest("go.micro.client", err.Error())
	}

	path, nmsg, err := newPathRequest(u.Path, method, body, msg, tags)
	if err != nil {
		return nil, errors.BadRequest("go.micro.client", err.Error())
	}

	u, err = url.Parse(fmt.Sprintf("%s://%s%s", scheme, host, path))
	if err != nil {
		return nil, errors.BadRequest("go.micro.client", err.Error())
	}

	b, err := cf.Marshal(nmsg)
	if err != nil {
		return nil, errors.BadRequest("go.micro.client", err.Error())
	}

	var hreq *http.Request
	if len(b) > 0 {
		hreq, err = http.NewRequestWithContext(ctx, method, u.String(), ioutil.NopCloser(bytes.NewBuffer(b)))
		hreq.ContentLength = int64(len(b))
	} else {
		hreq, err = http.NewRequestWithContext(ctx, method, u.String(), nil)
	}

	if err != nil {
		return nil, errors.BadRequest("go.micro.client", err.Error())
	}

	header := make(http.Header)

	if opts.Context != nil {
		if md, ok := opts.Context.Value(metadataKey{}).(metadata.Metadata); ok {
			for k, v := range md {
				header.Set(k, v)
			}
		}
	}

	if opts.AuthToken != "" {
		hreq.Header.Set(metadata.HeaderAuthorization, opts.AuthToken)
	}

	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		for k, v := range md {
			hreq.Header.Set(k, v)
		}
	}

	// set timeout in nanoseconds
	if opts.StreamTimeout > time.Duration(0) {
		hreq.Header.Set(metadata.HeaderTimeout, fmt.Sprintf("%d", opts.StreamTimeout))
	}
	if opts.RequestTimeout > time.Duration(0) {
		hreq.Header.Set(metadata.HeaderTimeout, fmt.Sprintf("%d", opts.RequestTimeout))
	}

	// set the content type for the request
	hreq.Header.Set(metadata.HeaderContentType, ct)

	return hreq, nil
}

func (h *httpClient) call(ctx context.Context, addr string, req client.Request, rsp interface{}, opts client.CallOptions) error {
	ct := req.ContentType()
	if len(opts.ContentType) > 0 {
		ct = opts.ContentType
	}

	cf, err := h.newCodec(ct)
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}
	hreq, err := newRequest(ctx, addr, req, ct, cf, req.Body(), opts)
	if err != nil {
		return err
	}

	// make the request
	hrsp, err := h.httpcli.Do(hreq)
	if err != nil {
		switch err := err.(type) {
		case *url.Error:
			if err, ok := err.Err.(net.Error); ok && err.Timeout() {
				return errors.Timeout("go.micro.client", err.Error())
			}
		case net.Error:
			if err.Timeout() {
				return errors.Timeout("go.micro.client", err.Error())
			}
		}
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	defer hrsp.Body.Close()

	return h.parseRsp(ctx, hrsp, rsp, opts)
}

func (h *httpClient) stream(ctx context.Context, addr string, req client.Request, opts client.CallOptions) (client.Stream, error) {
	ct := req.ContentType()
	if len(opts.ContentType) > 0 {
		ct = opts.ContentType
	}

	// get codec
	cf, err := h.newCodec(ct)
	if err != nil {
		return nil, errors.InternalServerError("go.micro.client", err.Error())
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
		reader:  bufio.NewReader(cc),
		request: req,
	}, nil
}

func (h *httpClient) newCodec(ct string) (codec.Codec, error) {
	h.RLock()
	defer h.RUnlock()

	if idx := strings.IndexRune(ct, ';'); idx >= 0 {
		ct = ct[:idx]
	}

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
		t, cerr := callOpts.Backoff(ctx, req, i)
		if cerr != nil {
			return nil, errors.InternalServerError("go.micro.client", cerr.Error())
		}

		// only sleep if greater than 0
		if t.Seconds() > 0 {
			time.Sleep(t)
		}

		node := next()

		stream, cerr := h.stream(ctx, node, req, callOpts)

		// record the result of the call to inform future routing decisions
		if verr := h.opts.Selector.Record(node, cerr); verr != nil {
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

func (h *httpClient) BatchPublish(ctx context.Context, p []client.Message, opts ...client.PublishOption) error {
	return h.publish(ctx, p, opts...)
}

func (h *httpClient) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	return h.publish(ctx, []client.Message{p}, opts...)
}

func (h *httpClient) publish(ctx context.Context, ps []client.Message, opts ...client.PublishOption) error {
	options := client.NewPublishOptions(opts...)

	// get proxy
	exchange := ""
	if v, ok := os.LookupEnv("MICRO_PROXY"); ok {
		exchange = v
	}

	omd, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		omd = metadata.New(2)
	}

	msgs := make([]*broker.Message, 0, len(ps))

	for _, p := range ps {
		md := metadata.Copy(omd)
		md[metadata.HeaderContentType] = p.ContentType()
		md[metadata.HeaderTopic] = p.Topic()

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
		if len(exchange) > 0 {
			topic = exchange
		}

		md.Set(metadata.HeaderTopic, topic)
		msgs = append(msgs, &broker.Message{Header: md, Body: body})
	}

	return h.opts.Broker.BatchPublish(ctx, msgs,
		broker.PublishContext(ctx),
		broker.PublishBodyOnly(options.BodyOnly),
	)
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
