package database

import (
	"strings"
	"testing"
)

func TestSQLiteDSN(t *testing.T) {
	dsn := SQLiteDSN("/tmp/example.sqlite")
	if !strings.Contains(dsn, "file:/tmp/example.sqlite") {
		t.Fatalf("unexpected path in dsn: %q", dsn)
	}
	for _, part := range []string{"_journal_mode=WAL", "_busy_timeout=5000", "_foreign_keys=on"} {
		if !strings.Contains(dsn, part) {
			t.Fatalf("missing %s in %q", part, dsn)
		}
	}
}

func TestSQLiteDSNPreservesFileURI(t *testing.T) {
	dsn := SQLiteDSN("file:/tmp/example.sqlite")
	for _, part := range []string{"_journal_mode=WAL", "_busy_timeout=5000", "_foreign_keys=on"} {
		if !strings.Contains(dsn, part) {
			t.Fatalf("missing %s in %q", part, dsn)
		}
	}
}
