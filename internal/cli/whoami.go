package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the current authenticated user, roles and permissions",
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

			roles, err := a.data.ListUserRoles(ctx, sess.AccessToken, user.ID)
			if err != nil {
				return fmt.Errorf("fetch roles: %w", err)
			}
			perms, err := a.perms.List(ctx, sess.AccessToken, user.ID)
			if err != nil {
				return fmt.Errorf("fetch permissions: %w", err)
			}

			fmt.Printf("User:           %s\n", user.Email)
			fmt.Printf("ID:             %s\n", user.ID)
			fmt.Printf("Email verified: %t\n", user.EmailVerified())
			fmt.Printf("Roles:          %s\n", joinOrDash(roles))
			fmt.Printf("Permissions:    %s\n", joinOrDash(perms))
			return nil
		},
	}
}

func joinOrDash(v []string) string {
	if len(v) == 0 {
		return "-"
	}
	return strings.Join(v, ", ")
}
