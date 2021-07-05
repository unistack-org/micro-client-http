package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"

	"github.com/unistack-org/micro/v3/client"
	"github.com/unistack-org/micro/v3/errors"
	rutil "github.com/unistack-org/micro/v3/util/reflect"
	util "github.com/unistack-org/micro/v3/util/router"
)

var (
	templateCache = make(map[string]util.Template)
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

func newPathRequest(path string, method string, body string, msg interface{}, tags []string) (string, interface{}, error) {
	// parse via https://github.com/googleapis/googleapis/blob/master/google/api/http.proto definition
	tpl, err := newTemplate(path)
	if err != nil {
		return "", nil, err
	}

	if len(tpl.Fields) > 0 && msg == nil {
		return "", nil, fmt.Errorf("nil message but path params requested: %v", path)
	}

	fieldsmapskip := make(map[string]struct{})
	fieldsmap := make(map[string]string, len(tpl.Fields))
	for _, v := range tpl.Fields {
		fieldsmap[v] = ""
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

		if t.name == "" {
			// fallback to lowercase
			t.name = strings.ToLower(fld.Name)
		}

		if !val.IsValid() || val.IsZero() {
			continue
		}

		// nolint: gocritic
		if _, ok := fieldsmap[t.name]; ok {
			switch val.Type().Kind() {
			case reflect.Slice:
				for idx := 0; idx < val.Len(); idx++ {
					values.Add(t.name, fmt.Sprintf("%v", val.Index(idx).Interface()))
				}
				fieldsmapskip[t.name] = struct{}{}
			default:
				fieldsmap[t.name] = fmt.Sprintf("%v", val.Interface())
			}
		} else if (body == "*" || body == t.name) && method != http.MethodGet {
			tnmsg.Field(i).Set(val)
		} else {
			if val.Type().Kind() == reflect.Slice {
				for idx := 0; idx < val.Len(); idx++ {
					values.Add(t.name, fmt.Sprintf("%v", val.Index(idx).Interface()))
				}
			} else {
				values.Add(t.name, fmt.Sprintf("%v", val.Interface()))
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
	for _, fld := range tpl.Pool {
		_, _ = b.WriteRune('/')
		if v, ok := fieldsmap[fld]; ok {
			if v != "" {
				_, _ = b.WriteString(v)
			}
		} else {
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

func newTemplate(path string) (util.Template, error) {
	mu.RLock()
	tpl, ok := templateCache[path]
	if ok {
		mu.RUnlock()
		return tpl, nil
	}
	mu.RUnlock()

	rule, err := util.Parse(path)
	if err != nil {
		return tpl, err
	}

	tpl = rule.Compile()
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
		if cerr != nil {
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
			if rerr != nil {
				return errors.InternalServerError("go.micro.client", rerr.Error())
			}
			return errors.New("go.micro.client", string(buf), int32(hrsp.StatusCode))
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
