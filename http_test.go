package http

import (
	"testing"
)

type Request struct {
	Name   string `json:"name"`
	Field1 string
	Field2 string
	Field3 int64
}

func TestValidPath(t *testing.T) {
	req := &Request{Name: "vtolstov", Field1: "field1", Field2: "field2", Field3: 10}
	p, m, err := newPathRequest("/api/v1/{name}/list", "GET", "", req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = p, m
}

func TestInvalidPath(t *testing.T) {
	req := &Request{Name: "vtolstov", Field1: "field1", Field2: "field2", Field3: 10}
	p, m, err := newPathRequest("/api/v1/{xname}/list", "GET", "", req)
	if err == nil {
		t.Fatalf("path param must not be filled")
	}
	_, _ = p, m
}
