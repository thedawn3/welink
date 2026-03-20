package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultPollInterval = 1 * time.Second
	defaultDebounce     = 1500 * time.Millisecond
)

type fileSnapshot struct {
	size    int64
	modTime time.Time
}

type pendingBatch struct {
	firstSeen time.Time
	lastSeen  time.Time
	events    []ChangeEvent
}

// Manager scans DB/WAL/SHM files and emits debounced revision callbacks.
type Manager struct {
	opts ManagerOptions

	mu             sync.RWMutex
	known          map[string]fileSnapshot
	pending        map[string]*pendingBatch
	lastScanAt     time.Time
	lastRevisionAt time.Time
	revisionSeq    int64
	running        bool

	cancel contextCancel
	wg     sync.WaitGroup

	nextSubID   int
	subscribers map[int]chan Revision
}

type contextCancel interface {
	Cancel()
	Done() <-chan struct{}
}

type managerContext struct {
	done   chan struct{}
	closed bool
	mu     sync.Mutex
}

func newManagerContext() *managerContext {
	return &managerContext{
		done: make(chan struct{}),
	}
}

func (c *managerContext) Cancel() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	close(c.done)
	c.closed = true
}

func (c *managerContext) Done() <-chan struct{} {
	return c.done
}

func NewManager(opts ManagerOptions) (*Manager, error) {
	if strings.TrimSpace(opts.Root) == "" {
		return nil, fmt.Errorf("root path is required")
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = defaultPollInterval
	}
	if opts.Debounce <= 0 {
		opts.Debounce = defaultDebounce
	}
	if len(opts.IgnoreDirs) == 0 {
		opts.IgnoreDirs = []string{"fts", ".git", "node_modules"}
	}

	return &Manager{
		opts:        opts,
		known:       make(map[string]fileSnapshot),
		pending:     make(map[string]*pendingBatch),
		subscribers: make(map[int]chan Revision),
	}, nil
}

func (m *Manager) Start() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}

	ctx := newManagerContext()
	m.cancel = ctx
	m.running = true
	m.mu.Unlock()

	m.wg.Add(1)
	go m.loop(ctx)
	return nil
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil
	}
	cancel := m.cancel
	m.running = false
	m.cancel = nil
	m.mu.Unlock()

	if cancel != nil {
		cancel.Cancel()
	}
	m.wg.Wait()
	return nil
}

func (m *Manager) loop(ctx contextCancel) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.opts.PollInterval)
	defer ticker.Stop()

	m.scan()
	m.flush(time.Now())

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.scan()
			m.flush(time.Now())
		}
	}
}

func (m *Manager) scan() {
	current, err := m.snapshotDBFiles()
	if err != nil {
		m.reportError(err)
	}

	now := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	for path, snap := range current {
		old, existed := m.known[path]
		if !existed {
			m.queueEventLocked(ChangeEvent{
				Path:       path,
				BaseDB:     NormalizeDBPath(path),
				Kind:       DetectFileKind(path),
				Op:         ChangeOpCreate,
				Size:       snap.size,
				ModTime:    snap.modTime,
				DetectedAt: now,
			})
			continue
		}
		if old.size != snap.size || !old.modTime.Equal(snap.modTime) {
			m.queueEventLocked(ChangeEvent{
				Path:       path,
				BaseDB:     NormalizeDBPath(path),
				Kind:       DetectFileKind(path),
				Op:         ChangeOpWrite,
				Size:       snap.size,
				ModTime:    snap.modTime,
				DetectedAt: now,
			})
		}
	}

	for path, snap := range m.known {
		if _, ok := current[path]; ok {
			continue
		}
		m.queueEventLocked(ChangeEvent{
			Path:       path,
			BaseDB:     NormalizeDBPath(path),
			Kind:       DetectFileKind(path),
			Op:         ChangeOpDelete,
			Size:       snap.size,
			ModTime:    snap.modTime,
			DetectedAt: now,
		})
	}

	m.known = current
	m.lastScanAt = now
}

