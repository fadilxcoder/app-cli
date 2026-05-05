package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/fadilxcoder/app-cli/internal/permissions"
)

func newRunSecureCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run-secure",
		Short: "Execute a permission-gated demo action",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			a, err := newApp()
			if err != nil {
				return err
			}

			sess, user, err := a.requireSession(ctx)
			if err != nil {
				return err
			}

			if !user.EmailVerified() {
				return errors.New("Access denied: email not verified")
			}

			if err := a.perms.Require(ctx, sess.AccessToken, user.ID, permissions.CanRunProtectedCommand); err != nil {
				if errors.Is(err, permissions.ErrDenied) {
					return fmt.Errorf("Access denied: %w", err)
				}
				return err
			}

			fmt.Println("Permission granted: secure action executed")
			return nil
		},
	}
}
