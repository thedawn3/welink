package runtime

import (
	"fmt"
	"slices"
	"sync"
	"time"
)

const (
	defaultMaxTaskRecords = 200
	defaultMaxLogRecords  = 1000
)

type StoreOptions struct {
	EngineType     string
	MaxTaskRecords int
	MaxLogRecords  int
	EventBuffer    int
	Hub            *EventHub
}

type Store struct {
	mu sync.RWMutex

	status RuntimeStatus

	tasks      map[string]TaskRecord
	taskOrder  []string
	maxTaskLen int

	logs      []RuntimeLogRecord
	maxLogLen int
	nextLogID uint64

	hub *EventHub
}

func NewStore(opts StoreOptions) *Store {
	maxTaskLen := opts.MaxTaskRecords
	if maxTaskLen <= 0 {
		maxTaskLen = defaultMaxTaskRecords
	}

	maxLogLen := opts.MaxLogRecords
	if maxLogLen <= 0 {
		maxLogLen = defaultMaxLogRecords
	}

	hub := opts.Hub
	if hub == nil {
		hub = NewEventHub(opts.EventBuffer)
	}

	now := formatNow()
	status := RuntimeStatus{
		EngineType:       opts.EngineType,
		DeploymentTarget: "host",
		DecryptState:     "idle",
		IsIndexing:       false,
		IsInitialized:    false,
		UpdatedAt:        now,
	}
	if status.EngineType == "" {
		status.EngineType = "welink"
	}

	return &Store{
		status:     status,
		tasks:      make(map[string]TaskRecord),
		taskOrder:  make([]string, 0, maxTaskLen),
		maxTaskLen: maxTaskLen,
		logs:       make([]RuntimeLogRecord, 0, maxLogLen),
		maxLogLen:  maxLogLen,
		hub:        hub,
	}
}

func (s *Store) Hub() *EventHub {
	return s.hub
}

func (s *Store) Status() RuntimeStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *Store) SetStatus(status RuntimeStatus) {
	s.mu.Lock()
	status.UpdatedAt = formatNow()
	s.status = status
	s.mu.Unlock()
	s.hub.Publish(EventRuntimeStatusChanged, status)
}

func (s *Store) UpdateStatus(updateFn func(*RuntimeStatus)) RuntimeStatus {
	s.mu.Lock()
	updateFn(&s.status)
	s.status.UpdatedAt = formatNow()
	status := s.status
	s.mu.Unlock()
	s.hub.Publish(EventRuntimeStatusChanged, status)
	return status
}

func (s *Store) UpsertTask(task TaskRecord) TaskRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := formatNow()
	if task.ID == "" {
		task.ID = fmt.Sprintf("task-%d", time.Now().UnixNano())
	}
	if task.Status == "" {
		task.Status = TaskStatusPending
	}
	if task.Kind == "" {
		task.Kind = TaskKindCustom
	}
	if task.CreatedAt == "" {
		task.CreatedAt = now
	}
	task.UpdatedAt = now
	if task.Metadata != nil {
		task.Metadata = cloneMap(task.Metadata)
	}

	if _, exists := s.tasks[task.ID]; !exists {
		s.taskOrder = append(s.taskOrder, task.ID)
	}
	s.tasks[task.ID] = task
	s.compactTasksLocked()

	s.hub.Publish(EventTaskUpserted, task)
	return task
}

func (s *Store) StartTask(id, kind, title, detail string) TaskRecord {
	now := formatNow()
	return s.UpsertTask(TaskRecord{
		ID:        id,
		Kind:      kind,
		Title:     title,
		Detail:    detail,
		Status:    TaskStatusRunning,
		StartedAt: now,
	})
}

func (s *Store) FinishTask(id, status, message, errMsg string) (TaskRecord, bool) {
	s.mu.Lock()
	existing, ok := s.tasks[id]
	if !ok {
		s.mu.Unlock()
		return TaskRecord{}, false
	}
	now := formatNow()
	existing.Status = status
	existing.Message = message
	existing.Error = errMsg
	existing.FinishedAt = now
	existing.UpdatedAt = now
	s.tasks[id] = existing
	s.mu.Unlock()

	s.hub.Publish(EventTaskUpserted, existing)
	return existing, true
}

func (s *Store) Tasks(limit int) []TaskRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.taskOrder) {
		limit = len(s.taskOrder)
	}
	result := make([]TaskRecord, 0, limit)
	for i := len(s.taskOrder) - 1; i >= 0 && len(result) < limit; i-- {
		id := s.taskOrder[i]
		task, ok := s.tasks[id]
		if !ok {
			continue
		}
		if task.Metadata != nil {
			task.Metadata = cloneMap(task.Metadata)
		}
		result = append(result, task)
	}
	return result
}

func (s *Store) Task(id string) (TaskRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	if !ok {
		return TaskRecord{}, false
	}
	if task.Metadata != nil {
		task.Metadata = cloneMap(task.Metadata)
	}
	return task, true
}

func (s *Store) AppendLog(level, source, message string, fields map[string]string) RuntimeLogRecord {
	s.mu.Lock()
	s.nextLogID++
	entry := RuntimeLogRecord{
		ID:        s.nextLogID,
		Level:     level,
		Source:    source,
		Message:   message,
		Timestamp: formatNow(),
	}
	if fields != nil {
		entry.Fields = cloneMap(fields)
	}
	s.logs = append(s.logs, entry)
	if len(s.logs) > s.maxLogLen {
		s.logs = slices.Clone(s.logs[len(s.logs)-s.maxLogLen:])
	}
	s.mu.Unlock()

	s.hub.Publish(EventLogAppended, entry)
	return entry
}

func (s *Store) Logs(limit int) []RuntimeLogRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.logs) {
		limit = len(s.logs)
	}
	result := make([]RuntimeLogRecord, 0, limit)
	for i := len(s.logs) - 1; i >= 0 && len(result) < limit; i-- {
		entry := s.logs[i]
		if entry.Fields != nil {
			entry.Fields = cloneMap(entry.Fields)
		}
		result = append(result, entry)
	}
	return result
}

func (s *Store) compactTasksLocked() {
	if len(s.taskOrder) <= s.maxTaskLen {
		return
	}
	start := len(s.taskOrder) - s.maxTaskLen
	toDelete := s.taskOrder[:start]
	s.taskOrder = slices.Clone(s.taskOrder[start:])
	for _, id := range toDelete {
		delete(s.tasks, id)
	}
}

func cloneMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func formatNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
