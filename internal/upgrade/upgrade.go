// Package upgrade implements myapp's self-update flow against GitHub Releases.
package upgrade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	defaultOwner = "fadilxcoder"
	defaultRepo  = "app-cli"
	binaryName   = "myapp"
	userAgent    = "myapp-self-updater"
)

// Release describes the subset of the GitHub Releases API we care about.
type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Draft   bool   `json:"draft"`
}

// Updater talks to GitHub Releases and replaces the current binary in place.
type Updater struct {
	Owner   string
	Repo    string
	HTTP    *http.Client
	Current string // current version (e.g. "v0.1.0" or "dev")
}

// New constructs an Updater for the project's hosted GitHub repo.
func New(currentVersion string) *Updater {
	return &Updater{
		Owner:   defaultOwner,
		Repo:    defaultRepo,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
		Current: currentVersion,
	}
}

// Latest returns the most recent published (non-draft) release.
func (u *Updater) Latest(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", u.Owner, u.Repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := u.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query latest release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("github api %d: %s", resp.StatusCode, body)
	}
	var rel Release
	if err := jsonDecode(resp.Body, &rel); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	if rel.TagName == "" {
		return nil, errors.New("github returned an empty tag_name")
	}
	return &rel, nil
}

// IsNewer reports whether tag is strictly greater than the Updater's
// current version. Semver-ish lexical comparison after stripping the
// leading "v"; good enough for tags like vMAJOR.MINOR.PATCH.
func (u *Updater) IsNewer(tag string) bool {
	if u.Current == "" || u.Current == "dev" {
		return true
	}
	return normalize(tag) > normalize(u.Current)
}

func normalize(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

// Apply downloads the right asset for the host OS/arch and atomically
// replaces the current binary on disk. The caller should print a message
// telling the user to re-run; the running process keeps using the old code.
func (u *Updater) Apply(ctx context.Context, rel *Release) error {
	asset, err := AssetName()
	if err != nil {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate current binary: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	if err := writableDir(exe); err != nil {
		return err
	}

	binURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s",
		u.Owner, u.Repo, rel.TagName, asset)
	sumsURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/SHA256SUMS",
		u.Owner, u.Repo, rel.TagName)

	tmp := exe + ".new"
	if err := download(ctx, u.HTTP, binURL, tmp); err != nil {
		return fmt.Errorf("download %s: %w", asset, err)
	}
	defer os.Remove(tmp) // best-effort if we bail before rename

	if err := verifyChecksum(ctx, u.HTTP, sumsURL, asset, tmp); err != nil {
		return fmt.Errorf("verify download: %w", err)
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		return fmt.Errorf("chmod new binary: %w", err)
	}

	if err := os.Rename(tmp, exe); err != nil {
		return fmt.Errorf("install new binary at %s: %w", exe, err)
	}
	return nil
}

// AssetName returns the release artifact name for the host OS/arch
// (e.g. "myapp-linux-amd64").
func AssetName() (string, error) {
	switch runtime.GOOS {
	case "linux", "darwin":
	default:
		return "", fmt.Errorf("self-update is unsupported on %s", runtime.GOOS)
	}
	switch runtime.GOARCH {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("self-update is unsupported on %s", runtime.GOARCH)
	}
	return fmt.Sprintf("%s-%s-%s", binaryName, runtime.GOOS, runtime.GOARCH), nil
}

func writableDir(exe string) error {
	dir := filepath.Dir(exe)
	probe := filepath.Join(dir, ".myapp-upgrade-probe")
	f, err := os.Create(probe)
	if err != nil {
		return fmt.Errorf("cannot write to %s — re-run with sudo or move the binary to a writable directory: %w", dir, err)
	}
	_ = f.Close()
	_ = os.Remove(probe)
	return nil
}

func download(ctx context.Context, c *http.Client, url, dst string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("http %d: %s", resp.StatusCode, body)
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = os.Remove(dst)
		return err
	}
	return nil
}

func verifyChecksum(ctx context.Context, c *http.Client, sumsURL, asset, file string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", sumsURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// SHA256SUMS is optional — older releases may not have it. Don't
		// fail the upgrade if it's simply missing.
		if resp.StatusCode == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("fetch SHA256SUMS: http %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	want := ""
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == asset {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("no checksum entry for %s in SHA256SUMS", asset)
	}

	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != want {
		return fmt.Errorf("checksum mismatch (got %s, want %s)", got, want)
	}
	return nil
}
