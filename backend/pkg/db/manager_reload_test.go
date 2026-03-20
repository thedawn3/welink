package db

import (
	"testing"

	"welink/backend/pkg/seed"
)

func TestNewDBManagerAllowsMissingDataOnStartup(t *testing.T) {
	dir := t.TempDir()

	mgr, err := NewDBManager(dir)
	if err != nil {
		t.Fatalf("new db manager: %v", err)
	}
	if mgr.Ready() {
		t.Fatal("expected manager to start not-ready when data files are missing")
	}
}

func TestDBManagerReloadTransitionsToReady(t *testing.T) {
	dir := t.TempDir()

	mgr, err := NewDBManager(dir)
	if err != nil {
		t.Fatalf("new db manager: %v", err)
	}
	if mgr.Ready() {
		t.Fatal("expected manager to start not-ready")
	}

	if err := seed.Generate(dir); err != nil {
		t.Fatalf("generate seed dbs: %v", err)
	}
	if err := mgr.Reload(dir); err != nil {
		t.Fatalf("reload db manager: %v", err)
	}
	if !mgr.Ready() {
		t.Fatal("expected manager to become ready after reload")
	}
}
