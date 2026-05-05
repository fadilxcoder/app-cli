package supabase

import (
	"context"
	"fmt"
	"net/url"

	"github.com/fadilxcoder/lpdi-cli-app/pkg/httpclient"
)

// DataClient wraps Supabase's PostgREST data API.
type DataClient struct {
	http    *httpclient.Client
	anonKey string
}

// NewDataClient builds a DataClient targeting <supabaseURL>/rest/v1.
func NewDataClient(supabaseURL, anonKey string) *DataClient {
	return &DataClient{
		http:    httpclient.New(supabaseURL + "/rest/v1"),
		anonKey: anonKey,
	}
}

func (c *DataClient) headers(accessToken string) map[string]string {
	h := map[string]string{
		"apikey": c.anonKey,
	}
	if accessToken != "" {
		h["Authorization"] = "Bearer " + accessToken
	}
	return h
}

// PermissionRow is a flattened row from permissions joined through
// role_permissions and user_roles for a given user.
type PermissionRow struct {
	Name string `json:"name"`
}

// nested response shape for the join query below
type userRoleRow struct {
	Roles struct {
		Name            string `json:"name"`
		RolePermissions []struct {
			Permissions PermissionRow `json:"permissions"`
		} `json:"role_permissions"`
	} `json:"roles"`
}

// ListUserPermissions returns the distinct permission names granted to
// the given user via user_roles → roles → role_permissions → permissions.
//
// The query relies on PostgREST's foreign-key embedding. RLS must allow
// the authenticated user to read these rows (see sql/schema.sql).
func (c *DataClient) ListUserPermissions(ctx context.Context, accessToken, userID string) ([]string, error) {
	if userID == "" {
		return nil, fmt.Errorf("user id is required")
	}
	q := url.Values{}
	q.Set("user_id", "eq."+userID)
	q.Set("select", "roles(name,role_permissions(permissions(name)))")

	var rows []userRoleRow
	path := "/user_roles?" + q.Encode()
	if err := c.http.Do(ctx, "GET", path, c.headers(accessToken), nil, &rows); err != nil {
		return nil, fmt.Errorf("list user permissions: %w", err)
	}

	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, r := range rows {
		for _, rp := range r.Roles.RolePermissions {
			name := rp.Permissions.Name
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return out, nil
}

// ListUserRoles returns the role names assigned to the given user.
func (c *DataClient) ListUserRoles(ctx context.Context, accessToken, userID string) ([]string, error) {
	if userID == "" {
		return nil, fmt.Errorf("user id is required")
	}
	q := url.Values{}
	q.Set("user_id", "eq."+userID)
	q.Set("select", "roles(name)")

	var rows []struct {
		Roles struct {
			Name string `json:"name"`
		} `json:"roles"`
	}
	path := "/user_roles?" + q.Encode()
	if err := c.http.Do(ctx, "GET", path, c.headers(accessToken), nil, &rows); err != nil {
		return nil, fmt.Errorf("list user roles: %w", err)
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.Roles.Name != "" {
			out = append(out, r.Roles.Name)
		}
	}
	return out, nil
}
