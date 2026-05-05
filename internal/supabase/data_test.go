package supabase

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListUserPermissions_DedupesAcrossRoles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/rest/v1/user_roles") {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("user_id") != "eq.u-1" {
			t.Errorf("filter = %q", r.URL.Query().Get("user_id"))
		}
		if !strings.Contains(r.URL.Query().Get("select"), "role_permissions(permissions(name))") {
			t.Errorf("missing embedded select")
		}
		if r.Header.Get("Authorization") != "Bearer ACCESS" {
			t.Errorf("missing bearer token")
		}
		_, _ = io.WriteString(w, `[
            {"roles":{"name":"admin","role_permissions":[
                {"permissions":{"name":"can_run_protected_command"}},
                {"permissions":{"name":"can_view_admin_panel"}}
            ]}},
            {"roles":{"name":"user","role_permissions":[
                {"permissions":{"name":"can_run_protected_command"}}
            ]}}
        ]`)
	}))
	defer srv.Close()

	c := NewDataClient(srv.URL, "anon")
	perms, err := c.ListUserPermissions(context.Background(), "ACCESS", "u-1")
	if err != nil {
		t.Fatalf("ListUserPermissions: %v", err)
	}
	want := map[string]bool{"can_run_protected_command": false, "can_view_admin_panel": false}
	if len(perms) != 2 {
		t.Fatalf("got %d perms; want 2: %v", len(perms), perms)
	}
	for _, p := range perms {
		if _, ok := want[p]; !ok {
			t.Errorf("unexpected permission %q", p)
		}
		want[p] = true
	}
	for k, seen := range want {
		if !seen {
			t.Errorf("missing permission %q", k)
		}
	}
}

func TestListUserRoles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[{"roles":{"name":"admin"}},{"roles":{"name":"user"}}]`)
	}))
	defer srv.Close()

	c := NewDataClient(srv.URL, "anon")
	roles, err := c.ListUserRoles(context.Background(), "tok", "u-1")
	if err != nil {
		t.Fatalf("ListUserRoles: %v", err)
	}
	if len(roles) != 2 || roles[0] != "admin" || roles[1] != "user" {
		t.Fatalf("unexpected roles: %v", roles)
	}
}
