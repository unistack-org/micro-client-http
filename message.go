package http

import (
	"github.com/micro/go-micro/client"
)

type httpMessage struct {
	topic       string
	contentType string
	payload     interface{}
}

func newHTTPMessage(topic string, payload interface{}, contentType string) client.Message {
	return &httpMessage{
		payload:     payload,
		topic:       topic,
		contentType: contentType,
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
