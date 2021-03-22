package http

import (
	"net/url"
	"testing"
)

func TestTemplate(t *testing.T) {
	tpl, err := newTemplate("/v1/{ClientID}/list")
	if err != nil {
		t.Fatal(err)
	}
	_ = tpl
	//	fmt.Printf("%#+v\n", tpl.Pool)
}

func TestNewPathRequest(t *testing.T) {
	type Message struct {
		Name string `json:"name"`
		Val1 string `protobuf:"bytes,1,opt,name=val1,proto3" json:"val1"`
		Val2 int64
		Val3 []string
	}

	omsg := &Message{Name: "test_name", Val1: "test_val1", Val2: 100, Val3: []string{"slice"}}

	for _, m := range []string{"POST", "PUT", "PATCH", "GET", "DELETE"} {
		body := ""
		path, nmsg, err := newPathRequest("/v1/test", m, body, omsg, []string{"protobuf", "json"})
		if err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(path)
		if err != nil {
			t.Fatal(err)
		}
		vals := u.Query()
		if v, ok := vals["name"]; !ok || v[0] != "test_name" {
			t.Fatalf("invalid path: %v nmsg: %v", path, nmsg)
		}
	}
}

func TestNewPathVarRequest(t *testing.T) {
	type Message struct {
		Name string `json:"name"`
		Val1 string `protobuf:"bytes,1,opt,name=val1,proto3" json:"val1"`
		Val2 int64
		Val3 []string
	}

	omsg := &Message{Name: "test_name", Val1: "test_val1", Val2: 100, Val3: []string{"slice"}}

	for _, m := range []string{"POST", "PUT", "PATCH", "GET", "DELETE"} {
		body := ""
		if m != "GET" {
			body = "*"
		}
		path, nmsg, err := newPathRequest("/v1/test/{val1}", m, body, omsg, []string{"protobuf", "json"})
		if err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(path)
		if err != nil {
			t.Fatal(err)
		}
		if m != "GET" {
			if _, ok := nmsg.(*Message); !ok {
				t.Fatalf("invalid nmsg: %#+v\n", nmsg)
			}
			if nmsg.(*Message).Name != "test_name" {
				t.Fatalf("invalid nmsg: %v nmsg: %v", path, nmsg)
			}
		} else {
			vals := u.Query()
			if v, ok := vals["val2"]; !ok || v[0] != "100" {
				t.Fatalf("invalid path: %v nmsg: %v", path, nmsg)
			}
		}
	}
}
