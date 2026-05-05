package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/fadilxcoder/lpdi-cli-app/internal/config"
)

// Session represents the locally-persisted auth state for the CLI.
type Session struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	UserID       string    `json:"user_id"`
	Email        string    `json:"email"`
}

// SaveSession persists the session to ~/.myapp/config.json (mode 0600).
func SaveSession(s *Session) error {
	path, err := config.ConfigFilePath()
	if err != nil {
		return err
	}
	buf, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		return fmt.Errorf("write session: %w", err)
	}
	return nil
}

// LoadSession reads the session file. Returns config.ErrMissing if it does
// not exist yet.
func LoadSession() (*Session, error) {
	path, err := config.ConfigFilePath()
	if err != nil {
		return nil, err
	}
	buf, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, config.ErrMissing
		}
		return nil, fmt.Errorf("read session: %w", err)
	}
	var s Session
	if err := json.Unmarshal(buf, &s); err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}
	return &s, nil
}

// ClearSession deletes the local session file. A missing file is not an error.
func ClearSession() error {
	path, err := config.ConfigFilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
