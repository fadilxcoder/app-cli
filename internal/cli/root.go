package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/fadilxcoder/lpdi-cli-app/internal/auth"
	"github.com/fadilxcoder/lpdi-cli-app/internal/config"
	"github.com/fadilxcoder/lpdi-cli-app/internal/permissions"
	"github.com/fadilxcoder/lpdi-cli-app/internal/supabase"
)

// Execute builds the root command and runs the CLI.
func Execute(version string) error {
	root := &cobra.Command{
		Use:           "myapp",
		Short:         "myapp — a permission-aware CLI backed by Supabase",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}

	root.AddCommand(
		newLoginCmd(),
		newLogoutCmd(),
		newWhoamiCmd(),
		newRunSecureCmd(),
	)
	return root.Execute()
}

// app bundles the runtime collaborators a command needs.
type app struct {
	cfg   *config.Config
	auth  *supabase.AuthClient
	data  *supabase.DataClient
	perms *permissions.Service
}

func newApp() (*app, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	authClient := supabase.NewAuthClient(cfg.SupabaseURL, cfg.SupabaseAnonKey)
	dataClient := supabase.NewDataClient(cfg.SupabaseURL, cfg.SupabaseAnonKey)
	return &app{
		cfg:   cfg,
		auth:  authClient,
		data:  dataClient,
		perms: permissions.NewService(dataClient),
	}, nil
}

// requireSession loads the local session, transparently refreshing the
// access token if it has expired or the server rejects it.
func (a *app) requireSession(ctx context.Context) (*auth.Session, *supabase.User, error) {
	sess, err := auth.LoadSession()
	if err != nil {
		if errors.Is(err, config.ErrMissing) {
			return nil, nil, fmt.Errorf("not logged in — run `myapp login` first")
		}
		return nil, nil, err
	}

	// Proactively refresh if we know the token has expired.
	if !sess.ExpiresAt.IsZero() && time.Now().After(sess.ExpiresAt) {
		if refreshed, rerr := a.refresh(ctx, sess); rerr == nil {
			sess = refreshed
		}
	}

	user, err := a.auth.GetUser(ctx, sess.AccessToken)
	if err == nil {
		return sess, user, nil
	}

	// Reactive refresh on rejection (e.g. revoked early).
	refreshed, rerr := a.refresh(ctx, sess)
	if rerr != nil {
		return nil, nil, fmt.Errorf("session invalid — run `myapp login` again: %w", err)
	}
	user, err = a.auth.GetUser(ctx, refreshed.AccessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("session invalid after refresh — run `myapp login` again: %w", err)
	}
	return refreshed, user, nil
}

// refresh exchanges the stored refresh_token for a new access_token and
// persists the result. Errors propagate so callers can fall back to a
// "please log in again" message.
func (a *app) refresh(ctx context.Context, sess *auth.Session) (*auth.Session, error) {
	if sess.RefreshToken == "" {
		return nil, errors.New("no refresh_token in local session")
	}
	tok, err := a.auth.RefreshSession(ctx, sess.RefreshToken)
	if err != nil {
		return nil, err
	}
	updated := &auth.Session{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenType:    tok.TokenType,
		ExpiresAt:    time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second),
		UserID:       tok.User.ID,
		Email:        tok.User.Email,
	}
	if updated.UserID == "" {
		updated.UserID = sess.UserID
	}
	if updated.Email == "" {
		updated.Email = sess.Email
	}
	if err := auth.SaveSession(updated); err != nil {
		return nil, err
	}
	return updated, nil
}
