package http

import (
	"go.unistack.org/micro/v3/client"
)

type httpMessage struct {
	payload     interface{}
	topic       string
	contentType string
}

func newHTTPMessage(topic string, payload interface{}, contentType string, opts ...client.MessageOption) client.Message {
	options := client.NewMessageOptions(opts...)
	if len(options.ContentType) == 0 {
		options.ContentType = contentType
	}

	return &httpMessage{
		payload:     payload,
		topic:       topic,
		contentType: options.ContentType,
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
