package upgrade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		current, candidate string
		want               bool
	}{
		{"v0.1.0", "v0.2.0", true},
		{"v0.1.0", "v0.1.0", false},
		{"v0.2.0", "v0.1.9", false},
		{"dev", "v0.1.0", true},
		{"", "v0.1.0", true},
	}
	for _, tc := range cases {
		u := &Updater{Current: tc.current}
		if got := u.IsNewer(tc.candidate); got != tc.want {
			t.Errorf("IsNewer(%s -> %s) = %v, want %v", tc.current, tc.candidate, got, tc.want)
		}
	}
}

func TestLatest_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/releases/latest") {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"tag_name":"v0.2.0","html_url":"https://example/v0.2.0","draft":false}`))
	}))
	defer srv.Close()

	u := newWithBase(srv.URL)
	rel, err := u.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if rel.TagName != "v0.2.0" {
		t.Errorf("tag = %q", rel.TagName)
	}
}

func TestApply_DownloadsAndReplacesBinary(t *testing.T) {
	asset, err := AssetName()
	if err != nil {
		t.Skip(err)
	}
	payload := []byte("#!/bin/sh\necho updated\n")
	sum := sha256.Sum256(payload)
	sumLine := fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), asset)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/"+asset):
			_, _ = w.Write(payload)
		case strings.HasSuffix(r.URL.Path, "/SHA256SUMS"):
			_, _ = w.Write([]byte(sumLine))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	exe := filepath.Join(dir, "myapp")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatalf("seed exe: %v", err)
	}

	u := newWithBase(srv.URL)
	if err := u.applyTo(context.Background(), &Release{TagName: "v0.2.0"}, exe); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatalf("read replaced: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("content not replaced: %q", got)
	}
}

func TestApply_RejectsTamperedDownload(t *testing.T) {
	asset, err := AssetName()
	if err != nil {
		t.Skip(err)
	}
	payload := []byte("legitimate")
	wrongSum := strings.Repeat("0", 64) // never matches

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/"+asset):
			_, _ = w.Write(payload)
		case strings.HasSuffix(r.URL.Path, "/SHA256SUMS"):
			fmt.Fprintf(w, "%s  %s\n", wrongSum, asset)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	exe := filepath.Join(dir, "myapp")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatalf("seed exe: %v", err)
	}

	u := newWithBase(srv.URL)
	err = u.applyTo(context.Background(), &Release{TagName: "v0.2.0"}, exe)
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("expected checksum error, got %v", err)
	}
	// The original binary must remain untouched on failure.
	got, _ := os.ReadFile(exe)
	if string(got) != "old" {
		t.Errorf("binary was mutated despite verify failure: %q", got)
	}
}

// --- test helpers ---

// newWithBase builds an Updater that hits a custom base URL — used to
// retarget GitHub API + asset downloads at an httptest server.
func newWithBase(base string) *testUpdater {
	return &testUpdater{
		Updater: &Updater{
			Owner:   "test",
			Repo:    "repo",
			HTTP:    &http.Client{},
			Current: "v0.1.0",
		},
		base: base,
	}
}

type testUpdater struct {
	*Updater
	base string
}

// Latest is overridden to hit the test server's /releases/latest.
func (t *testUpdater) Latest(ctx context.Context) (*Release, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", t.base+"/repos/test/repo/releases/latest", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := t.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var rel Release
	if err := jsonDecode(resp.Body, &rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// applyTo runs the same flow as Updater.Apply but against a known path,
// so we don't need to spawn a real binary at os.Executable() in tests.
func (t *testUpdater) applyTo(ctx context.Context, rel *Release, exe string) error {
	asset, err := AssetName()
	if err != nil {
		return err
	}
	binURL := t.base + "/" + rel.TagName + "/" + asset
	sumsURL := t.base + "/" + rel.TagName + "/SHA256SUMS"

	tmp := exe + ".new"
	if err := download(ctx, t.HTTP, binURL, tmp); err != nil {
		return err
	}
	defer os.Remove(tmp)
	if err := verifyChecksum(ctx, t.HTTP, sumsURL, asset, tmp); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		return err
	}
	return os.Rename(tmp, exe)
}
