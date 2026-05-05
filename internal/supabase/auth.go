package supabase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fadilxcoder/lpdi-cli-app/pkg/httpclient"
)

// AuthClient wraps Supabase's GoTrue REST API.
type AuthClient struct {
	http    *httpclient.Client
	anonKey string
}

// NewAuthClient builds an AuthClient targeting <supabaseURL>/auth/v1.
func NewAuthClient(supabaseURL, anonKey string) *AuthClient {
	return &AuthClient{
		http:    httpclient.New(supabaseURL + "/auth/v1"),
		anonKey: anonKey,
	}
}

// TokenResponse is the GoTrue /token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

// User is the GoTrue user object (minimal fields).
type User struct {
	ID               string                 `json:"id"`
	Email            string                 `json:"email"`
	EmailConfirmedAt *time.Time             `json:"email_confirmed_at"`
	ConfirmedAt      *time.Time             `json:"confirmed_at"`
	UserMetadata     map[string]any         `json:"user_metadata"`
	AppMetadata      map[string]any         `json:"app_metadata"`
	Identities       []map[string]any       `json:"identities,omitempty"`
	Aud              string                 `json:"aud"`
	Role             string                 `json:"role"`
	CreatedAt        time.Time              `json:"created_at"`
	Raw              map[string]any         `json:"-"`
}

// EmailVerified returns true if the user's email has been confirmed.
func (u *User) EmailVerified() bool {
	if u == nil {
		return false
	}
	if u.EmailConfirmedAt != nil && !u.EmailConfirmedAt.IsZero() {
		return true
	}
	if u.ConfirmedAt != nil && !u.ConfirmedAt.IsZero() {
		return true
	}
	return false
}

// SignInWithPassword exchanges email+password for an access token.
func (c *AuthClient) SignInWithPassword(ctx context.Context, email, password string) (*TokenResponse, error) {
	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}
	headers := map[string]string{
		"apikey": c.anonKey,
	}
	body := map[string]string{
		"email":    email,
		"password": password,
	}
	var out TokenResponse
	if err := c.http.Do(ctx, "POST", "/token?grant_type=password", headers, body, &out); err != nil {
		return nil, fmt.Errorf("sign in: %w", err)
	}
	return &out, nil
}

// GetUser fetches the currently-authenticated user using the given access token.
func (c *AuthClient) GetUser(ctx context.Context, accessToken string) (*User, error) {
	if accessToken == "" {
		return nil, errors.New("access token is required")
	}
	headers := map[string]string{
		"apikey":        c.anonKey,
		"Authorization": "Bearer " + accessToken,
	}
	var out User
	if err := c.http.Do(ctx, "GET", "/user", headers, nil, &out); err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &out, nil
}

// RefreshSession exchanges a refresh_token for a fresh access_token.
func (c *AuthClient) RefreshSession(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	if refreshToken == "" {
		return nil, errors.New("refresh token is required")
	}
	headers := map[string]string{
		"apikey": c.anonKey,
	}
	body := map[string]string{
		"refresh_token": refreshToken,
	}
	var out TokenResponse
	if err := c.http.Do(ctx, "POST", "/token?grant_type=refresh_token", headers, body, &out); err != nil {
		return nil, fmt.Errorf("refresh session: %w", err)
	}
	return &out, nil
}

// Logout invalidates the access token server-side. A failure here is
// non-fatal — the local session is always cleared by the caller.
func (c *AuthClient) Logout(ctx context.Context, accessToken string) error {
	headers := map[string]string{
		"apikey":        c.anonKey,
		"Authorization": "Bearer " + accessToken,
	}
	return c.http.Do(ctx, "POST", "/logout", headers, nil, nil)
}
