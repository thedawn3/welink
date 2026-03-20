package ingest

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"sync"
	"time"
)

const (
	defaultMaxTaskLogs = 2000
	defaultStopGrace   = 5 * time.Second
)

type taskRun struct {
	mu      sync.RWMutex
	status  TaskStatus
	logs    []LogEntry
	nextSeq int64
	maxLogs int
	cancel  context.CancelFunc
	cmd     *exec.Cmd
	done    chan struct{}
}

// Orchestrator manages command-driven decrypt tasks and their logs.
type Orchestrator struct {
	mu             sync.RWMutex
	tasks          map[string]*taskRun
	defaultMaxLogs int
}

func NewOrchestrator(defaultMaxLogs int) *Orchestrator {
	if defaultMaxLogs <= 0 {
		defaultMaxLogs = defaultMaxTaskLogs
	}
	return &Orchestrator{
		tasks:          make(map[string]*taskRun),
		defaultMaxLogs: defaultMaxLogs,
	}
}

func (o *Orchestrator) StartTask(opts StartOptions) (string, error) {
	if opts.Command == "" {
		return "", errors.New("command is required")
	}

	taskID := opts.ID
	if taskID == "" {
		taskID = fmt.Sprintf("task-%d", time.Now().UnixNano())
	}
	name := opts.Name
	if name == "" {
		name = "decrypt"
	}

	maxLogs := opts.MaxLogEntries
	if maxLogs <= 0 {
		maxLogs = o.defaultMaxLogs
	}

	o.mu.Lock()
	if _, exists := o.tasks[taskID]; exists {
		o.mu.Unlock()
		return "", fmt.Errorf("task %q already exists", taskID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, opts.Command, opts.Args...)
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}
	cmd.Env = mergeEnv(opts.Env)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		o.mu.Unlock()
		return "", fmt.Errorf("open stdout pipe failed: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		o.mu.Unlock()
		return "", fmt.Errorf("open stderr pipe failed: %w", err)
	}

	now := time.Now()
	task := &taskRun{
		status: TaskStatus{
			ID:           taskID,
			Name:         name,
			Command:      opts.Command,
			Args:         append([]string(nil), opts.Args...),
			WorkDir:      opts.WorkDir,
			State:        TaskStateRunning,
			StartedAt:    now,
			LastUpdateAt: now,
		},
		maxLogs: maxLogs,
		cancel:  cancel,
		cmd:     cmd,
		done:    make(chan struct{}),
	}
	o.tasks[taskID] = task
	o.mu.Unlock()

	task.appendLog("system", fmt.Sprintf("starting command: %s %v", opts.Command, opts.Args))
	if err := cmd.Start(); err != nil {
		task.finish(TaskStateFailed, err, exitCodeFromErr(err))
		return taskID, err
	}
	task.setPID(cmd.Process.Pid)
	task.appendLog("system", fmt.Sprintf("process started with pid=%d", cmd.Process.Pid))

	go streamPipe(task, "stdout", stdoutPipe)
	go streamPipe(task, "stderr", stderrPipe)
	go o.waitTask(taskID, task)

	return taskID, nil
}

func (o *Orchestrator) waitTask(taskID string, task *taskRun) {
	err := task.cmd.Wait()
	state := TaskStateSucceeded
	if err != nil {
		task.mu.RLock()
		currentState := task.status.State
		task.mu.RUnlock()
		if currentState == TaskStateStopping {
			state = TaskStateStopped
		} else {
			state = TaskStateFailed
		}
	}
	task.finish(state, err, exitCodeFromErr(err))
	task.appendLog("system", "task finished")
	_ = taskID
}

func streamPipe(task *taskRun, stream string, reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		task.appendLog(stream, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		task.appendLog("system", fmt.Sprintf("%s read error: %v", stream, err))
	}
}

func (o *Orchestrator) StopTask(taskID string) error {
	return o.StopTaskWithGrace(taskID, defaultStopGrace)
}

func (o *Orchestrator) StopTaskWithGrace(taskID string, grace time.Duration) error {
	task, ok := o.getTask(taskID)
	if !ok {
		return fmt.Errorf("task %q not found", taskID)
	}
	if grace <= 0 {
		grace = defaultStopGrace
	}

	task.mu.Lock()
	if task.status.State != TaskStateRunning {
		task.mu.Unlock()
		return nil
	}
	task.status.State = TaskStateStopping
	task.status.LastUpdateAt = time.Now()
	task.mu.Unlock()
	task.appendLog("system", "stop requested")

	task.cancel()

	select {
	case <-task.done:
		return nil
	case <-time.After(grace):
		task.appendLog("system", "grace timeout reached, killing process")
		if task.cmd != nil && task.cmd.Process != nil {
			_ = task.cmd.Process.Kill()
		}
		<-task.done
		return nil
	}
}

