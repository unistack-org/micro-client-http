package http

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/unistack-org/micro/v3/client"
	"github.com/unistack-org/micro/v3/codec"
	"github.com/unistack-org/micro/v3/errors"
	rutil "github.com/unistack-org/micro/v3/util/reflect"
	util "github.com/unistack-org/micro/v3/util/router"
)

var (
	templateCache = make(map[string]util.Template)
	mu            sync.RWMutex
)

func newPathRequest(path string, method string, body string, msg interface{}, tags []string) (string, interface{}, error) {
	// parse via https://github.com/googleapis/googleapis/blob/master/google/api/http.proto definition
	tpl, err := newTemplate(path)
	if err != nil {
		return "", nil, err
	}

	if len(tpl.Fields) > 0 && msg == nil {
		return "", nil, fmt.Errorf("nil message but path params requested: %v", path)
	}

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

	values := make(map[string]string)
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
				t = &tag{key: tn, name: tp[3][5:], opts: append(tp[:3], tp[4:]...)}
			default:
				t = &tag{key: tn, name: tp[0], opts: tp[1:]}
			}
			if t.name != "" {
				break
			}
		}

		if t.name == "" {
			// fallback to lowercase
			t.name = strings.ToLower(fld.Name)
		}

		if _, ok := fieldsmap[t.name]; ok {
			fieldsmap[t.name] = fmt.Sprintf("%v", val.Interface())
		} else if (body == "*" || body == t.name) && method != http.MethodGet {
			tnmsg.Field(i).Set(val)
		} else {
			values[t.name] = fmt.Sprintf("%v", val.Interface())
		}
	}

	// check not filled stuff
	for k, v := range fieldsmap {
		if v == "" {
			return "", nil, fmt.Errorf("path param %s not filled", k)
		}
	}

	var b strings.Builder
	for _, fld := range tpl.Pool {
		_, _ = b.WriteRune('/')
		if v, ok := fieldsmap[fld]; ok {
			_, _ = b.WriteString(v)
		} else {
			_, _ = b.WriteString(fld)
		}
	}

	idx := 0
	for k, v := range values {
		if idx == 0 {
			_, _ = b.WriteRune('?')
		} else {
			_, _ = b.WriteRune('&')
		}
		_, _ = b.WriteString(k)
		_, _ = b.WriteRune('=')
		_, _ = b.WriteString(v)
		idx++
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

func parseRsp(ctx context.Context, hrsp *http.Response, cf codec.Codec, rsp interface{}, opts client.CallOptions) error {
	b, err := ioutil.ReadAll(hrsp.Body)
	if err != nil {
		return errors.InternalServerError("go.micro.client", err.Error())
	}

	if hrsp.StatusCode < 400 {
		// unmarshal
		if err := cf.Unmarshal(b, rsp); err != nil {
			return errors.InternalServerError("go.micro.client", err.Error())
		}
		return nil
	}

	errmap, ok := opts.Context.Value(errorMapKey{}).(map[string]interface{})
	if !ok || errmap == nil {
		// user not provide map of errors
		// id: req.Service() ??
		return errors.New("go.micro.client", string(b), int32(hrsp.StatusCode))
	}

	if err, ok = errmap[fmt.Sprintf("%d", hrsp.StatusCode)].(error); !ok {
		err, ok = errmap["default"].(error)
	}
	if !ok {
		return errors.New("go.micro.client", string(b), int32(hrsp.StatusCode))
	}

	if cerr := cf.Unmarshal(b, err); cerr != nil {
		return errors.InternalServerError("go.micro.client", cerr.Error())
	}

	return err
}

type tag struct {
	key  string
	name string
	opts []string
}
