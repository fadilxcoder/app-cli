package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/fadilxcoder/app-cli/internal/auth"
)

func newLoginCmd() *cobra.Command {
	var email, password string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Supabase using email + password",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			a, err := newApp()
			if err != nil {
				return err
			}

			email = strings.TrimSpace(email)
			if email == "" {
				email, err = promptLine("Email: ")
				if err != nil {
					return err
				}
			}
			if password == "" {
				password, err = promptSecret("Password: ")
				if err != nil {
					return err
				}
			}
			if email == "" || password == "" {
				return errors.New("email and password are required")
			}

			tok, err := a.auth.SignInWithPassword(ctx, email, password)
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}

			sess := &auth.Session{
				AccessToken:  tok.AccessToken,
				RefreshToken: tok.RefreshToken,
				TokenType:    tok.TokenType,
				ExpiresAt:    time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second),
				UserID:       tok.User.ID,
				Email:        tok.User.Email,
			}
			if err := auth.SaveSession(sess); err != nil {
				return err
			}

			fmt.Printf("Logged in as %s\n", sess.Email)
			if !tok.User.EmailVerified() {
				fmt.Println("Warning: email is NOT verified. Restricted commands will be blocked.")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&email, "email", "e", "", "Account email")
	cmd.Flags().StringVarP(&password, "password", "p", "", "Account password (omit for interactive prompt)")
	return cmd
}

func promptLine(prompt string) (string, error) {
	fmt.Print(prompt)
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

func promptSecret(prompt string) (string, error) {
	fmt.Print(prompt)
	fd := int(syscall.Stdin)
	if !term.IsTerminal(fd) {
		// Fallback for non-interactive stdin (e.g. piped input)
		return promptLine("")
	}
	buf, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return strings.TrimSpace(string(buf)), nil
}
