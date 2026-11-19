package db

import (
	"errors"
	"strings"
	"testing"
)

func TestSanitizeDatabaseErrorRedactsURLAndPassword(t *testing.T) {
	databaseURL := "postgres://gsi:supersecret@db:5432/workorder?sslmode=disable"
	raw := errors.New("failed to parse as URL (" + databaseURL + ")")

	got := sanitizeDatabaseError(raw, databaseURL)
	if strings.Contains(got, "supersecret") {
		t.Fatalf("sanitized error still contains the password: %s", got)
	}
	if !strings.Contains(got, "redacted") {
		t.Fatalf("sanitized error does not contain a redaction marker: %s", got)
	}
}

func TestSanitizeDatabaseErrorRedactsPasswordOnly(t *testing.T) {
	got := sanitizeDatabaseError(errors.New("password=supersecret rejected"), "postgres://gsi:supersecret@db:5432/workorder")
	if strings.Contains(got, "supersecret") {
		t.Fatalf("sanitized error still contains the password: %s", got)
	}
}

func TestSanitizeDatabaseErrorRedactsUnparseableURL(t *testing.T) {
	databaseURL := "postgres://gsi:supersecret@db:notaport/workorder"
	raw := errors.New("failed to parse \"" + databaseURL + "\": invalid port \":notaport\" after host")

	got := sanitizeDatabaseError(raw, databaseURL)
	if strings.Contains(got, "supersecret") {
		t.Fatalf("sanitized error still contains the password: %s", got)
	}
}
