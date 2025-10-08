package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.unistack.org/micro/v4/client"
	"go.unistack.org/micro/v4/codec"
	"go.unistack.org/micro/v4/logger"
	"go.unistack.org/micro/v4/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"go.unistack.org/micro-client-http/v4/builder"
)

func buildHTTPRequest(
	ctx context.Context,
	addr string,
	path string,
	ct string,
	cf codec.Codec,
	msg any,
	opts client.CallOptions,
	log logger.Logger,
) (
	*http.Request,
	error,
) {
	protoMsg, ok := msg.(proto.Message)
	if !ok {
		return nil, errors.New("msg must be a proto message type")
	}

	var (
		method  = http.MethodPost
		bodyOpt string

		parameters = map[string]map[string]string{}
	)

	if opts.Context != nil {
		if v, ok := methodFromOpts(opts); ok {
			method = v
		}
		if v, ok := pathFromOpts(opts); ok {
			path = v
		}
		if v, ok := bodyFromOpts(opts); ok {
			bodyOpt = v
		}
		if h, ok := headerFromOpts(opts); ok && len(h) > 0 {
			m, ok := parameters["header"]
			if !ok {
				m = make(map[string]string)
				parameters["header"] = m
			}
			for idx := 0; idx+1 < len(h); idx += 2 {
				m[h[idx]] = h[idx+1]
			}
		}
		if c, ok := cookieFromOpts(opts); ok && len(c) > 0 {
			m, ok := parameters["cookie"]
			if !ok {
				m = make(map[string]string)
				parameters["cookie"] = m
			}
			for idx := 0; idx+1 < len(c); idx += 2 {
				m[c[idx]] = c[idx+1]
			}
		}
	}

	reqBuilder, err := builder.NewRequestBuilder(path, method, bodyOpt, protoMsg)
	if err != nil {
		return nil, fmt.Errorf("new request builder: %w", err)
	}

	resolvedPath, newMsg, err := reqBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resolvedURL := joinURL(addr, resolvedPath)

	u, err := normalizeURL(resolvedURL)
	if err != nil {
		return nil, fmt.Errorf("normalize url: %w", err)
	}

	body, err := marshallMsg(cf, newMsg)
	if err != nil {
		return nil, fmt.Errorf("marshal msg: %w", err)
	}

	var hreq *http.Request

	if len(body) > 0 {
		hreq, err = http.NewRequestWithContext(ctx, method, u.String(), io.NopCloser(bytes.NewBuffer(body)))
		hreq.ContentLength = int64(len(body))
	} else {
		hreq, err = http.NewRequestWithContext(ctx, method, u.String(), nil)
	}

	if err != nil {
		return nil, fmt.Errorf("new http request: %w", err)
	}

	setHeadersAndCookies(ctx, hreq, ct, opts)
	if err = validateHeadersAndCookies(hreq, parameters); err != nil {
		return nil, fmt.Errorf("validate headers and cookies: %w", err)
	}

	if log.V(logger.DebugLevel) {
		log.Debug(
			ctx,
			fmt.Sprintf("request %s to %s with headers %v body %s", method, u.String(), hreq.Header, body),
		)
	}

	return hreq, nil
}

func joinURL(addr, resolvedPath string) string {
	if addr == "" {
		return resolvedPath
	}
	if resolvedPath == "" {
		return addr
	}

	switch {
	case strings.HasSuffix(addr, "/") && strings.HasPrefix(resolvedPath, "/"):
		return addr + resolvedPath[1:]
	case !strings.HasSuffix(addr, "/") && !strings.HasPrefix(resolvedPath, "/"):
		return addr + "/" + resolvedPath
	default:
		return addr + resolvedPath
	}
}

func normalizeURL(raw string) (*url.URL, error) {
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("invalid scheme: %q (must be http or https)", u.Scheme)
	}

	if u.Host == "" {
		return nil, errors.New("missing host in url")
	}

	return u, nil
}

func marshallMsg(cf codec.Codec, msg proto.Message) ([]byte, error) {
	if msg == nil {
		return nil, nil
	}
	isEmpty := true
	msg.ProtoReflect().Range(func(protoreflect.FieldDescriptor, protoreflect.Value) bool {
		isEmpty = false
		return false
	})
	if isEmpty {
		return nil, nil
	}
	return cf.Marshal(msg)
}

func setHeadersAndCookies(ctx context.Context, r *http.Request, ct string, opts client.CallOptions) {
	r.Header = make(http.Header)

	r.Header.Set(metadata.HeaderContentType, ct)
	r.Header.Set("Content-Length", fmt.Sprintf("%d", r.ContentLength))

	if opts.AuthToken != "" {
		r.Header.Set(metadata.HeaderAuthorization, opts.AuthToken)
	}

	if opts.StreamTimeout > time.Duration(0) {
		r.Header.Set(metadata.HeaderTimeout, fmt.Sprintf("%d", opts.StreamTimeout))
	}
	if opts.RequestTimeout > time.Duration(0) {
		r.Header.Set(metadata.HeaderTimeout, fmt.Sprintf("%d", opts.RequestTimeout))
	}

	if opts.RequestMetadata != nil {
		for k, v := range opts.RequestMetadata {
			if k == "Cookie" {
				applyCookies(r, v)
				continue
			}
			r.Header[k] = append(r.Header[k], v...)
		}
	}

	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		for k, v := range md {
			if k == "Cookie" {
				applyCookies(r, v)
				continue
			}
			r.Header[k] = append(r.Header[k], v...)
		}
	}
}

func applyCookies(r *http.Request, rawCookies []string) {
	if len(rawCookies) == 0 {
		return
	}

	tmp := http.Request{Header: http.Header{}}
	for _, raw := range rawCookies {
		tmp.Header.Add("Cookie", raw)
	}

	for _, c := range tmp.Cookies() {
		r.AddCookie(c)
	}
}

func validateHeadersAndCookies(r *http.Request, parameters map[string]map[string]string) error {
	if headers, ok := parameters["header"]; ok {
		for name, required := range headers {
			if required == "true" && r.Header.Get(name) == "" {
				return fmt.Errorf("missing required header: %s", name)
			}
		}
	}

	if cookies, ok := parameters["cookie"]; ok {
		cookieMap := map[string]string{}
		for _, c := range r.Cookies() {
			cookieMap[c.Name] = c.Value
		}

		for name, required := range cookies {
			if required == "true" {
				if _, ok := cookieMap[name]; !ok {
					return fmt.Errorf("missing required cookie: %s", name)
				}
			}
		}
	}

	return nil
}
