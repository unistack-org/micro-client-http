package http

import (
	"net/url"
	"strings"
	"testing"
)

type Request struct {
	Name     string `json:"name"`
	Field1   string `json:"field1"`
	ClientID string
	Field2   string
	Field3   int64
}

func TestPathWithHeader(t *testing.T) {
	req := &Request{Name: "vtolstov", Field1: "field1", ClientID: "1234567890"}
	p, m, err := newPathRequest(
		"/api/v1/test?Name={name}&Field1={field1}",
		"POST",
		"*",
		req,
		nil,
		map[string]map[string]string{"header": {"ClientID": "true"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	if m != nil {
		t.Fatal("new struct must be nil")
	}
	if u.Query().Get("Name") != "vtolstov" || u.Query().Get("Field1") != "field1" {
		t.Fatalf("invalid values %v", u.Query())
	}
}

func TestPathValues(t *testing.T) {
	req := &Request{Name: "vtolstov", Field1: "field1"}
	p, m, err := newPathRequest("/api/v1/test?Name={name}&Field1={field1}", "POST", "*", req, nil, nil)
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
	p, m, err := newPathRequest("/api/v1/{name}/list", "GET", "", req, nil, nil)
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
	_, _, err := newPathRequest("/api/v1/{xname}/list", "GET", "", req, nil, nil)
	if err == nil {
		t.Fatal("path param must not be filled")
	}
}
