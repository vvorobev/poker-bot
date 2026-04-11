package storage

import (
	"path/filepath"
	"testing"
)

func TestOpen(t *testing.T) {
	// WAL mode requires a file-based DB (not :memory:)
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	var mode string
	if err := db.QueryRow("PRAGMA journal_mode;").Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("expected journal_mode=wal, got %q", mode)
	}

	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys;").Scan(&fk); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("expected foreign_keys=1, got %d", fk)
	}
}

func TestOpenMemory(t *testing.T) {
	// :memory: open must succeed (WAL not applicable there)
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) failed: %v", err)
	}
	db.Close()
}

func TestRunMigrations(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// All 4 tables must exist
	want := map[string]bool{
		"players":          false,
		"games":            false,
		"game_participants": false,
		"settlements":      false,
	}
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table';")
	if err != nil {
		t.Fatalf("sqlite_master query: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("table %q not found", name)
		}
	}

	// Idempotent: second run must not fail
	if err := RunMigrations(db); err != nil {
		t.Fatalf("second RunMigrations failed: %v", err)
	}
}

func TestForeignKeyEnforcement(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Insert game_participant with non-existent game_id must fail
	_, err = db.Exec(
		`INSERT INTO game_participants (game_id, player_id) VALUES (9999, 9999)`,
	)
	if err == nil {
		t.Error("expected FK error, got nil")
	}
}
