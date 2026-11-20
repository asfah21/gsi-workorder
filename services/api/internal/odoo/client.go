package odoo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrInvalidCredentials means Odoo rejected the supplied login/password.
	ErrInvalidCredentials = errors.New("odoo: invalid credentials")
	// ErrUnavailable means Odoo could not be reached or returned a non-auth error.
	ErrUnavailable = errors.New("odoo: unavailable")
	// ErrUnexpectedResponse means Odoo returned success without a usable token.
	ErrUnexpectedResponse = errors.New("odoo: unexpected response")
)

// Client is the small Odoo read-only API client needed by M1A authentication.
// Passwords are only sent to Odoo and are never stored or logged here.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a client for the Odoo API base URL.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Login verifies a login/password pair against Odoo. It intentionally returns
// only success/failure: the application issues its own local JWT and does not
// persist or reuse Odoo's token.
func (c *Client) Login(ctx context.Context, login, password string) error {
	loginURL, err := url.JoinPath(c.baseURL, "api/auth/login")
	if err != nil {
		return fmt.Errorf("%w: build login URL: %v", ErrUnavailable, err)
	}

	payload, err := json.Marshal(map[string]string{
		"login":    login,
		"password": password,
	})
	if err != nil {
		return fmt.Errorf("%w: encode login request", ErrUnavailable)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("%w: build login request: %v", ErrUnavailable, err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("%w: read login response", ErrUnavailable)
	}

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrInvalidCredentials
	default:
		return fmt.Errorf("%w: status %d", ErrUnavailable, response.StatusCode)
	}

	var envelope struct {
		Data *struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("%w: decode login response", ErrUnexpectedResponse)
	}
	if envelope.Data == nil || strings.TrimSpace(envelope.Data.Token) == "" {
		return ErrUnexpectedResponse
	}
	return nil
}
