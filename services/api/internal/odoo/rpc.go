package odoo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const rpcID = 1

type rpcRequest struct {
	JSONRPC string    `json:"jsonrpc"`
	Method  string    `json:"method"`
	Params  rpcParams `json:"params"`
	ID      int       `json:"id"`
}

type rpcParams struct {
	Service string `json:"service"`
	Method  string `json:"method"`
	Args    []any  `json:"args"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ServiceLogin authenticates a worker service account against Odoo JSON-RPC.
// The returned uid is used for subsequent read-only execute_kw calls.
func (c *Client) ServiceLogin(ctx context.Context, db, login, password string) (int64, error) {
	raw, err := c.call(ctx, "common", "login", []any{db, login, password})
	if err != nil {
		return 0, err
	}

	var uid int64
	if err := json.Unmarshal(raw, &uid); err == nil {
		if uid > 0 {
			return uid, nil
		}
	}

	var result bool
	if err := json.Unmarshal(raw, &result); err == nil && !result {
		return 0, ErrInvalidCredentials
	}
	return 0, ErrUnexpectedResponse
}

// SearchRead runs a read-only Odoo search_read. domain is the Odoo domain
// array, for example []any{[]any{"share", "=", false}}.
func (c *Client) SearchRead(
	ctx context.Context,
	db string,
	uid int64,
	password string,
	model string,
	domain []any,
	fields []string,
	searchContext map[string]any,
) ([]map[string]json.RawMessage, error) {
	kwargs := map[string]any{"fields": fields}
	if searchContext != nil {
		kwargs["context"] = searchContext
	}

	args := []any{
		db,
		uid,
		password,
		model,
		"search_read",
		[]any{domain},
		kwargs,
	}
	raw, err := c.call(ctx, "object", "execute_kw", args)
	if err != nil {
		return nil, err
	}

	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("%w: decode search_read result", ErrUnexpectedResponse)
	}
	return rows, nil
}

func (c *Client) call(ctx context.Context, service, method string, args []any) (json.RawMessage, error) {
	rpcURL, err := url.JoinPath(c.baseURL, "jsonrpc")
	if err != nil {
		return nil, fmt.Errorf("%w: build jsonrpc URL: %v", ErrUnavailable, err)
	}

	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  "call",
		Params: rpcParams{
			Service: service,
			Method:  method,
			Args:    args,
		},
		ID: rpcID,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: encode jsonrpc request", ErrUnavailable)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: build jsonrpc request: %v", ErrUnavailable, err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read jsonrpc response", ErrUnavailable)
	}

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, ErrInvalidCredentials
	default:
		return nil, fmt.Errorf("%w: jsonrpc status %d", ErrUnavailable, response.StatusCode)
	}

	var envelope rpcResponse
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return nil, fmt.Errorf("%w: decode jsonrpc envelope", ErrUnexpectedResponse)
	}
	if envelope.Error != nil {
		message := strings.TrimSpace(envelope.Error.Message)
		if message == "" {
			message = "unknown Odoo JSON-RPC error"
		}
		return nil, fmt.Errorf("%w: odoo rpc %d: %s", ErrUnavailable, envelope.Error.Code, message)
	}
	if len(envelope.Result) == 0 {
		return nil, ErrUnexpectedResponse
	}
	return envelope.Result, nil
}
