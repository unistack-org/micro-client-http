package http

import (
	"go.unistack.org/micro/v3/client"
	"go.unistack.org/micro/v3/metadata"
)

type httpEvent struct {
	payload     interface{}
	topic       string
	contentType string
	opts        client.MessageOptions
}

func newHTTPEvent(topic string, payload interface{}, contentType string, opts ...client.MessageOption) client.Message {
	options := client.NewMessageOptions(opts...)

	if len(options.ContentType) > 0 {
		contentType = options.ContentType
	}

	return &httpEvent{
		payload:     payload,
		topic:       topic,
		contentType: contentType,
		opts:        options,
	}
}

func (c *httpEvent) ContentType() string {
	return c.contentType
}

func (c *httpEvent) Topic() string {
	return c.topic
}

func (c *httpEvent) Payload() interface{} {
	return c.payload
}

func (c *httpEvent) Metadata() metadata.Metadata {
	return c.opts.Metadata
}
