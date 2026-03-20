package runtime

import "time"

const (
	TaskStatusPending   = "pending"
	TaskStatusRunning   = "running"
	TaskStatusSucceeded = "succeeded"
	TaskStatusFailed    = "failed"
	TaskStatusCanceled  = "canceled"
)

const (
	TaskKindDecrypt = "decrypt"
	TaskKindIngest  = "ingest"
	TaskKindSync    = "sync"
	TaskKindReindex = "reindex"
	TaskKindCustom  = "custom"
)

const (
	EventRuntimeStatusChanged = "runtime.status.changed"
	EventTaskUpserted         = "runtime.task.upserted"
	EventLogAppended          = "runtime.log.appended"
)

type RuntimeStatus struct {
	EngineType       string  `json:"engine_type"`
	DeploymentTarget string  `json:"deployment_target,omitempty"`
	DecryptState     string  `json:"decrypt_state"`
	IsIndexing       bool    `json:"is_indexing"`
	IsInitialized    bool    `json:"is_initialized"`
	TotalCached      int     `json:"total_cached"`
	DataRevision     int64   `json:"data_revision"`
	LastDecryptAt    *string `json:"last_decrypt_at,omitempty"`
	LastReindexAt    *string `json:"last_reindex_at,omitempty"`
	LastMessageAt    *string `json:"last_message_at,omitempty"`
	LastSNSAt        *string `json:"last_sns_at,omitempty"`
	PendingChanges   int     `json:"pending_changes"`
	LastError        string  `json:"last_error,omitempty"`
	LastChangeReason string  `json:"last_change_reason,omitempty"`
	UpdatedAt        string  `json:"updated_at"`
}

type RuntimeEvent struct {
	ID        uint64    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload,omitempty"`
}

type TaskRecord struct {
	ID         string            `json:"id"`
	Kind       string            `json:"kind"`
	Status     string            `json:"status"`
	Title      string            `json:"title,omitempty"`
	Detail     string            `json:"detail,omitempty"`
	Message    string            `json:"message,omitempty"`
	Error      string            `json:"error,omitempty"`
	Progress   float64           `json:"progress,omitempty"`
	CreatedAt  string            `json:"created_at"`
	StartedAt  string            `json:"started_at,omitempty"`
	FinishedAt string            `json:"finished_at,omitempty"`
	UpdatedAt  string            `json:"updated_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type RuntimeLogRecord struct {
	ID        uint64            `json:"id"`
	Level     string            `json:"level"`
	Source    string            `json:"source"`
	Message   string            `json:"message"`
	Timestamp string            `json:"timestamp"`
	Fields    map[string]string `json:"fields,omitempty"`
}
