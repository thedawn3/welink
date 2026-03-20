package ingest

import "time"

// TaskState describes the current lifecycle state of a decrypt task.
type TaskState string

const (
	TaskStateIdle      TaskState = "idle"
	TaskStateRunning   TaskState = "running"
	TaskStateStopping  TaskState = "stopping"
	TaskStateStopped   TaskState = "stopped"
	TaskStateSucceeded TaskState = "succeeded"
	TaskStateFailed    TaskState = "failed"
)

// StartOptions configures a command-driven decrypt task.
type StartOptions struct {
	ID            string
	Name          string
	Command       string
	Args          []string
	WorkDir       string
	Env           map[string]string
	MaxLogEntries int
	BuiltinStage  *StageOptions
}

// TaskStatus is a snapshot-friendly status model for one task.
type TaskStatus struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Command      string    `json:"command"`
	Args         []string  `json:"args"`
	WorkDir      string    `json:"work_dir"`
	PID          int       `json:"pid"`
	State        TaskState `json:"state"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
	LastUpdateAt time.Time `json:"last_update_at"`
	ExitCode     *int      `json:"exit_code,omitempty"`
	Error        string    `json:"error,omitempty"`
}

// LogEntry is a sequence-addressable log record for one task.
type LogEntry struct {
	Seq     int64     `json:"seq"`
	Time    time.Time `json:"time"`
	Stream  string    `json:"stream"` // system | stdout | stderr
	Message string    `json:"message"`
}
