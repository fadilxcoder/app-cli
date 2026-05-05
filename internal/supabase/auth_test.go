package supabase

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSignInWithPassword_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/auth/v1/token") {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("grant_type") != "password" {
			t.Errorf("grant_type = %q", r.URL.Query().Get("grant_type"))
		}
		if r.Header.Get("apikey") != "anon-key" {
			t.Errorf("apikey header missing")
		}
		buf, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(buf), `"email":"a@b.c"`) {
			t.Errorf("body missing email: %s", buf)
		}
		_, _ = io.WriteString(w, `{
            "access_token":"AT","refresh_token":"RT","token_type":"bearer","expires_in":3600,
            "user":{"id":"u-1","email":"a@b.c","email_confirmed_at":"2026-01-01T00:00:00Z"}
        }`)
	}))
	defer srv.Close()

	c := NewAuthClient(srv.URL, "anon-key")
	tok, err := c.SignInWithPassword(context.Background(), "a@b.c", "secret")
	if err != nil {
		t.Fatalf("SignInWithPassword: %v", err)
	}
	if tok.AccessToken != "AT" || tok.RefreshToken != "RT" || tok.ExpiresIn != 3600 {
		t.Fatalf("unexpected token: %+v", tok)
	}
	if tok.User.ID != "u-1" || tok.User.Email != "a@b.c" {
		t.Fatalf("unexpected user: %+v", tok.User)
	}
	if !tok.User.EmailVerified() {
		t.Fatal("EmailVerified() = false; want true")
	}
}

func TestEmailVerified_FalseWhenUnconfirmed(t *testing.T) {
	u := &User{} // both timestamps nil
	if u.EmailVerified() {
		t.Fatal("EmailVerified() = true; want false")
	}
}

func TestEmailVerified_TrueWhenConfirmedAt(t *testing.T) {
	now := time.Now()
	u := &User{ConfirmedAt: &now}
	if !u.EmailVerified() {
		t.Fatal("EmailVerified() = false; want true")
	}
}

func TestGetUser_RequiresToken(t *testing.T) {
	c := NewAuthClient("http://unused", "anon")
	if _, err := c.GetUser(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestRefreshSession_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("grant_type") != "refresh_token" {
			t.Errorf("grant_type = %q", r.URL.Query().Get("grant_type"))
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["refresh_token"] != "RT" {
			t.Errorf("refresh_token = %q", body["refresh_token"])
		}
		_, _ = io.WriteString(w, `{"access_token":"AT2","refresh_token":"RT2","token_type":"bearer","expires_in":3600,"user":{"id":"u-1","email":"a@b.c"}}`)
	}))
	defer srv.Close()

	c := NewAuthClient(srv.URL, "anon")
	tok, err := c.RefreshSession(context.Background(), "RT")
	if err != nil {
		t.Fatalf("RefreshSession: %v", err)
	}
	if tok.AccessToken != "AT2" || tok.RefreshToken != "RT2" {
		t.Fatalf("unexpected token: %+v", tok)
	}
}

func TestRefreshSession_RequiresToken(t *testing.T) {
	c := NewAuthClient("http://unused", "anon")
	if _, err := c.RefreshSession(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty refresh_token")
	}
}
