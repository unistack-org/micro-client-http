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

	"go.unistack.org/micro/v3/client"
	"go.unistack.org/micro/v3/codec"
	"go.unistack.org/micro/v3/logger"
	"go.unistack.org/micro/v3/metadata"
	rutil "go.unistack.org/micro/v3/util/reflect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"go.unistack.org/micro-client-http/v3/builder"
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
	var (
		method  = http.MethodPost
		bodyOpt string

		parameters = map[string]map[string]string{}

		resolvedPath string
		body         []byte
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

	if protoMsg, ok := msg.(proto.Message); ok {
		reqBuilder, err := builder.NewRequestBuilder(path, method, bodyOpt, protoMsg)
		if err != nil {
			return nil, fmt.Errorf("new request builder: %w", err)
		}

		rp, newMsg, err := reqBuilder.Build()
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		resolvedPath = rp

		b, err := marshallMsg(cf, newMsg)
		if err != nil {
			return nil, fmt.Errorf("marshal msg: %w", err)
		}
		body = b
	} else {
		var err error
		resolvedPath, err = resolveStructPath(path, msg)
		if err != nil {
			return nil, fmt.Errorf("resolve struct path: %w", err)
		}
		if msg != nil {
			b, err := cf.Marshal(msg)
			if err != nil {
				return nil, fmt.Errorf("marshal msg: %w", err)
			}
			body = b
		}
	}

	resolvedURL := joinURL(addr, resolvedPath)

	u, err := normalizeURL(resolvedURL)
	if err != nil {
		return nil, fmt.Errorf("normalize url: %w", err)
	}

	var hreq *http.Request
	if len(body) > 0 {
		hreq, err = http.NewRequestWithContext(ctx, method, u.String(), io.NopCloser(bytes.NewBuffer(body)))
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
		if shouldLogBody(ct) {
			log.Debug(
				ctx,
				fmt.Sprintf(
					"micro.client http request: method=%s url=%s headers=%v body=%s",
					method, u.String(), hreq.Header, body,
				),
			)
		} else {
			log.Debug(
				ctx,
				fmt.Sprintf(
					"micro.client http request: method=%s url=%s headers=%v",
					method, u.String(), hreq.Header,
				),
			)
		}
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
			r.Header[k] = append(r.Header[k], v)
		}
	}

	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		for k, v := range md {
			if k == "Cookie" {
				applyCookies(r, v)
				continue
			}
			r.Header[k] = append(r.Header[k], v)
		}
	}
}

func applyCookies(r *http.Request, rawCookies string) {
	if len(rawCookies) == 0 {
		return
	}

	tmp := http.Request{Header: http.Header{}}
	tmp.Header.Set("Cookie", rawCookies)

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

func resolveStructPath(path string, msg any) (string, error) {
	if !strings.Contains(path, "{") {
		return path, nil
	}
	if msg == nil {
		return path, nil
	}

	result := path
	for start := strings.Index(result, "{"); start != -1; start = strings.Index(result, "{") {
		end := strings.Index(result[start:], "}")
		if end == -1 {
			break
		}
		end += start
		placeholder := result[start+1 : end]

		// support nested paths: {user.id}
		var fieldVal any
		cur := msg
		for _, part := range strings.Split(placeholder, ".") {
			var err error
			_, fieldVal, err = rutil.StructFieldNameByTag(cur, "json", part)
			if err != nil {
				return "", fmt.Errorf("struct has no field for path placeholder %q: %w", placeholder, err)
			}
			cur = fieldVal
		}

		if rutil.IsZero(fieldVal) {
			return "", fmt.Errorf("path placeholder %q has zero value", placeholder)
		}

		result = result[:start] + fmt.Sprintf("%v", fieldVal) + result[end+1:]
	}

	return result, nil
}

func shouldLogBody(contentType string) bool {
	ct := strings.ToLower(strings.Split(contentType, ";")[0])
	switch {
	case strings.HasPrefix(ct, "text/"): // => text/html, text/plain, text/csv etc.
		return true
	case ct == "application/json",
		ct == "application/xml",
		ct == "application/x-www-form-urlencoded",
		ct == "application/yaml":
		return true
	default:
		return false
	}
}
