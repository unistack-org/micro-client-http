package http

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"go.unistack.org/micro/v4/client"
	"go.unistack.org/micro/v4/errors"
	"go.unistack.org/micro/v4/options"
	"go.unistack.org/micro/v4/semconv"
	"go.unistack.org/micro/v4/tracer"
)

var DefaultContentType = "application/json"

type Client struct {
	funcCall   client.FuncCall
	funcStream client.FuncStream
	httpClient *http.Client
	opts       client.Options
	mu         sync.RWMutex
}

func NewClient(opts ...client.Option) *Client {
	clientOpts := client.NewOptions(opts...)

	if len(clientOpts.ContentType) == 0 {
		clientOpts.ContentType = DefaultContentType
	}

	c := &Client{opts: clientOpts}

	dialer, ok := httpDialerFromOpts(clientOpts)
	if !ok {
		dialer = defaultHTTPDialer()
	}

	c.httpClient, ok = httpClientFromOpts(clientOpts)
	if !ok {
		c.httpClient = defaultHTTPClient(dialer, clientOpts.TLSConfig)
	}

	c.funcCall = c.fnCall
	c.funcStream = c.fnStream

	return c
}

func (c *Client) Name() string {
	return c.opts.Name
}

func (c *Client) Init(opts ...client.Option) error {
	for _, o := range opts {
		o(&c.opts)
	}

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

func (c *Client) Options() client.Options {
	return c.opts
}

func (c *Client) NewRequest(service, method string, req any, opts ...client.RequestOption) client.Request {
	reqOpts := client.NewRequestOptions(opts...)
	if reqOpts.ContentType == "" {
		reqOpts.ContentType = c.opts.ContentType
	}

	return &httpRequest{
		service: service,
		method:  method,
		request: req,
		opts:    reqOpts,
	}
}

func (c *Client) Call(ctx context.Context, req client.Request, rsp any, opts ...client.CallOption) error {
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

func (c *Client) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	return c.funcStream(ctx, req, opts...)
}

func (c *Client) String() string {
	return "http"
}
