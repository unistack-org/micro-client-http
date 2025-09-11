// Package builder implements google.api.http-style request building (gRPC JSON transcoding)
// for HTTP requests, closely following the google.api.http spec.
// See full spec for details: https://github.com/googleapis/googleapis/blob/master/google/api/http.proto
package builder

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type RequestBuilder struct {
	path       string        // e.g. "/v1/{name=projects/*/topics/*}:publish" or "/users/{user.id}"
	method     string        // GET, POST, PATCH, etc. (not used in mapping rules, but convenient for callers)
	bodyOption bodyOption    // "", "*", or top-level field name
	msg        proto.Message // request struct
}

func NewRequestBuilder(
	path string,
	method string,
	bodyOpt string,
	msg proto.Message,
) (
	*RequestBuilder,
	error,
) {
	rb := &RequestBuilder{
		path:       path,
		method:     method,
		bodyOption: bodyOption(bodyOpt),
		msg:        msg,
	}

	if err := rb.validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	return rb, nil
}

// Build applies mapping rules and returns:
//
//	resolvedPath — path with placeholders substituted and query appended
//	newMsg       — same concrete type as input, filtered to contain only the body fields
//	err          — if mapping/validation failed
func (b *RequestBuilder) Build() (resolvedPath string, newMsg proto.Message, err error) {
	tmpl, isCached := getCachedPathTemplate(b.path)
	if !isCached {
		tmpl, err = parsePathTemplate(b.path)
		if err != nil {
			return "", nil, fmt.Errorf("parse path template: %w", err)
		}
		setPathTemplateCache(b.path, tmpl)
	}

	var usedFieldsPath *usedFields
	resolvedPath, usedFieldsPath, err = resolvePathPlaceholders(tmpl, b.msg)
	if err != nil {
		return "", nil, fmt.Errorf("resolve path placeholders: %w", err)
	}

	// if all set fields are already used in path, no need to process query/body
	if allFieldsUsed(b.msg, usedFieldsPath) {
		return resolvedPath, initZeroMsg(b.msg), nil
	}

	switch {
	case b.bodyOption.isWithoutBody():
		var query url.Values
		query, err = buildQuery(b.msg, usedFieldsPath, "")
		if err != nil {
			return "", nil, fmt.Errorf("build query: %w", err)
		}

		return resolvedPath + encodeQuery(query), initZeroMsg(b.msg), nil

	case b.bodyOption.isSingleField():
		fieldBody := b.bodyOption.String()

		newMsg, err = buildSingleFieldBody(b.msg, fieldBody)
		if err != nil {
			return "", nil, fmt.Errorf("build single field body: %w", err)
		}

		var query url.Values
		query, err = buildQuery(b.msg, usedFieldsPath, fieldBody)
		if err != nil {
			return "", nil, fmt.Errorf("build query: %w", err)
		}

		return resolvedPath + encodeQuery(query), newMsg, nil

	case b.bodyOption.isFullBody():
		newMsg, err = buildFullBody(b.msg, usedFieldsPath)
		if err != nil {
			return "", nil, fmt.Errorf("build full body: %w", err)
		}

		return resolvedPath, newMsg, nil

	default:
		return "", nil, fmt.Errorf("unsupported body option %s", b.bodyOption.String())
	}
}

func (b *RequestBuilder) validate() error {
	if b.path == "" {
		return errors.New("path is empty")
	}
	if err := validateHTTPMethod(b.method); err != nil {
		return fmt.Errorf("validate http method: %w", err)
	}
	if err := validateHTTPMethodAndBody(b.method, b.bodyOption); err != nil {
		return fmt.Errorf("validate http method and body: %w", err)
	}
	if b.msg == nil {
		return errors.New("msg is nil")
	}
	return nil
}

func validateHTTPMethod(method string) error {
	switch strings.ToUpper(method) {
	case http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace:
		return nil
	default:
		return errors.New("invalid http method")
	}
}

func validateHTTPMethodAndBody(method string, bodyOpt bodyOption) error {
	switch method {
	case http.MethodGet, http.MethodDelete, http.MethodHead, http.MethodOptions:
		if !bodyOpt.isWithoutBody() {
			return fmt.Errorf("%s method must not have a body", method)
		}
	}
	return nil
}

func allFieldsUsed(msg proto.Message, used *usedFields) bool {
	if used.len() == 0 {
		return false
	}
	count := 0
	msg.ProtoReflect().Range(func(protoreflect.FieldDescriptor, protoreflect.Value) bool {
		count++
		return true
	})
	return used.len() == count
}

func encodeQuery(query url.Values) string {
	if len(query) == 0 {
		return ""
	}
	enc := query.Encode()
	if enc == "" {
		return ""
	}
	return "?" + enc
}

func initZeroMsg(msg proto.Message) proto.Message {
	return msg.ProtoReflect().New().Interface()
}
