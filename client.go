package http

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"go.unistack.org/micro-client-http/v3/status"
	"go.unistack.org/micro/v3/broker"
	"go.unistack.org/micro/v3/client"
	"go.unistack.org/micro/v3/codec"
	"go.unistack.org/micro/v3/errors"
	"go.unistack.org/micro/v3/metadata"
	"go.unistack.org/micro/v3/options"
	"go.unistack.org/micro/v3/semconv"
	"go.unistack.org/micro/v3/tracer"
)

var _ client.Client = (*Client)(nil)

var DefaultContentType = "application/json"

type Client struct {
	funcPublish      client.FuncPublish
	funcBatchPublish client.FuncBatchPublish
	funcCall         client.FuncCall
	funcStream       client.FuncStream
	httpClient       *http.Client
	opts             client.Options
	mu               sync.RWMutex
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
	defer sp.Finish()

	err := c.funcCall(ctx, req, rsp, opts...)

	c.opts.Meter.Counter(semconv.ClientRequestInflight, "endpoint", req.Endpoint()).Dec()
	te := time.Since(ts)
	c.opts.Meter.Summary(semconv.ClientRequestLatencyMicroseconds, "endpoint", req.Endpoint()).Update(te.Seconds())
	c.opts.Meter.Histogram(semconv.ClientRequestDurationSeconds, "endpoint", req.Endpoint()).Update(te.Seconds())

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

	c.opts.Meter.Counter(semconv.ClientRequestTotal, "endpoint", req.Endpoint(), "status", statusLabel, "code", strconv.Itoa(statusCode)).Inc()

	return err
}

func (c *Client) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	return c.funcStream(ctx, req, opts...)
}

func (c *Client) String() string {
	return "http"
}

func (c *Client) BatchPublish(ctx context.Context, ps []client.Message, opts ...client.PublishOption) error {
	return c.funcBatchPublish(ctx, ps, opts...)
}

func (c *Client) fnBatchPublish(ctx context.Context, ps []client.Message, opts ...client.PublishOption) error {
	return c.publish(ctx, ps, opts...)
}

func (c *Client) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	return c.funcPublish(ctx, p, opts...)
}

func (c *Client) fnPublish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	return c.publish(ctx, []client.Message{p}, opts...)
}

func (c *Client) publish(ctx context.Context, ps []client.Message, opts ...client.PublishOption) error {
	var body []byte

	options := client.NewPublishOptions(opts...)

	// get proxy
	exchange := ""
	if v, ok := os.LookupEnv("MICRO_PROXY"); ok {
		exchange = v
	}
	// get the exchange
	if len(options.Exchange) > 0 {
		exchange = options.Exchange
	}

	msgs := make([]*broker.Message, 0, len(ps))

	omd, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		omd = metadata.New(2)
	}

	for _, p := range ps {
		md := metadata.Copy(omd)
		topic := p.Topic()
		if len(exchange) > 0 {
			topic = exchange
		}
		md.Set(metadata.HeaderTopic, topic)
		iter := p.Metadata().Iterator()
		var k, v string
		for iter.Next(&k, &v) {
			md.Set(k, v)
		}

		md[metadata.HeaderContentType] = p.ContentType()

		// passed in raw data
		if d, ok := p.Payload().(*codec.Frame); ok {
			body = d.Data
		} else {
			// use codec for payload
			cf, err := c.newCodec(p.ContentType())
			if err != nil {
				return errors.InternalServerError("go.micro.client", "%+v", err)
			}
			// set the body
			b, err := cf.Marshal(p.Payload())
			if err != nil {
				return errors.InternalServerError("go.micro.client", "%+v", err)
			}
			body = b
		}
		msgs = append(msgs, &broker.Message{Header: md, Body: body})
	}

	return c.opts.Broker.BatchPublish(ctx, msgs,
		broker.PublishContext(options.Context),
		broker.PublishBodyOnly(options.BodyOnly),
	)
}

func (c *Client) NewMessage(topic string, msg interface{}, opts ...client.MessageOption) client.Message {
	return newHTTPEvent(topic, msg, c.opts.ContentType, opts...)
}
