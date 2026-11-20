package auth

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestTokenPairParsesByType(t *testing.T) {
	now := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	manager := newTokenManagerWithClock("0123456789abcdef0123456789abcdef", func() time.Time { return now })

	pair, err := manager.IssuePair(42, RolePlanner)
	if err != nil {
		t.Fatalf("IssuePair() error = %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected non-empty access and refresh tokens")
	}
	if pair.ExpiresIn != int64(AccessTokenTTL/time.Second) {
		t.Fatalf("ExpiresIn = %d", pair.ExpiresIn)
	}

	accessClaims, err := manager.ParseAccess(pair.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccess() error = %v", err)
	}
	if accessClaims.Subject != "42" || accessClaims.Role != RolePlanner {
		t.Fatalf("access claims = %#v", accessClaims)
	}
	if _, err := manager.ParseRefresh(pair.AccessToken); err == nil {
		t.Fatal("ParseRefresh() accepted an access token")
	}

	refreshClaims, err := manager.ParseRefresh(pair.RefreshToken)
	if err != nil {
		t.Fatalf("ParseRefresh() error = %v", err)
	}
	if refreshClaims.Subject != strconv.FormatInt(42, 10) {
		t.Fatalf("refresh subject = %q", refreshClaims.Subject)
	}
	if _, err := manager.ParseAccess(pair.RefreshToken); err == nil {
		t.Fatal("ParseAccess() accepted a refresh token")
	}
}

func TestAccessTokenExpires(t *testing.T) {
	now := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	manager := newTokenManagerWithClock("0123456789abcdef0123456789abcdef", func() time.Time { return now })

	pair, err := manager.IssuePair(7, RoleAdmin)
	if err != nil {
		t.Fatalf("IssuePair() error = %v", err)
	}

	now = now.Add(AccessTokenTTL + time.Second)
	if _, err := manager.ParseAccess(pair.AccessToken); err == nil {
		t.Fatal("ParseAccess() accepted an expired token")
	}
}

func TestTamperedTokenRejected(t *testing.T) {
	manager := NewTokenManager("0123456789abcdef0123456789abcdef")
	pair, err := manager.IssuePair(7, RoleAdmin)
	if err != nil {
		t.Fatalf("IssuePair() error = %v", err)
	}

	parts := strings.Split(pair.AccessToken, ".")
	if len(parts) != 3 {
		t.Fatalf("token parts = %d, want 3", len(parts))
	}
	signature := []byte(parts[2])
	replacement := byte('A')
	if signature[0] == replacement {
		replacement = 'B'
	}
	signature[0] = replacement
	parts[2] = string(signature)
	tampered := strings.Join(parts, ".")
	if _, err := manager.ParseAccess(tampered); err == nil {
		t.Fatal("ParseAccess() accepted a tampered token")
	}
}
