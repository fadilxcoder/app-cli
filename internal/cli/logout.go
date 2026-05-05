package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/fadilxcoder/app-cli/internal/auth"
	"github.com/fadilxcoder/app-cli/internal/config"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear the local Supabase session",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()

			sess, err := auth.LoadSession()
			if err != nil && !errors.Is(err, config.ErrMissing) {
				return err
			}

			if sess != nil {
				if a, appErr := newApp(); appErr == nil {
					// Best-effort server-side revocation; ignore failures.
					_ = a.auth.Logout(ctx, sess.AccessToken)
				}
			}
			if err := auth.ClearSession(); err != nil {
				return err
			}
			fmt.Println("Logged out.")
			return nil
		},
	}
}
