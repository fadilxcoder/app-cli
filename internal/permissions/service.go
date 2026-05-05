package permissions

import (
	"context"
	"errors"
	"fmt"

	"github.com/fadilxcoder/app-cli/internal/supabase"
)

// Well-known permissions used by the CLI.
const (
	CanRunProtectedCommand = "can_run_protected_command"
)

// ErrDenied is returned when a user lacks a required permission.
var ErrDenied = errors.New("permission denied")

// Service evaluates per-user permissions using the Supabase data API.
type Service struct {
	data *supabase.DataClient
}

// NewService constructs a permissions Service.
func NewService(data *supabase.DataClient) *Service {
	return &Service{data: data}
}

// Has returns true if the user has the named permission.
func (s *Service) Has(ctx context.Context, accessToken, userID, name string) (bool, error) {
	perms, err := s.data.ListUserPermissions(ctx, accessToken, userID)
	if err != nil {
		return false, err
	}
	for _, p := range perms {
		if p == name {
			return true, nil
		}
	}
	return false, nil
}

// Require returns ErrDenied wrapped with context if the user lacks the
// named permission.
func (s *Service) Require(ctx context.Context, accessToken, userID, name string) error {
	ok, err := s.Has(ctx, accessToken, userID, name)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: missing %q", ErrDenied, name)
	}
	return nil
}

// List returns every permission granted to the user.
func (s *Service) List(ctx context.Context, accessToken, userID string) ([]string, error) {
	return s.data.ListUserPermissions(ctx, accessToken, userID)
}
