package http

import (
	"net/url"
	"testing"
)

func TestParsing(t *testing.T) {
	type Message struct {
		IIN string `protobuf:"bytes,1,opt,name=iin,proto3" json:"iin"`
	}

	omsg := &Message{IIN: "5555"}

	for _, m := range []string{"POST"} {
		body := ""
		path, nmsg, err := newPathRequest("/users/iin/{iin}/push-notifications", m, body, omsg, []string{"protobuf", "json"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(path)
		if err != nil {
			t.Fatal(err)
		}
		_ = nmsg
		if u.Path != "/users/iin/5555/push-notifications" {
			t.Fatalf("newPathRequest invalid path %s", u.Path)
		}
		if nmsg != nil {
			t.Fatalf("new message must be nil: %v\n", nmsg)
		}
	}
}

func TestNewPathRequest(t *testing.T) {
	type Message struct {
		Name string `json:"name"`
		Val1 string `protobuf:"bytes,1,opt,name=val1,proto3" json:"val1"`
		Val3 []string
		Val2 int64
	}

	omsg := &Message{Name: "test_name", Val1: "test_val1", Val2: 100, Val3: []string{"slice"}}

	for _, m := range []string{"POST", "PUT", "PATCH", "GET", "DELETE"} {
		body := ""
		path, nmsg, err := newPathRequest("/v1/test", m, body, omsg, []string{"protobuf", "json"}, nil)
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

func TestNewPathRequestWithEmptyBody(t *testing.T) {
	val := struct{}{}

	for _, m := range []string{"POST", "PUT", "PATCH", "GET", "DELETE"} {
		body := `{"type": "invalid"}`
		path, nmsg, err := newPathRequest("/v1/test", m, body, val, []string{"protobuf", "json"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if nmsg == nil {
			t.Fatalf("invalid path: nil nmsg")
		}

		u, err := url.Parse(path)
		if err != nil {
			t.Fatal(err)
		}
		vals := u.Query()
		if len(vals) != 0 {
			t.Fatalf("invalid path: %v nmsg: %v", path, nmsg)
		}
	}
}

func TestNewPathVarRequest(t *testing.T) {
	type Message struct {
		Name string `json:"name"`
		Val1 string `protobuf:"bytes,1,opt,name=val1,proto3" json:"val1"`
		Val3 []string
		Val2 int64
	}

	omsg := &Message{Name: "test_name", Val1: "test_val1", Val2: 100, Val3: []string{"slice"}}

	for _, m := range []string{"POST", "PUT", "PATCH", "GET", "DELETE"} {
		body := ""
		if m != "GET" {
			body = "*"
		}
		path, nmsg, err := newPathRequest("/v1/test/{val1}", m, body, omsg, []string{"protobuf", "json"}, nil)
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
