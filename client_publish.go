package http

import (
	"context"
	"os"

	"go.unistack.org/micro/v3/broker"
	"go.unistack.org/micro/v3/client"
	"go.unistack.org/micro/v3/codec"
	"go.unistack.org/micro/v3/errors"
	"go.unistack.org/micro/v3/metadata"
)

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

func (c *Client) fnPublish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	return c.publish(ctx, []client.Message{p}, opts...)
}

func (c *Client) fnBatchPublish(ctx context.Context, ps []client.Message, opts ...client.PublishOption) error {
	return c.publish(ctx, ps, opts...)
}
