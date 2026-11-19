package db

import (
	"net/url"
	"strings"
)

// sanitizeDatabaseError removes credentials from errors that may contain the
// raw DATABASE_URL. pgx and golang-migrate can echo the connection string in
// parse errors, so errors must be redacted before they reach logs.
func sanitizeDatabaseError(err error, databaseURL string) string {
	if err == nil {
		return ""
	}
	if databaseURL == "" {
		return err.Error()
	}

	message := strings.ReplaceAll(err.Error(), databaseURL, redactDatabaseURL(databaseURL))
	if parsed, parseErr := url.Parse(databaseURL); parseErr == nil && parsed.User != nil {
		if password, ok := parsed.User.Password(); ok && password != "" {
			message = strings.ReplaceAll(message, password, "redacted")
		}
	}
	return message
}

func redactDatabaseURL(databaseURL string) string {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return "<invalid DATABASE_URL>"
	}
	if parsed.User != nil {
		if _, ok := parsed.User.Password(); ok {
			parsed.User = url.UserPassword(parsed.User.Username(), "redacted")
		}
	}
	return parsed.String()
}
