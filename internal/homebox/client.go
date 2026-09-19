// Package homebox is a small client for the Homebox v0.26+ REST API
// (entities/tags/entity-types generation).
package homebox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client talks to a single Homebox instance. It is safe for concurrent use.
type Client struct {
	baseURL  string
	email    string
	password string
	http     *http.Client

	mu              sync.Mutex
	token           string // includes the "Bearer " prefix as returned by the API
	attachmentToken string
	expiresAt       time.Time
}

// NewClient creates an unauthenticated client. Call Login before making requests.
func NewClient(baseURL, email, password string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		email:   email,
		password: password,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Login authenticates against POST /api/v1/users/login and stores the token.
func (c *Client) Login(ctx context.Context) error {
	body, _ := json.Marshal(map[string]any{
		"username":     c.email,
		"password":     c.password,
		"stayLoggedIn": true,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/users/login", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("homebox: build login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("homebox: login: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("homebox: login failed: %s: %s", resp.Status, snippet(resp.Body))
	}
	var tok TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return fmt.Errorf("homebox: decode login response: %w", err)
	}
	if tok.Token == "" {
		return fmt.Errorf("homebox: login response contained no token")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	// The API returns the token already prefixed with "Bearer ".
	c.token = tok.Token
	c.attachmentToken = tok.AttachmentToken
	// Refresh a bit before actual expiry to avoid edge-case 401s.
	c.expiresAt = tok.ExpiresAt.Add(-2 * time.Minute)
	log.Printf("logged in to %s as %s (token valid until %s)", c.baseURL, c.email, tok.ExpiresAt.Format(time.RFC3339))
	return nil
}

// Status performs an unauthenticated health check against GET /api/v1/status.
func (c *Client) Status(ctx context.Context) (*StatusResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/status", nil)
	if err != nil {
		return nil, fmt.Errorf("homebox: build status request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("homebox: GET /api/v1/status: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("homebox: GET /api/v1/status: %s: %s", resp.Status, snippet(resp.Body))
	}
	var out StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("homebox: decode status: %w", err)
	}
	return &out, nil
}

// authValid reports whether a stored token exists and has not expired.
// Must be called with c.mu held.
func (c *Client) authValid() bool {
	return c.token != "" && (c.expiresAt.IsZero() || time.Now().Before(c.expiresAt))
}

// ensureAuth logs in if there is no valid token.
func (c *Client) ensureAuth(ctx context.Context) error {
	c.mu.Lock()
	valid := c.authValid()
	c.mu.Unlock()
	if valid {
		return nil
	}
	return c.Login(ctx)
}

// doJSON performs an authenticated JSON request and decodes the response into out.
// If out is nil, the response body is discarded. On 401 it re-authenticates once and retries.
func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, payload any, out any) error {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			return fmt.Errorf("homebox: encode request body: %w", err)
		}
	}

	for attempt := 0; attempt < 2; attempt++ {
		if err := c.ensureAuth(ctx); err != nil {
			return err
		}
		resp, err := c.doRaw(ctx, method, path, query, body, "application/json", nil)
		if err != nil {
			return err
		}
		unauthorized := resp.StatusCode == http.StatusUnauthorized
		if unauthorized {
			resp.Body.Close()
			c.mu.Lock()
			c.token = "" // force re-login on the next attempt
			c.mu.Unlock()
			if attempt == 0 {
				continue
			}
			return fmt.Errorf("homebox: %s %s: 401 Unauthorized after re-login", method, path)
		}
		if err := checkStatus(method, path, resp); err != nil {
			resp.Body.Close()
			return err
		}
		if out != nil && resp.StatusCode != http.StatusNoContent {
			defer resp.Body.Close()
			if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
				return fmt.Errorf("homebox: decode %s %s response: %w", method, path, err)
			}
			return nil
		}
		resp.Body.Close()
		return nil
	}
	return fmt.Errorf("homebox: %s %s: exhausted retries", method, path)
}

// doMultipart uploads a file as multipart/form-data to path and decodes the JSON
// response into out. Like doJSON it re-authenticates once on 401.
func (c *Client) doMultipart(ctx context.Context, method, path string, file io.Reader, filename string, fields map[string]string, out any) error {
	for attempt := 0; attempt < 2; attempt++ {
		if err := c.ensureAuth(ctx); err != nil {
			return err
		}
		// The request body is consumed on the first attempt, so build a fresh
		// pipe-backed body each time and stream the file into it.
		pr, pw := io.Pipe()
		mw := multipart.NewWriter(pw)
		go func() {
			var err error
			defer func() { pw.CloseWithError(err) }()
			for k, v := range fields {
				if werr := mw.WriteField(k, v); werr != nil {
					err = werr
					return
				}
			}
			fw, werr := mw.CreateFormFile("file", filename)
			if werr != nil {
				err = werr
				return
			}
			if _, err = io.Copy(fw, file); err != nil {
				return
			}
			err = mw.Close()
		}()

		contentType := mw.FormDataContentType()
		resp, err := c.doRaw(ctx, method, path, nil, nil, contentType, pr)
		if err != nil {
			pr.Close()
			return err
		}
		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			pr.Close()
			c.mu.Lock()
			c.token = ""
			c.mu.Unlock()
			if attempt == 0 {
				// Rewind is not possible on arbitrary readers; the caller passes
				// a *os.File, so seek back to the start if we can.
				if seeker, ok := file.(io.Seeker); ok {
					if _, serr := seeker.Seek(0, io.SeekStart); serr != nil {
						return fmt.Errorf("homebox: cannot rewind file for retry: %w", serr)
					}
				} else {
					return fmt.Errorf("homebox: %s %s: 401 Unauthorized (file not seekable, cannot retry)", method, path)
				}
				continue
			}
			return fmt.Errorf("homebox: %s %s: 401 Unauthorized after re-login", method, path)
		}
		defer resp.Body.Close()
		if err := checkStatus(method, path, resp); err != nil {
			return err
		}
		if out != nil && resp.StatusCode != http.StatusNoContent {
			if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
				return fmt.Errorf("homebox: decode %s %s response: %w", method, path, err)
			}
		}
		return nil
	}
	return fmt.Errorf("homebox: %s %s: exhausted retries", method, path)
}

// doRaw builds and sends an HTTP request. Either body or rawBody may be nil.
func (c *Client) doRaw(ctx context.Context, method, path string, query url.Values, body []byte, contentType string, rawBody io.Reader) (*http.Response, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var rdr io.Reader
	switch {
	case body != nil:
		rdr = bytes.NewReader(body)
	case rawBody != nil:
		rdr = rawBody
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return nil, fmt.Errorf("homebox: build %s %s request: %w", method, path, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	c.mu.Lock()
	token := c.token
	c.mu.Unlock()
	req.Header.Set("Authorization", token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("homebox: %s %s: %w", method, path, err)
	}
	return resp, nil
}

// checkStatus converts non-2xx responses into errors including a body snippet.
func checkStatus(method, path string, resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("homebox: %s %s: %s: %s", method, path, resp.Status, snippet(resp.Body))
}

// snippet reads up to 512 bytes from r for error messages.
func snippet(r io.Reader) string {
	b, _ := io.ReadAll(io.LimitReader(r, 512))
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "<empty response body>"
	}
	return s
}
