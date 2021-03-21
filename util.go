package http

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"

	rutil "github.com/unistack-org/micro/v3/util/reflect"
	util "github.com/unistack-org/micro/v3/util/router"
)

var (
	templateCache = make(map[string]util.Template)
	mu            sync.RWMutex
)

func newPathRequest(path string, method string, body string, msg interface{}) (string, interface{}, error) {
	// parse via https://github.com/googleapis/googleapis/blob/master/google/api/http.proto definition
	tpl, err := newTemplate(path)
	if err != nil {
		return "", nil, err
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
		lfield := strings.ToLower(fld.Name)
		if _, ok := fieldsmap[lfield]; ok {
			fieldsmap[lfield] = fmt.Sprintf("%v", val.Interface())
		} else if (body == "*" || body == lfield) && method != http.MethodGet {
			tnmsg.Field(i).Set(val)
		} else {
			values[lfield] = fmt.Sprintf("%v", val.Interface())
		}
	}

	// check not filled stuff
	for k, v := range fieldsmap {
		if v == "" {
			return "", nil, fmt.Errorf("path param %s not filled %s", k, v)
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
