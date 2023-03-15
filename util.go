package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"

	"go.unistack.org/micro/v3/client"
	"go.unistack.org/micro/v3/errors"
	"go.unistack.org/micro/v3/logger"
	rutil "go.unistack.org/micro/v3/util/reflect"
)

var (
	templateCache = make(map[string][]string)
	mu            sync.RWMutex
)

// Error struct holds error
type Error struct {
	err interface{}
}

// Error func for error interface
func (err *Error) Error() string {
	return fmt.Sprintf("%v", err.err)
}

func GetError(err error) interface{} {
	if rerr, ok := err.(*Error); ok {
		return rerr.err
	}
	return err
}

func newPathRequest(path string, method string, body string, msg interface{}, tags []string, parameters map[string]map[string]string) (string, interface{}, error) {
	// parse via https://github.com/googleapis/googleapis/blob/master/google/api/http.proto definition
	tpl, err := newTemplate(path)
	if err != nil {
		return "", nil, err
	}

	if len(tpl) > 0 && msg == nil {
		return "", nil, fmt.Errorf("nil message but path params requested: %v", path)
	}

	fieldsmapskip := make(map[string]struct{})
	fieldsmap := make(map[string]string, len(tpl))
	for _, v := range tpl {
		var vs, ve int
		for i := 0; i < len(v); i++ {
			switch v[i] {
			case '{':
				vs = i + 1
			case '}':
				ve = i
			}
			if ve != 0 {
				fieldsmap[v[vs:ve]] = ""
				vs = 0
				ve = 0
			}
		}
	}

	nmsg, err := rutil.Zero(msg)
	if err != nil {
		return "", nil, err
	}

	// we cant switch on message and use proto helpers, to avoid dependency to protobuf
	tmsg := reflect.ValueOf(msg)
	if tmsg.Kind() == reflect.Ptr {
		tmsg = tmsg.Elem()
	}

	tnmsg := reflect.ValueOf(nmsg)
	if tnmsg.Kind() == reflect.Ptr {
		tnmsg = tnmsg.Elem()
	}

	values := url.Values{}
	// copy cycle
	for i := 0; i < tmsg.NumField(); i++ {
		val := tmsg.Field(i)
		if val.IsZero() {
			continue
		}
		fld := tmsg.Type().Field(i)
		// Skip unexported fields.
		if fld.PkgPath != "" {
			continue
		}
		/* check for empty PkgPath can be replaced with new method IsExported
		if !fld.IsExported() {
			continue
		}
		*/
		t := &tag{}
		for _, tn := range tags {
			ts, ok := fld.Tag.Lookup(tn)
			if !ok {
				continue
			}

			tp := strings.Split(ts, ",")
			// special
			switch tn {
			case "protobuf": // special
				for _, p := range tp {
					if idx := strings.Index(p, "name="); idx > 0 {
						t = &tag{key: tn, name: p[idx:]}
					}
				}
			default:
				t = &tag{key: tn, name: tp[0]}
			}
			if t.name != "" {
				break
			}
		}

		cname := t.name
		if cname == "" {
			cname = fld.Name
			// fallback to lowercase
			t.name = strings.ToLower(fld.Name)
		}
		if _, ok := parameters["header"][cname]; ok {
			continue
		}
		if _, ok := parameters["cookie"][cname]; ok {
			continue
		}

		if !val.IsValid() || val.IsZero() {
			continue
		}

		// nolint: gocritic, nestif
		if _, ok := fieldsmap[t.name]; ok {
			switch val.Type().Kind() {
			case reflect.Slice:
				for idx := 0; idx < val.Len(); idx++ {
					values.Add(t.name, getParam(val.Index(idx)))
				}
				fieldsmapskip[t.name] = struct{}{}
			default:
				fieldsmap[t.name] = getParam(val)
			}
		} else if (body == "*" || body == t.name) && method != http.MethodGet {
			if tnmsg.Field(i).CanSet() {
				tnmsg.Field(i).Set(val)
			}
		} else {
			if val.Type().Kind() == reflect.Slice {
				for idx := 0; idx < val.Len(); idx++ {
					values.Add(t.name, getParam(val.Index(idx)))
				}
			} else {
				values.Add(t.name, getParam(val))
			}
		}
	}

	// check not filled stuff
	for k, v := range fieldsmap {
		_, ok := fieldsmapskip[k]
		if !ok && v == "" {
			return "", nil, fmt.Errorf("path param %s not filled", k)
		}
	}

	var b strings.Builder

	for _, fld := range tpl {
		_, _ = b.WriteRune('/')
		// nolint: nestif
		var vs, ve, vf int
		var pholder bool
		for i := 0; i < len(fld); i++ {
			switch fld[i] {
			case '{':
				vs = i + 1
			case '}':
				ve = i
			}
			// nolint: nestif
			if vs > 0 && ve != 0 {
				if vm, ok := fieldsmap[fld[vs:ve]]; ok {
					if vm != "" {
						_, _ = b.WriteString(fld[vf : vs-1])
						_, _ = b.WriteString(vm)
						vf = ve + 1
					}
				} else {
					_, _ = b.WriteString(fld)
				}
				vs = 0
				ve = 0
				pholder = true
			}
		}
		if !pholder {
			_, _ = b.WriteString(fld)
		}
	}

	if len(values) > 0 {
		_, _ = b.WriteRune('?')
		_, _ = b.WriteString(values.Encode())
	}

	if rutil.IsZero(nmsg) {
		return b.String(), nil, nil
	}

	return b.String(), nmsg, nil
}

