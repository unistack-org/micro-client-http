package http

import (
	"go.unistack.org/micro/v4/client"
	"go.unistack.org/micro/v4/metadata"
)

type httpMessage struct {
	payload     interface{}
	topic       string
	contentType string
	opts        client.MessageOptions
}

func newHTTPMessage(topic string, payload interface{}, contentType string, opts ...client.MessageOption) client.Message {
	options := client.NewMessageOptions(opts...)

	if len(options.ContentType) > 0 {
		contentType = options.ContentType
	}

	return &httpMessage{
		payload:     payload,
		topic:       topic,
		contentType: contentType,
		opts:        options,
	}
}

func (h *httpMessage) ContentType() string {
	return h.contentType
}

func (h *httpMessage) Topic() string {
	return h.topic
}

func (h *httpMessage) Payload() interface{} {
	return h.payload
}

func (h *httpMessage) Metadata() metadata.Metadata {
	return h.opts.Metadata
}
