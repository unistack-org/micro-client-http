package http

import (
	"go.unistack.org/micro/v4/client"
	"go.unistack.org/micro/v4/codec"
)

type httpRequest struct {
	service     string
	method      string
	contentType string
	request     interface{}
	opts        client.RequestOptions
}

func newHTTPRequest(service, method string, request interface{}, contentType string, opts ...client.RequestOption) client.Request {
	options := client.NewRequestOptions(opts...)
	if len(options.ContentType) == 0 {
		options.ContentType = contentType
	}

	return &httpRequest{
		service:     service,
		method:      method,
		request:     request,
		contentType: options.ContentType,
		opts:        options,
	}
}

func (h *httpRequest) ContentType() string {
	return h.contentType
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

func (h *httpRequest) Codec() codec.Codec {
	return nil
}

func (h *httpRequest) Body() interface{} {
	return h.request
}

func (h *httpRequest) Stream() bool {
	return h.opts.Stream
}