func newTemplate(path string) ([]string, error) {
	if len(path) == 0 || path[0] != '/' {
		return nil, fmt.Errorf("path must starts with /")
	}
	mu.RLock()
	tpl, ok := templateCache[path]
	if ok {
		mu.RUnlock()
		return tpl, nil
	}
	mu.RUnlock()

	tpl = strings.Split(path[1:], "/")
	mu.Lock()
	templateCache[path] = tpl
	mu.Unlock()

	return tpl, nil
}

func (h *httpClient) parseRsp(ctx context.Context, hrsp *http.Response, rsp interface{}, opts client.CallOptions) error {
	var err error

	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		// fast path return
		if hrsp.StatusCode == http.StatusNoContent {
			return nil
		}
		ct := DefaultContentType

		if htype := hrsp.Header.Get("Content-Type"); htype != "" {
			ct = htype
		}

		cf, cerr := h.newCodec(ct)
		if hrsp.StatusCode >= 400 && cerr != nil {
			var buf []byte
			if hrsp.Body != nil {
				buf, err = io.ReadAll(hrsp.Body)
				if err != nil && h.opts.Logger.V(logger.ErrorLevel) {
					h.opts.Logger.Errorf(ctx, "failed to read body: %v", err)
				}
			}
			if h.opts.Logger.V(logger.DebugLevel) {
				h.opts.Logger.Debugf(ctx, "response %s with %v", buf, hrsp.Header)
			}
			// response like text/plain or something else, return original error
			return errors.New("go.micro.client", string(buf), int32(hrsp.StatusCode))
		} else if cerr != nil {
			if h.opts.Logger.V(logger.DebugLevel) {
				h.opts.Logger.Debugf(ctx, "response with %v unknown content-type", hrsp.Header, ct)
			}
			return errors.InternalServerError("go.micro.client", cerr.Error())
		}

		// succeseful response
		if hrsp.StatusCode < 400 {
			if err = cf.ReadBody(hrsp.Body, rsp); err != nil {
				return errors.InternalServerError("go.micro.client", err.Error())
			}
			return nil
		}

		// response with error
		var rerr interface{}
		errmap, ok := opts.Context.Value(errorMapKey{}).(map[string]interface{})
		if ok && errmap != nil {
			rerr, ok = errmap[fmt.Sprintf("%d", hrsp.StatusCode)]
			if !ok {
				rerr, ok = errmap["default"]
			}
		}

		if !ok || rerr == nil {
			buf, rerr := io.ReadAll(hrsp.Body)
			if h.opts.Logger.V(logger.DebugLevel) {
				h.opts.Logger.Debugf(ctx, "response %s with %v", buf, hrsp.Header)
			}
			if rerr != nil {
				return errors.InternalServerError("go.micro.client", rerr.Error())
			}
			return errors.New("go.micro.client", string(buf), int32(hrsp.StatusCode))
		}

		if h.opts.Logger.V(logger.DebugLevel) {
			buf, rerr := io.ReadAll(hrsp.Body)
			h.opts.Logger.Debugf(ctx, "response %s with %v", buf, hrsp.Header)
			if err != nil {
				return errors.InternalServerError("go.micro.client", rerr.Error())
			}
			hrsp.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
		}

		if cerr := cf.ReadBody(hrsp.Body, rerr); cerr != nil {
			return errors.InternalServerError("go.micro.client", cerr.Error())
		}

		if err, ok = rerr.(error); !ok {
			err = &Error{rerr}
		}

	}

	return err
}

type tag struct {
	key  string
	name string
}

func getParam(val reflect.Value) string {
	var v string
	switch val.Kind() {
	case reflect.Ptr:
		switch reflect.Indirect(val).Type().String() {
		case
			"wrapperspb.BoolValue",
			"wrapperspb.BytesValue",
			"wrapperspb.DoubleValue",
			"wrapperspb.FloatValue",
			"wrapperspb.Int32Value", "wrapperspb.Int64Value",
			"wrapperspb.StringValue",
			"wrapperspb.UInt32Value", "wrapperspb.UInt64Value":
			if eva := reflect.Indirect(val).FieldByName("Value"); eva.IsValid() {
				v = getParam(eva)
			}
		}
	default:
		v = fmt.Sprintf("%v", val.Interface())
	}
	return v
}
