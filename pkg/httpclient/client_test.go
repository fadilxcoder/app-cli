package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDo_DecodesJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("missing Accept header")
		}
		if r.Header.Get("X-Test") != "yes" {
			t.Errorf("custom header not propagated")
		}
		_, _ = io.WriteString(w, `{"ok":true,"name":"alice"}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	var out struct {
		OK   bool   `json:"ok"`
		Name string `json:"name"`
	}
	err := c.Do(context.Background(), "GET", "/", map[string]string{"X-Test": "yes"}, nil, &out)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if !out.OK || out.Name != "alice" {
		t.Fatalf("unexpected payload: %+v", out)
	}
}

func TestDo_MarshalsBodyAndReturnsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(buf), `"email":"a@b.c"`) {
			t.Errorf("body not marshalled, got %s", string(buf))
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":"bad creds"}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	body := map[string]string{"email": "a@b.c", "password": "x"}
	err := c.Do(context.Background(), "POST", "/x", nil, body, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var he *Error
	if !errors.As(err, &he) {
		t.Fatalf("error is not *Error: %T", err)
	}
	if he.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", he.StatusCode)
	}
}

func TestDo_NilOutSkipsDecode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"foo": "bar"})
	}))
	defer srv.Close()

	c := New(srv.URL)
	if err := c.Do(context.Background(), "GET", "/", nil, nil, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
}
