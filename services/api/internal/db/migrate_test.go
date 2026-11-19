package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4/source"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func TestFileSourceURLIsReadable(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "000001_test.up.sql")
	if err := os.WriteFile(file, []byte("SELECT 1;"), 0o600); err != nil {
		t.Fatalf("write migration fixture: %v", err)
	}

	sourceURL, err := fileSourceURL(dir)
	if err != nil {
		t.Fatalf("fileSourceURL() error = %v", err)
	}

	driver, err := source.Open(sourceURL)
	if err != nil {
		t.Fatalf("open migration source %q: %v", sourceURL, err)
	}
	defer driver.Close()

	version, err := driver.First()
	if err != nil {
		t.Fatalf("read first migration: %v", err)
	}
	if version != 1 {
		t.Fatalf("first migration version = %d, want 1", version)
	}
}
