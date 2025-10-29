package http

import (
	"go.unistack.org/micro/v3/client"
	"go.unistack.org/micro/v3/metadata"
)

type httpMessage struct {
	topic   string
	payload interface{}
	opts    client.MessageOptions
}

func (m *httpMessage) Topic() string {
	return m.topic
}

func (m *httpMessage) Payload() interface{} {
	return m.payload
}

func (m *httpMessage) ContentType() string {
	return m.opts.ContentType
}

func (m *httpMessage) Metadata() metadata.Metadata {
	return m.opts.Metadata
}
