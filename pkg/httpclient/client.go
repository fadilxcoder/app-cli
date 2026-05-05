package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a thin JSON-over-HTTP helper used by the Supabase clients.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// New returns a Client with a sensible default timeout.
func New(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 20 * time.Second},
	}
}

// Do executes a JSON request. If body is non-nil it is marshalled as JSON.
// If out is non-nil the response body is unmarshalled into it.
// Any non-2xx response is returned as an *Error.
func (c *Client) Do(
	ctx context.Context,
	method, path string,
	headers map[string]string,
	body any,
	out any,
) error {
	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(buf)
	}

	url := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &Error{
			StatusCode: resp.StatusCode,
			URL:        url,
			Body:       string(respBody),
		}
	}

	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// Error represents a non-2xx HTTP response.
type Error struct {
	StatusCode int
	URL        string
	Body       string
}

func (e *Error) Error() string {
	return fmt.Sprintf("http %d on %s: %s", e.StatusCode, e.URL, e.Body)
}
