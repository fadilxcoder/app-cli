package auth

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/fadilxcoder/lpdi-cli-app/internal/config"
)

func TestSessionRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
	}

	want := &Session{
		AccessToken:  "AT",
		RefreshToken: "RT",
		TokenType:    "bearer",
		ExpiresAt:    time.Now().Add(time.Hour).UTC().Truncate(time.Second),
		UserID:       "u-1",
		Email:        "a@b.c",
	}
	if err := SaveSession(want); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	got, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if got.AccessToken != want.AccessToken || got.RefreshToken != want.RefreshToken ||
		got.UserID != want.UserID || got.Email != want.Email {
		t.Fatalf("round trip mismatch: %+v vs %+v", got, want)
	}

	// File mode must not leak to other users.
	path, err := config.ConfigFilePath()
	if err != nil {
		t.Fatalf("ConfigFilePath: %v", err)
	}
	if filepath.Dir(path) != filepath.Join(tmp, ".myapp") {
		t.Fatalf("path not in temp: %s", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("session file mode = %v; want 0600", info.Mode().Perm())
	}

	if err := ClearSession(); err != nil {
		t.Fatalf("ClearSession: %v", err)
	}
	if _, err := LoadSession(); !errors.Is(err, config.ErrMissing) {
		t.Errorf("LoadSession after clear: got %v, want ErrMissing", err)
	}
}

func TestLoadSession_MissingReturnsTypedError(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
	}
	_, err := LoadSession()
	if !errors.Is(err, config.ErrMissing) {
		t.Fatalf("got %v, want ErrMissing", err)
	}
}
