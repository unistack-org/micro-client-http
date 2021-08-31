package http

import (
	"net/url"
	"strings"
	"testing"
)

type Request struct {
	Name   string `json:"name"`
	Field1 string `json:"field1"`
	Field2 string
	Field3 int64
}

func TestPathValues(t *testing.T) {
	req := &Request{Name: "vtolstov", Field1: "field1"}
	p, m, err := newPathRequest("/api/v1/test?Name={name}&Field1={field1}", "POST", "*", req, nil)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	_ = m
	if u.Query().Get("Name") != "vtolstov" || u.Query().Get("Field1") != "field1" {
		t.Fatalf("invalid values %v", u.Query())
	}
}

func TestValidPath(t *testing.T) {
	req := &Request{Name: "vtolstov", Field1: "field1", Field2: "field2", Field3: 10}
	p, m, err := newPathRequest("/api/v1/{name}/list", "GET", "", req, nil)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	_ = m
	parts := strings.Split(u.RawQuery, "&")
	if len(parts) != 3 {
		t.Fatalf("invalid path: %v", parts)
	}
}

func TestInvalidPath(t *testing.T) {
	req := &Request{Name: "vtolstov", Field1: "field1", Field2: "field2", Field3: 10}
	p, m, err := newPathRequest("/api/v1/{xname}/list", "GET", "", req, nil)
	if err == nil {
		t.Fatalf("path param must not be filled")
	}
	_, _ = p, m
}
