package http

import (
	"net/url"
	"testing"
)

func TestNewPathRequest(t *testing.T) {
	type Message struct {
		Name string
		Val1 string
		Val2 int64
		Val3 []string
	}

	omsg := &Message{Name: "test_name", Val1: "test_val1", Val2: 100, Val3: []string{"slice"}}

	for _, m := range []string{"POST", "PUT", "PATCH", "GET", "DELETE"} {
		path, nmsg, err := newPathRequest("/v1/test", m, "", omsg)
		if err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(path)
		if err != nil {
			t.Fatal(err)
		}
		vals := u.Query()
		if v, ok := vals["name"]; !ok || v[0] != "test_name" {
			t.Fatalf("invlid path: %v nmsg: %v", path, nmsg)
		}
	}
}
