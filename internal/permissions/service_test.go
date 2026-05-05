package permissions

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fadilxcoder/lpdi-cli-app/internal/supabase"
)

func newSrv(t *testing.T, body string) (*httptest.Server, *supabase.DataClient) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, supabase.NewDataClient(srv.URL, "anon")
}

func TestRequire_GrantedAndDenied(t *testing.T) {
	body := `[{"roles":{"name":"user","role_permissions":[
        {"permissions":{"name":"can_run_protected_command"}}
    ]}}]`
	_, dc := newSrv(t, body)
	s := NewService(dc)

	if err := s.Require(context.Background(), "tok", "u-1", CanRunProtectedCommand); err != nil {
		t.Errorf("Require granted: %v", err)
	}

	err := s.Require(context.Background(), "tok", "u-1", "can_view_admin_panel")
	if err == nil || !errors.Is(err, ErrDenied) {
		t.Errorf("Require denied: got %v, want ErrDenied", err)
	}
}

func TestHas_NoRoles(t *testing.T) {
	_, dc := newSrv(t, `[]`)
	s := NewService(dc)
	ok, err := s.Has(context.Background(), "tok", "u-1", CanRunProtectedCommand)
	if err != nil {
		t.Fatalf("Has: %v", err)
	}
	if ok {
		t.Fatal("Has = true for empty role set")
	}
}
