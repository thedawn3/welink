package sync

import "time"

// FileKind tags the database file type detected by the watcher.
type FileKind string

const (
	FileKindDB  FileKind = "db"
	FileKindWAL FileKind = "wal"
	FileKindSHM FileKind = "shm"
)

// ChangeOp is the type of detected filesystem change.
type ChangeOp string

const (
	ChangeOpCreate ChangeOp = "create"
	ChangeOpWrite  ChangeOp = "write"
	ChangeOpDelete ChangeOp = "delete"
)

// ChangeEvent is one normalized DB/WAL/SHM file-system event.
type ChangeEvent struct {
	Path       string    `json:"path"`
	BaseDB     string    `json:"base_db"`
	Kind       FileKind  `json:"kind"`
	Op         ChangeOp  `json:"op"`
	Size       int64     `json:"size"`
	ModTime    time.Time `json:"mod_time"`
	DetectedAt time.Time `json:"detected_at"`
}

// Revision summarizes a debounced change batch that can trigger reindex.
type Revision struct {
	Seq              int64         `json:"seq"`
	ID               string        `json:"id"`
	Root             string        `json:"root"`
	OccurredAt       time.Time     `json:"occurred_at"`
	ChangedDatabases []string      `json:"changed_databases"`
	Events           []ChangeEvent `json:"events"`
}

// Status reports watcher runtime info for diagnostics.
type Status struct {
	Running         bool      `json:"running"`
	KnownFiles      int       `json:"known_files"`
	PendingBatches  int       `json:"pending_batches"`
	LastScanAt      time.Time `json:"last_scan_at"`
	LastRevisionAt  time.Time `json:"last_revision_at"`
	LastRevisionSeq int64     `json:"last_revision_seq"`
}

// ManagerOptions configures file scanning and debounce behavior.
type ManagerOptions struct {
	Root         string
	PollInterval time.Duration
	Debounce     time.Duration
	WatchWAL     bool
	IgnoreDirs   []string
	OnRevision   func(Revision)
	OnError      func(error)
}
