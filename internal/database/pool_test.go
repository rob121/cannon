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
	if got := sqlDB.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("expected max open conns 1, got %d", got)
	}
}
