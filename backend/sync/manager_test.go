package sync

import (
	"path/filepath"
	"testing"
	"time"
)

func TestIsWatchableDBFileRespectsWatchWAL(t *testing.T) {
	dbPath := filepath.Join("/tmp", "message_0.db")
	walPath := dbPath + "-wal"
	shmPath := dbPath + "-shm"

	if !IsWatchableDBFile(dbPath, false) {
		t.Fatal("expected db file to be watchable")
	}
	if IsWatchableDBFile(walPath, false) || IsWatchableDBFile(shmPath, false) {
		t.Fatal("expected wal/shm files to be ignored when watch_wal=false")
	}
	if !IsWatchableDBFile(walPath, true) || !IsWatchableDBFile(shmPath, true) {
		t.Fatal("expected wal/shm files to be watchable when watch_wal=true")
	}
}

func TestManagerFlushCoalescesBaseDatabaseEvents(t *testing.T) {
	manager, err := NewManager(ManagerOptions{
		Root:      t.TempDir(),
		Debounce:  200 * time.Millisecond,
		WatchWAL:  true,
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	now := time.Now().Add(-time.Second)
	base := filepath.Join(manager.opts.Root, "message", "message_0.db")

	manager.mu.Lock()
	manager.queueEventLocked(ChangeEvent{
		Path:       base,
		BaseDB:     base,
		Kind:       FileKindDB,
		Op:         ChangeOpWrite,
		DetectedAt: now,
	})
	manager.queueEventLocked(ChangeEvent{
		Path:       base + "-wal",
		BaseDB:     base,
		Kind:       FileKindWAL,
		Op:         ChangeOpWrite,
		DetectedAt: now.Add(50 * time.Millisecond),
	})
	manager.mu.Unlock()

	var revisions []Revision
	manager.opts.OnRevision = func(revision Revision) {
		revisions = append(revisions, revision)
	}

	manager.flush(time.Now())

	if len(revisions) != 1 {
		t.Fatalf("expected 1 revision, got %d", len(revisions))
	}
	if len(revisions[0].ChangedDatabases) != 1 || revisions[0].ChangedDatabases[0] != base {
		t.Fatalf("unexpected changed databases: %#v", revisions[0].ChangedDatabases)
	}
	if len(revisions[0].Events) != 2 {
		t.Fatalf("expected both db and wal events to be coalesced, got %#v", revisions[0].Events)
	}
}
