package odoo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLoginSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/login" {
			t.Fatalf("path = %q, want /api/auth/login", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"status":"success","data":{"token":"odoo-token"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL+"/", time.Second)
	if err := client.Login(context.Background(), "12345", "secret"); err != nil {
		t.Fatalf("Login() error = %v", err)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"invalid"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	err := client.Login(context.Background(), "12345", "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
	if strings.Contains(err.Error(), "wrong-password") {
		t.Fatal("error leaked the password")
	}
}

func TestLoginUnexpectedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	err := client.Login(context.Background(), "12345", "secret")
	if !errors.Is(err, ErrUnexpectedResponse) {
		t.Fatalf("Login() error = %v, want ErrUnexpectedResponse", err)
	}
}

func TestLoginServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	err := client.Login(context.Background(), "12345", "secret")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Login() error = %v, want ErrUnavailable", err)
	}
}

func TestLoginMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	err := client.Login(context.Background(), "12345", "secret")
	if !errors.Is(err, ErrUnexpectedResponse) {
		t.Fatalf("Login() error = %v, want ErrUnexpectedResponse", err)
	}
}
