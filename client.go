package http

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"go.unistack.org/micro/v4/client"
	"go.unistack.org/micro/v4/errors"
	"go.unistack.org/micro/v4/options"
	"go.unistack.org/micro/v4/semconv"
	"go.unistack.org/micro/v4/tracer"

	"go.unistack.org/micro-client-http/v4/status"
)

var _ client.Client = (*Client)(nil)

var DefaultContentType = "application/json"

type Client struct {
	funcCall         client.FuncCall
	funcStream       client.FuncStream
	httpClient       *http.Client
	opts             client.Options
	mu               sync.RWMutex
	inflightRequests map[string]*int64
	inflightMu       sync.RWMutex
}

func NewClient(opts ...client.Option) *Client {
	clientOpts := client.NewOptions(opts...)

	if len(clientOpts.ContentType) == 0 {
		clientOpts.ContentType = DefaultContentType
	}

	c := &Client{
		opts:             clientOpts,
		inflightRequests: make(map[string]*int64),
	}

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

	// Registering the gauge metrics when creating a client
	c.registerGauges()

	return c
}

// registerGauges registers gauge metrics for tracking active requests
func (c *Client) registerGauges() {
	// A gauge for the total number of active requests
	c.opts.Meter.Gauge(semconv.ClientRequestInflight, c.getTotalInflightCount)
}

func (c *Client) getTotalInflightCount() float64 {
	total := int64(0)
	c.inflightMu.RLock()
	defer c.inflightMu.RUnlock()

	for _, count := range c.inflightRequests {
		total += atomic.LoadInt64(count)
	}
	return float64(total)
}

// getOrCreateEndpointCounter returns or creates a counter for a specific endpoint
func (c *Client) getOrCreateEndpointCounter(endpoint string) *int64 {
	c.inflightMu.RLock()
	if countPtr, exists := c.inflightRequests[endpoint]; exists {
		c.inflightMu.RUnlock()
		return countPtr
	}
	c.inflightMu.RUnlock()

	c.inflightMu.Lock()
	defer c.inflightMu.Unlock()

	if countPtr, exists := c.inflightRequests[endpoint]; exists {
		return countPtr
	}

	countPtr := new(int64)
	c.inflightRequests[endpoint] = countPtr
	return countPtr
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

	// Re-registering the metric gauge after initialization
	c.registerGauges()

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
	endpoint := req.Endpoint()

	// Incrementing the active request counter for this endpoint
	countPtr := c.getOrCreateEndpointCounter(endpoint)
	atomic.AddInt64(countPtr, 1)

	var sp tracer.Span
	ctx, sp = c.opts.Tracer.Start(ctx, endpoint+" rpc-client",
		tracer.WithSpanKind(tracer.SpanKindClient),
		tracer.WithSpanLabels("endpoint", endpoint),
	)
	defer sp.Finish()

	err := c.funcCall(ctx, req, rsp, opts...)

	// Decrementing the active request counter
	atomic.AddInt64(countPtr, -1)

	te := time.Since(ts)
	c.opts.Meter.Summary(semconv.ClientRequestLatencyMicroseconds, "endpoint", endpoint).Update(te.Seconds())
	c.opts.Meter.Histogram(semconv.ClientRequestDurationSeconds, "endpoint", endpoint).Update(te.Seconds())

	var (
		statusCode  int
		statusLabel string
	)

	if err == nil {
		statusCode = http.StatusOK
		statusLabel = "success"
	} else if st, ok := status.FromError(err); ok {
		statusCode = st.Code()
		statusLabel = "failure"
		sp.SetStatus(tracer.SpanStatusError, err.Error())
	} else if me := errors.FromError(err); me != nil {
		statusCode = int(me.Code)
		statusLabel = "failure"
		sp.SetStatus(tracer.SpanStatusError, err.Error())
	} else {
		statusCode = http.StatusInternalServerError
		statusLabel = "failure"
		sp.SetStatus(tracer.SpanStatusError, err.Error())
	}

	c.opts.Meter.Counter(semconv.ClientRequestTotal, "endpoint", endpoint, "status", statusLabel, "code", strconv.Itoa(statusCode)).Inc()

	return err
}

func (c *Client) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	return c.funcStream(ctx, req, opts...)
}

func (c *Client) String() string {
	return "http"
}
