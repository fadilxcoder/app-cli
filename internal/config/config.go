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
	envFileName    = ".env"
)

// Config holds runtime configuration sourced from env / .env.
type Config struct {
	SupabaseURL     string
	SupabaseAnonKey string
}

// Build-time defaults injected via `go build -ldflags "-X ..."`.
// Empty in source so tests and local dev builds still depend on env/.env.
// Released binaries embed the maintainer's hosted Supabase project so
// end users can run `myapp login` without any prior configuration.
var (
	bakedSupabaseURL     string
	bakedSupabaseAnonKey string
)

// Load reads SUPABASE_URL and SUPABASE_ANON_KEY from the environment.
// Lookup order (first match wins per key):
//  1. Process environment
//  2. ./.env in the current working directory
//  3. ~/.myapp/.env
//  4. Build-time defaults baked into released binaries
func Load() (*Config, error) {
	_ = godotenv.Load() // ./ .env — optional
	if home, err := os.UserHomeDir(); err == nil {
		_ = godotenv.Load(filepath.Join(home, configDirName, envFileName))
	}

	url := strings.TrimSpace(os.Getenv(EnvSupabaseURL))
	if url == "" {
		url = bakedSupabaseURL
	}
	key := strings.TrimSpace(os.Getenv(EnvSupabaseAnonKey))
	if key == "" {
		key = bakedSupabaseAnonKey
	}

	if url == "" || key == "" {
		return nil, fmt.Errorf(
			"%s and %s must be set — export them in your shell, or create ./.env or ~/.myapp/.env",
			EnvSupabaseURL, EnvSupabaseAnonKey,
		)
	}
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return nil, fmt.Errorf("%s is malformed (got %q) — expected an https:// URL", EnvSupabaseURL, url)
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