func (m *Manager) flush(now time.Time) {
	revisions := make([]Revision, 0)

	m.mu.Lock()
	for base, batch := range m.pending {
		if now.Sub(batch.lastSeen) < m.opts.Debounce {
			continue
		}

		events := append([]ChangeEvent(nil), batch.events...)
		delete(m.pending, base)

		m.revisionSeq++
		seq := m.revisionSeq
		changed := collectChangedDatabases(events)

		revision := Revision{
			Seq:              seq,
			ID:               fmt.Sprintf("rev-%d", seq),
			Root:             m.opts.Root,
			OccurredAt:       now,
			ChangedDatabases: changed,
			Events:           events,
		}
		revisions = append(revisions, revision)
		m.lastRevisionAt = now
	}
	m.mu.Unlock()

	for _, revision := range revisions {
		m.emitRevision(revision)
	}
}

func (m *Manager) emitRevision(revision Revision) {
	if m.opts.OnRevision != nil {
		m.opts.OnRevision(revision)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ch := range m.subscribers {
		select {
		case ch <- revision:
		default:
		}
	}
}

func (m *Manager) queueEventLocked(event ChangeEvent) {
	base := event.BaseDB
	batch, ok := m.pending[base]
	if !ok {
		batch = &pendingBatch{
			firstSeen: event.DetectedAt,
		}
		m.pending[base] = batch
	}
	batch.lastSeen = event.DetectedAt
	batch.events = append(batch.events, event)
}

func (m *Manager) snapshotDBFiles() (map[string]fileSnapshot, error) {
	out := make(map[string]fileSnapshot)
	err := filepath.WalkDir(m.opts.Root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		if d.IsDir() {
			if m.shouldIgnoreDir(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if !IsWatchableDBFile(path, m.opts.WatchWAL) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		out[path] = fileSnapshot{
			size:    info.Size(),
			modTime: info.ModTime(),
		}
		return nil
	})
	return out, err
}

func (m *Manager) shouldIgnoreDir(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	for _, item := range m.opts.IgnoreDirs {
		if strings.ToLower(item) == base {
			return true
		}
	}
	return false
}

func (m *Manager) reportError(err error) {
	if err == nil {
		return
	}
	if m.opts.OnError != nil {
		m.opts.OnError(err)
	}
}

func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Status{
		Running:         m.running,
		KnownFiles:      len(m.known),
		PendingBatches:  len(m.pending),
		LastScanAt:      m.lastScanAt,
		LastRevisionAt:  m.lastRevisionAt,
		LastRevisionSeq: m.revisionSeq,
	}
}

func (m *Manager) Subscribe(buffer int) (int, <-chan Revision) {
	if buffer <= 0 {
		buffer = 16
	}
	ch := make(chan Revision, buffer)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextSubID++
	id := m.nextSubID
	m.subscribers[id] = ch
	return id, ch
}

func (m *Manager) Unsubscribe(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch, ok := m.subscribers[id]
	if !ok {
		return
	}
	delete(m.subscribers, id)
	close(ch)
}

func IsWatchableDBFile(path string, watchWAL bool) bool {
	kind, ok := detectKind(path)
	if !ok {
		return false
	}
	if kind == FileKindDB {
		return true
	}
	return watchWAL && (kind == FileKindWAL || kind == FileKindSHM)
}

func DetectFileKind(path string) FileKind {
	kind, ok := detectKind(path)
	if !ok {
		return FileKindDB
	}
	return kind
}

func NormalizeDBPath(path string) string {
	switch DetectFileKind(path) {
	case FileKindWAL:
		return strings.TrimSuffix(path, "-wal")
	case FileKindSHM:
		return strings.TrimSuffix(path, "-shm")
	default:
		return path
	}
}

func detectKind(path string) (FileKind, bool) {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".db"):
		return FileKindDB, true
	case strings.HasSuffix(lower, ".db-wal"):
		return FileKindWAL, true
	case strings.HasSuffix(lower, ".db-shm"):
		return FileKindSHM, true
	default:
		return "", false
	}
}

func collectChangedDatabases(events []ChangeEvent) []string {
	seen := make(map[string]struct{}, len(events))
	out := make([]string, 0, len(events))
	for _, event := range events {
		if _, ok := seen[event.BaseDB]; ok {
			continue
		}
		seen[event.BaseDB] = struct{}{}
		out = append(out, event.BaseDB)
	}
	sort.Strings(out)
	return out
}
