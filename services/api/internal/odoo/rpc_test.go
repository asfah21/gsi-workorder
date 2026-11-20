package odoo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServiceLoginSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jsonrpc" {
			t.Fatalf("path = %q, want /jsonrpc", r.URL.Path)
		}
		var request rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Params.Service != "common" || request.Params.Method != "login" {
			t.Fatalf("params = %#v", request.Params)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":6333}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	uid, err := client.ServiceLogin(context.Background(), "odoo_new", "svc", "secret")
	if err != nil {
		t.Fatalf("ServiceLogin() error = %v", err)
	}
	if uid != 6333 {
		t.Fatalf("uid = %d, want 6333", uid)
	}
}

func TestServiceLoginRejectsFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":false}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	if _, err := client.ServiceLogin(context.Background(), "odoo_new", "svc", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("ServiceLogin() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestSearchReadSendsFieldsAndContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var request struct {
			Params struct {
				Service string            `json:"service"`
				Method  string            `json:"method"`
				Args    []json.RawMessage `json:"args"`
			} `json:"params"`
		}
		if err := json.Unmarshal(body, &request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Params.Service != "object" || request.Params.Method != "execute_kw" {
			t.Fatalf("params = %#v", request.Params)
		}
		if len(request.Params.Args) != 7 {
			t.Fatalf("args count = %d, want 7", len(request.Params.Args))
		}
		if string(request.Params.Args[3]) != `"res.users"` || string(request.Params.Args[4]) != `"search_read"` {
			t.Fatalf("model/method args = %s/%s", request.Params.Args[3], request.Params.Args[4])
		}
		if string(request.Params.Args[5]) != `[[["share","=",false]]]` {
			t.Fatalf("domain args = %s", request.Params.Args[5])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":[{"id":6333,"login":"2306001","active":true}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	rows, err := client.SearchRead(
		context.Background(),
		"odoo_new",
		6333,
		"secret",
		"res.users",
		[]any{[]any{"share", "=", false}},
		[]string{"id", "login", "active"},
		map[string]any{"active_test": false},
	)
	if err != nil {
		t.Fatalf("SearchRead() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
}

func TestRPCErrorMapsToUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":200,"message":"access denied"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	_, err := client.SearchRead(
		context.Background(), "odoo_new", 1, "secret", "res.users",
		[]any{}, []string{"id"}, nil,
	)
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("SearchRead() error = %v, want ErrUnavailable", err)
	}
}