func (o *Orchestrator) Status(taskID string) (TaskStatus, bool) {
	task, ok := o.getTask(taskID)
	if !ok {
		return TaskStatus{}, false
	}
	return task.snapshotStatus(), true
}

func (o *Orchestrator) ListStatuses() []TaskStatus {
	o.mu.RLock()
	defer o.mu.RUnlock()

	out := make([]TaskStatus, 0, len(o.tasks))
	for _, task := range o.tasks {
		out = append(out, task.snapshotStatus())
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].StartedAt.After(out[j].StartedAt)
	})
	return out
}

func (o *Orchestrator) Logs(taskID string, afterSeq int64, limit int) ([]LogEntry, bool) {
	task, ok := o.getTask(taskID)
	if !ok {
		return nil, false
	}
	return task.logsAfter(afterSeq, limit), true
}

func (o *Orchestrator) WaitTask(taskID string, timeout time.Duration) (TaskStatus, error) {
	task, ok := o.getTask(taskID)
	if !ok {
		return TaskStatus{}, fmt.Errorf("task %q not found", taskID)
	}
	if timeout <= 0 {
		<-task.done
		return task.snapshotStatus(), nil
	}
	select {
	case <-task.done:
		return task.snapshotStatus(), nil
	case <-time.After(timeout):
		return task.snapshotStatus(), fmt.Errorf("wait task %q timeout", taskID)
	}
}

func (o *Orchestrator) RemoveTask(taskID string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	task, ok := o.tasks[taskID]
	if !ok {
		return fmt.Errorf("task %q not found", taskID)
	}
	task.mu.RLock()
	running := task.status.State == TaskStateRunning || task.status.State == TaskStateStopping
	task.mu.RUnlock()
	if running {
		return fmt.Errorf("task %q is still running", taskID)
	}
	delete(o.tasks, taskID)
	return nil
}

func (o *Orchestrator) getTask(taskID string) (*taskRun, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	task, ok := o.tasks[taskID]
	return task, ok
}

func (t *taskRun) setPID(pid int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status.PID = pid
	t.status.LastUpdateAt = time.Now()
}

func (t *taskRun) finish(state TaskState, err error, exitCode *int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.status.State == TaskStateSucceeded || t.status.State == TaskStateFailed || t.status.State == TaskStateStopped {
		return
	}
	t.status.State = state
	t.status.FinishedAt = time.Now()
	t.status.LastUpdateAt = t.status.FinishedAt
	t.status.ExitCode = exitCode
	if err != nil {
		t.status.Error = err.Error()
	}
	close(t.done)
}

func (t *taskRun) appendLog(stream, message string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.nextSeq++
	t.logs = append(t.logs, LogEntry{
		Seq:     t.nextSeq,
		Time:    time.Now(),
		Stream:  stream,
		Message: message,
	})
	if len(t.logs) > t.maxLogs {
		cut := len(t.logs) - t.maxLogs
		t.logs = append([]LogEntry(nil), t.logs[cut:]...)
	}
	t.status.LastUpdateAt = time.Now()
}

func (t *taskRun) snapshotStatus() TaskStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := t.status
	out.Args = append([]string(nil), t.status.Args...)
	return out
}

func (t *taskRun) logsAfter(afterSeq int64, limit int) []LogEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if limit <= 0 {
		limit = 200
	}
	out := make([]LogEntry, 0, limit)
	for _, entry := range t.logs {
		if entry.Seq <= afterSeq {
			continue
		}
		out = append(out, entry)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func mergeEnv(extra map[string]string) []string {
	env := os.Environ()
	if len(extra) == 0 {
		return env
	}

	seen := make(map[string]int, len(env))
	for idx, item := range env {
		eq := -1
		for i, ch := range item {
			if ch == '=' {
				eq = i
				break
			}
		}
		if eq <= 0 {
			continue
		}
		seen[item[:eq]] = idx
	}

	for key, value := range extra {
		pair := key + "=" + value
		if idx, ok := seen[key]; ok {
			env[idx] = pair
			continue
		}
		env = append(env, pair)
	}
	return env
}

func exitCodeFromErr(err error) *int {
	if err == nil {
		code := 0
		return &code
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code := exitErr.ExitCode()
		return &code
	}
	return nil
}
