package runtime

import (
	"context"
	"testing"
	"time"
)

func TestStoreUpdateStatusPublishesEvent(t *testing.T) {
	store := NewStore(StoreOptions{EngineType: "welink", EventBuffer: 4})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := store.Hub().Subscribe(ctx)
	store.UpdateStatus(func(status *RuntimeStatus) {
		status.IsInitialized = true
		status.TotalCached = 12
		status.DataRevision = 1
	})

	select {
	case evt := <-ch:
		if evt.Type != EventRuntimeStatusChanged {
			t.Fatalf("expected status changed event, got %s", evt.Type)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatalf("expected status event")
	}
}

func TestStoreTaskAndLogCaps(t *testing.T) {
	store := NewStore(StoreOptions{
		MaxTaskRecords: 2,
		MaxLogRecords:  2,
	})

	store.UpsertTask(TaskRecord{ID: "t1", Kind: TaskKindCustom, Status: TaskStatusPending})
	store.UpsertTask(TaskRecord{ID: "t2", Kind: TaskKindCustom, Status: TaskStatusPending})
	store.UpsertTask(TaskRecord{ID: "t3", Kind: TaskKindCustom, Status: TaskStatusPending})

	tasks := store.Tasks(10)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks retained, got %d", len(tasks))
	}
	if _, ok := store.Task("t1"); ok {
		t.Fatalf("expected oldest task trimmed")
	}

	store.AppendLog("info", "test", "one", nil)
	store.AppendLog("info", "test", "two", nil)
	store.AppendLog("info", "test", "three", nil)
	logs := store.Logs(10)
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs retained, got %d", len(logs))
	}
	if logs[0].Message != "three" || logs[1].Message != "two" {
		t.Fatalf("unexpected log order: %+v", logs)
	}
}
