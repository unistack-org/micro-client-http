package http

import (
	"go.unistack.org/micro/v4/client"
	"go.unistack.org/micro/v4/codec"
)

type httpRequest struct {
	service string
	method  string
	request any
	opts    client.RequestOptions
}

func (h *httpRequest) Service() string {
	return h.service
}

func (h *httpRequest) Method() string {
	return h.method
}

func (h *httpRequest) Endpoint() string {
	return h.method
}

func (h *httpRequest) ContentType() string {
	return h.opts.ContentType
}

func (h *httpRequest) Body() any {
	return h.request
}

func (h *httpRequest) Codec() codec.Codec {
	return nil
}

func (h *httpRequest) Stream() bool {
	return h.opts.Stream
}
