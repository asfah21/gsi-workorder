package p2h

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotConfigured      = errors.New("p2h: Odoo P2H API is not configured")
	ErrUnauthorized       = errors.New("p2h: Odoo rejected the API token")
	ErrUnavailable        = errors.New("p2h: Odoo P2H API unavailable")
	ErrUnexpectedResponse = errors.New("p2h: unexpected Odoo response")
)

const maxResponseBytes = 16 << 20

type Client struct {
	baseURL  string
	http     *http.Client
	token    string
	login    string
	password string

	mu          sync.Mutex
	cachedToken string
	expiresAt   time.Time
}

func NewClient(baseURL string, timeout time.Duration, token, login, password string) *Client {
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		http:     &http.Client{Timeout: timeout},
		token:    strings.TrimSpace(token),
		login:    strings.TrimSpace(login),
		password: password,
	}
}

func (c *Client) Configured() bool {
	return c.baseURL != "" && (c.token != "" || (c.login != "" && c.password != ""))
}

type ListPage struct {
	Reports []Report
	Total   int // -1 when the API does not expose a total.
}

func (c *Client) List(ctx context.Context, limit, offset int) (ListPage, error) {
	if !c.Configured() {
		return ListPage{}, ErrNotConfigured
	}
	if limit < 1 {
		limit = 100
	}
	path := "/api/p2h?limit=" + strconv.Itoa(limit) + "&offset=" + strconv.Itoa(offset)
	data, err := c.get(ctx, path)
	if err != nil {
		return ListPage{}, err
	}

	dtos, total, err := decodeList(data)
	if err != nil {
		return ListPage{}, err
	}
	reports := make([]Report, 0, len(dtos))
	for _, dto := range dtos {
		report, parseErr := parseReport(dto)
		if parseErr != nil {
			return ListPage{}, parseErr
		}
		reports = append(reports, report)
	}
	return ListPage{Reports: reports, Total: total}, nil
}

func (c *Client) Get(ctx context.Context, id int64) (Report, error) {
	if !c.Configured() {
		return Report{}, ErrNotConfigured
	}
	if id <= 0 {
		return Report{}, fmt.Errorf("p2h: invalid report id")
	}
	data, err := c.get(ctx, "/api/p2h/"+strconv.FormatInt(id, 10))
	if err != nil {
		return Report{}, err
	}
	dto, raw, err := decodeDetail(data)
	if err != nil {
		return Report{}, err
	}
	report, err := parseReport(dto)
	if err != nil {
		return Report{}, err
	}
	report.Raw = raw
	report.Findings = parseFindings(dto.ChecklistLines)
	return report, nil
}

func (c *Client) get(ctx context.Context, path string) (json.RawMessage, error) {
	for attempt := 0; attempt < 2; attempt++ {
		token, err := c.accessToken(ctx)
		if err != nil {
			return nil, err
		}
		endpoint := c.baseURL + path
		if _, err := url.Parse(endpoint); err != nil {
			return nil, fmt.Errorf("%w: build URL: %v", ErrUnavailable, err)
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("%w: build request: %v", ErrUnavailable, err)
		}
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Authorization", "Bearer "+token)

		response, err := c.http.Do(request)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
		response.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("%w: read response", ErrUnavailable)
		}

		if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
			if attempt == 0 {
				c.clearToken()
				continue
			}
			return nil, ErrUnauthorized
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return nil, fmt.Errorf("%w: status %d", ErrUnavailable, response.StatusCode)
		}

		var envelope envelope
		if err := json.Unmarshal(body, &envelope); err != nil {
			return nil, fmt.Errorf("%w: decode envelope", ErrUnexpectedResponse)
		}
		if envelope.Code == http.StatusUnauthorized || envelope.Code == http.StatusForbidden {
			if attempt == 0 {
				c.clearToken()
				continue
			}
			return nil, ErrUnauthorized
		}
		if envelope.Code != 0 && envelope.Code != http.StatusOK {
			message := strings.TrimSpace(envelope.Message)
			if message == "" {
				message = "Odoo P2H API error"
			}
			return nil, fmt.Errorf("%w: code %d: %s", ErrUnavailable, envelope.Code, message)
		}
		if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
			return nil, fmt.Errorf("%w: empty data", ErrUnexpectedResponse)
		}
		return envelope.Data, nil
	}
	return nil, ErrUnauthorized
}

func (c *Client) accessToken(ctx context.Context) (string, error) {
	if c.token != "" {
		return c.token, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cachedToken != "" && (c.expiresAt.IsZero() || time.Now().Before(c.expiresAt)) {
		return c.cachedToken, nil
	}
	if c.login == "" || c.password == "" {
		return "", ErrNotConfigured
	}

	endpoint, err := url.JoinPath(c.baseURL, "api/auth/login")
	if err != nil {
		return "", fmt.Errorf("%w: build login URL: %v", ErrUnavailable, err)
	}
	payload, err := json.Marshal(map[string]string{"login": c.login, "password": c.password})
	if err != nil {
		return "", fmt.Errorf("%w: encode login", ErrUnavailable)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("%w: build login request: %v", ErrUnavailable, err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	response, err := c.http.Do(request)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	response.Body.Close()
	if readErr != nil {
		return "", fmt.Errorf("%w: read login response", ErrUnavailable)
	}
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return "", ErrUnauthorized
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("%w: login status %d", ErrUnavailable, response.StatusCode)
	}

	var envelope struct {
		Code int `json:"code"`
		Data *struct {
			AccessToken string `json:"access_token"`
			Token       string `json:"token"`
			ExpiresAt   string `json:"expires_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", fmt.Errorf("%w: decode login response", ErrUnexpectedResponse)
	}
	if envelope.Code == http.StatusUnauthorized || envelope.Code == http.StatusForbidden {
		return "", ErrUnauthorized
	}
	if envelope.Code != 0 && envelope.Code != http.StatusOK {
		return "", fmt.Errorf("%w: login code %d", ErrUnavailable, envelope.Code)
	}
	if envelope.Data == nil {
		return "", fmt.Errorf("%w: login data missing", ErrUnexpectedResponse)
	}
	token := strings.TrimSpace(envelope.Data.AccessToken)
	if token == "" {
		token = strings.TrimSpace(envelope.Data.Token)
	}
	if token == "" {
		return "", fmt.Errorf("%w: login token missing", ErrUnexpectedResponse)
	}
	c.cachedToken = token
	c.expiresAt = time.Now().Add(10 * time.Minute)
	if raw := strings.TrimSpace(envelope.Data.ExpiresAt); raw != "" {
		if expiry, parseErr := time.Parse(time.RFC3339, raw); parseErr == nil {
			c.expiresAt = expiry.Add(-30 * time.Second)
		}
	}
	return token, nil
}

func (c *Client) clearToken() {
	c.mu.Lock()
	c.cachedToken = ""
	c.expiresAt = time.Time{}
	c.mu.Unlock()
}
