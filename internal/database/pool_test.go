package database

import (
	"testing"

	"github.com/rob121/cannon/internal/config"
)

func TestConfigurePoolSQLite(t *testing.T) {
	db, err := Open(config.DatabaseConfig{
		Type: "sqlite",
		DSN:  t.TempDir() + "/pool.sqlite",
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if got := sqlDB.Stats().MaxOpenConnections; got != SQLiteMaxOpenConns {
		t.Fatalf("expected max open conns %d, got %d", SQLiteMaxOpenConns, got)
	}
}
