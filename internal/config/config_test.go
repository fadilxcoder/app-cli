package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// scrubEnv ensures the test starts from a known empty env state and
// restores both vars on exit.
func scrubEnv(t *testing.T) {
	t.Helper()
	t.Setenv(EnvSupabaseURL, "")
	t.Setenv(EnvSupabaseAnonKey, "")
	_ = os.Unsetenv(EnvSupabaseURL)
	_ = os.Unsetenv(EnvSupabaseAnonKey)
}

func TestLoad_ProcessEnvWins(t *testing.T) {
	scrubEnv(t)
	t.Setenv(EnvSupabaseURL, "https://x.supabase.co/")
	t.Setenv(EnvSupabaseAnonKey, "anon-key")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.SupabaseURL != "https://x.supabase.co" {
		t.Errorf("trailing slash not stripped: %q", cfg.SupabaseURL)
	}
	if cfg.SupabaseAnonKey != "anon-key" {
		t.Errorf("anon key not propagated: %q", cfg.SupabaseAnonKey)
	}
}

func TestLoad_FallsBackToHomeDotMyappDotEnv(t *testing.T) {
	scrubEnv(t)

	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
	}
	// Switch CWD to a subdir without a .env so the fallback is exercised.
	cwd := filepath.Join(tmp, "work")
	if err := os.Mkdir(cwd, 0o755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	envPath := filepath.Join(dir, envFileName)
	body := "SUPABASE_URL=https://home.supabase.co\nSUPABASE_ANON_KEY=home-anon-key\n"
	if err := os.WriteFile(envPath, []byte(body), 0o600); err != nil {
		t.Fatalf("write home .env: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.SupabaseURL != "https://home.supabase.co" || cfg.SupabaseAnonKey != "home-anon-key" {
		t.Fatalf("home fallback not used: %+v", cfg)
	}
}

func TestLoad_ErrorMentionsFallbackLocations(t *testing.T) {
	scrubEnv(t)
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
	}
	cwd := filepath.Join(tmp, "work")
	if err := os.Mkdir(cwd, 0o755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "~/.myapp/.env") || !strings.Contains(msg, "./.env") {
		t.Errorf("error doesn't mention both .env locations: %s", msg)
	}
}
