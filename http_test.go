package http

import (
	"net/url"
	"strings"
	"testing"
)

type request struct {
	NestedTest *request `json:"nested_test"`
	Name       string   `json:"name"`
	Field1     string   `json:"field1"`
	ClientID   string
	Field2     string
	Field3     int64
}

func TestNestedPath(t *testing.T) {
	req := &request{Name: "first", NestedTest: &request{Name: "second"}, Field1: "fieldval"}
	p, m, err := newPathRequest("/api/v1/{name}/{nested_test.name}", "PUT", "*", req, []string{"json", "protobuf"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(p)
	if err != nil {
		t.Fatal(err)
	}
	if s := u.String(); s != "/api/v1/first/second?field1=fieldval" {
		t.Fatalf("nested path error %s", s)
	}
	_ = m
}

func TestPathWithHeader(t *testing.T) {
	req := &request{Name: "vtolstov", Field1: "field1", ClientID: "1234567890"}
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
	req := &request{Name: "vtolstov", Field1: "field1"}
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
	req := &request{Name: "vtolstov", Field1: "field1", Field2: "field2", Field3: 10}
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
	req := &request{Name: "vtolstov", Field1: "field1", Field2: "field2", Field3: 10}
	s, _, err := newPathRequest("/api/v1/{xname}/list", "GET", "", req, nil, nil)
	if err == nil {
		t.Fatalf("path param must not be filled: %s", s)
	}
}
