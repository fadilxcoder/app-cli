package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/fadilxcoder/app-cli/internal/upgrade"
)

func newUpgradeCmd(currentVersion string) *cobra.Command {
	var checkOnly bool

	cmd := &cobra.Command{
		Use:     "upgrade",
		Aliases: []string{"update"},
		Short:   "Self-update myapp to the latest GitHub release",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()

			u := upgrade.New(currentVersion)
			rel, err := u.Latest(ctx)
			if err != nil {
				return err
			}

			if !u.IsNewer(rel.TagName) {
				fmt.Printf("Already on latest: %s\n", currentVersion)
				return nil
			}

			fmt.Printf("New release available: %s -> %s\n", currentVersion, rel.TagName)
			if checkOnly {
				fmt.Println(rel.HTMLURL)
				return nil
			}

			fmt.Println("Downloading and verifying checksum...")
			if err := u.Apply(ctx, rel); err != nil {
				return err
			}
			fmt.Printf("Updated to %s. Re-run `myapp <command>` to use the new binary.\n", rel.TagName)
			return nil
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "Print the available version without installing")
	return cmd
}
