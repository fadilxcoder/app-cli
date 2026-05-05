package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

const (
	EnvSupabaseURL     = "SUPABASE_URL"
	EnvSupabaseAnonKey = "SUPABASE_ANON_KEY"

	configDirName  = ".myapp"
	configFileName = "config.json"
)

// Config holds runtime configuration sourced from env / .env.
type Config struct {
	SupabaseURL     string
	SupabaseAnonKey string
}

// Load reads SUPABASE_URL and SUPABASE_ANON_KEY from the environment.
// If a .env file exists in the working directory it is loaded first
// (existing env vars take precedence).
func Load() (*Config, error) {
	_ = godotenv.Load() // optional; ignore missing file

	url := strings.TrimSpace(os.Getenv(EnvSupabaseURL))
	key := strings.TrimSpace(os.Getenv(EnvSupabaseAnonKey))

	if url == "" {
		return nil, fmt.Errorf("%s is not set", EnvSupabaseURL)
	}
	if key == "" {
		return nil, fmt.Errorf("%s is not set", EnvSupabaseAnonKey)
	}

	url = strings.TrimRight(url, "/")
	return &Config{SupabaseURL: url, SupabaseAnonKey: key}, nil
}

// ConfigDir returns ~/.myapp, creating it if it does not exist.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	dir := filepath.Join(home, configDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	return dir, nil
}

// ConfigFilePath returns ~/.myapp/config.json.
func ConfigFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// ErrMissing is returned when the local session file does not exist.
var ErrMissing = errors.New("no local session")
