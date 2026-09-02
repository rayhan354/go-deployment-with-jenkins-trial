package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler(rr, req)

	expectedPrefix := "Hello, DevOps! version="
	got := rr.Body.String()
	if len(got) < len(expectedPrefix) || got[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("handler returned unexpected body: got %q, want prefix %q", got, expectedPrefix)
	}
}
