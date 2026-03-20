package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	stdsync "sync"
	"time"

	"welink/backend/config"
	"welink/backend/ingest"
	runtimepkg "welink/backend/runtime"
	syncmgr "welink/backend/sync"

	"github.com/gin-gonic/gin"
)

type systemTaskItem struct {
	ID             string  `json:"id,omitempty"`
	Type           string  `json:"type,omitempty"`
	Status         string  `json:"status,omitempty"`
	Message        string  `json:"message,omitempty"`
	Detail         string  `json:"detail,omitempty"`
	Error          string  `json:"error,omitempty"`
	Progress       float64 `json:"progress,omitempty"`
	StartedAt      string  `json:"started_at,omitempty"`
	FinishedAt     string  `json:"finished_at,omitempty"`
	UpdatedAt      string  `json:"updated_at,omitempty"`
	WorkDir        string  `json:"work_dir,omitempty"`
	CommandSummary string  `json:"command_summary,omitempty"`
}

type systemLogItem struct {
	ID      uint64            `json:"id,omitempty"`
	Time    string            `json:"time,omitempty"`
	Level   string            `json:"level,omitempty"`
	Source  string            `json:"source,omitempty"`
	Stream  string            `json:"stream,omitempty"`
	TaskID  string            `json:"task_id,omitempty"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type directoryValidation struct {
	Path           string   `json:"path,omitempty"`
	Exists         bool     `json:"exists"`
	Readable       bool     `json:"readable"`
	Writable       bool     `json:"writable,omitempty"`
	StandardLayout bool     `json:"standard_layout"`
	HasContact     bool     `json:"has_contact"`
	HasMessage     bool     `json:"has_message"`
	Ready          bool     `json:"ready"`
	SameAsAnalysis bool     `json:"same_as_analysis,omitempty"`
	ContactDBPath  string   `json:"contact_db_path,omitempty"`
	MessageDirPath string   `json:"message_dir_path,omitempty"`
	MessageDBCount int      `json:"message_db_count,omitempty"`
	SNSDBPath      string   `json:"sns_db_path,omitempty"`
	HasSNS         bool     `json:"has_sns"`
	Issues         []string `json:"issues,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
}

type capabilityValidation struct {
	Supported bool     `json:"supported"`
	Enabled   bool     `json:"enabled"`
	Provider  string   `json:"provider,omitempty"`
	Issues    []string `json:"issues,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

type snsValidation struct {
	Detected bool     `json:"detected"`
	Ready    bool     `json:"ready"`
	DBPath   string   `json:"db_path,omitempty"`
	Issues   []string `json:"issues,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type runtimeConfigCheck struct {
	DeploymentTarget string               `json:"deployment_target"`
	Mode             string               `json:"mode"`
	AnalysisDir      directoryValidation  `json:"analysis_dir"`
	SourceDir        directoryValidation  `json:"source_dir"`
	WorkDir          directoryValidation  `json:"work_dir"`
	Decrypt          capabilityValidation `json:"decrypt"`
	Sync             capabilityValidation `json:"sync"`
	SNS              snsValidation        `json:"sns"`
	Issues           []string             `json:"issues,omitempty"`
	Warnings         []string             `json:"warnings,omitempty"`
	SuggestedActions []string             `json:"suggested_actions,omitempty"`
}

type decryptValidationResult struct {
	Options      ingest.StartOptions
	ConfigCheck  runtimeConfigCheck
	ActionLabel  string
	ActionIssues []string
}

type systemRuntime struct {
	cfg             *config.Config
	store           *runtimepkg.Store
	orchestrator    *ingest.Orchestrator
	syncManager     *syncmgr.Manager
	reindex         func(from, to int64)
	onSnapshotReady func(reason string) error

	mu             stdsync.Mutex
	currentReindex string
}

func newSystemRuntime(cfg *config.Config) *systemRuntime {
	store := runtimepkg.NewStore(runtimepkg.StoreOptions{
		EngineType:     cfg.Runtime.EngineType,
		MaxTaskRecords: cfg.Runtime.MaxTaskRecords,
		MaxLogRecords:  cfg.Runtime.MaxLogRecords,
		EventBuffer:    cfg.Sync.EventBuffer,
	})
	rt := &systemRuntime{
		cfg:          cfg,
		store:        store,
		orchestrator: ingest.NewOrchestrator(cfg.Runtime.MaxLogRecords),
	}
	rt.store.UpdateStatus(func(status *runtimepkg.RuntimeStatus) {
		status.DeploymentTarget = detectDeploymentTarget()
		rt.applyDataFreshness(status)
	})
	return rt
}

func (rt *systemRuntime) applyInitialAnalysisStatus(isIndexing, isInitialized bool, totalCached int) {
	rt.store.UpdateStatus(func(status *runtimepkg.RuntimeStatus) {
		status.IsIndexing = isIndexing
		status.IsInitialized = isInitialized
		status.TotalCached = totalCached
		rt.applyDataFreshness(status)
	})
}

func (rt *systemRuntime) onDataRevision(revision syncmgr.Revision, trigger func()) {
	rt.store.AppendLog("info", "sync", "detected change revision "+revision.ID, map[string]string{
		"changed_databases": strings.Join(revision.ChangedDatabases, ","),
	})
	rt.store.UpdateStatus(func(status *runtimepkg.RuntimeStatus) {
		status.PendingChanges++
		status.LastChangeReason = strings.Join(revision.ChangedDatabases, ",")
	})
	rt.publishEvent("runtime.revision.detected", map[string]any{
		"revision":          revision.ID,
		"changed_databases": revision.ChangedDatabases,
		"occurred_at":       revision.OccurredAt.UTC().Format(time.RFC3339),
	})
	trigger()
}

func (rt *systemRuntime) bindReindexHooks(totalCached func() int) (func(int64, int64), func(int64, int64)) {
	onStart := func(from, to int64) {
		task := rt.store.StartTask("", runtimepkg.TaskKindReindex, "Reindex chat data", fmt.Sprintf("from=%d to=%d", from, to))
		rt.mu.Lock()
		rt.currentReindex = task.ID
		rt.mu.Unlock()
		rt.store.UpdateStatus(func(status *runtimepkg.RuntimeStatus) {
			status.IsIndexing = true
			status.IsInitialized = false
			status.LastError = ""
			status.LastChangeReason = fmt.Sprintf("from=%d to=%d", from, to)
		})
		rt.publishEvent("runtime.reindex.started", map[string]any{
			"task_id": task.ID,
			"from":    from,
			"to":      to,
			"message": "reindex started",
		})
	}

	onFinish := func(from, to int64) {
		now := time.Now().UTC().Format(time.RFC3339)
		total := totalCached()
		rt.mu.Lock()
		taskID := rt.currentReindex
		rt.currentReindex = ""
		rt.mu.Unlock()
		if taskID != "" {
			rt.store.FinishTask(taskID, runtimepkg.TaskStatusSucceeded, "reindex completed", "")
		}
		rt.store.UpdateStatus(func(status *runtimepkg.RuntimeStatus) {
			status.IsIndexing = false
			status.IsInitialized = true
			status.TotalCached = total
			status.DataRevision++
			status.PendingChanges = 0
			status.LastReindexAt = &now
			rt.applyDataFreshness(status)
		})
		rt.publishEvent("runtime.reindex.finished", map[string]any{
			"from":         from,
			"to":           to,
			"total_cached": total,
			"message":      "reindex finished",
		})
	}

	return onStart, onFinish
}

func (rt *systemRuntime) startConfiguredDecrypt() {
	if !rt.cfg.Decrypt.Enabled || !rt.cfg.Decrypt.AutoStart {
		return
	}
	result, err := rt.validateDecryptStart(decryptStartRequest{
		Platform:        rt.cfg.Ingest.Platform,
		SourceDataDir:   rt.cfg.Ingest.SourceDataDir,
		AnalysisDataDir: rt.cfg.Ingest.AnalysisDataDir,
		WorkDir:         rt.cfg.Ingest.WorkDir,
		AutoRefresh:     boolPtr(rt.cfg.Sync.Enabled),
		WALEnabled:      boolPtr(rt.cfg.Sync.WatchWAL),
	})
	if err != nil {
		rt.store.AppendLog("warn", "decrypt", "decrypt auto-start skipped: "+err.Error(), nil)
		return
	}
	_, _ = rt.startDecrypt(result.Options)
}

func (rt *systemRuntime) startDecrypt(opts ingest.StartOptions) (string, error) {
	if opts.BuiltinStage != nil {
		return rt.startBuiltinStage(*opts.BuiltinStage)
	}

	taskID, err := rt.orchestrator.StartTask(opts)
	if err != nil {
		rt.store.AppendLog("error", "decrypt", err.Error(), nil)
		rt.store.UpdateStatus(func(status *runtimepkg.RuntimeStatus) {
			status.DecryptState = "error"
			status.LastError = err.Error()
		})
		return "", err
	}

	rt.store.UpdateStatus(func(status *runtimepkg.RuntimeStatus) {
		status.DecryptState = "running"
		status.LastError = ""
	})
	rt.store.AppendLog("info", "decrypt", "started decrypt task "+taskID, map[string]string{
		"command":  opts.Command,
		"work_dir": opts.WorkDir,
	})
	rt.publishEvent("runtime.decrypt.started", map[string]any{"task_id": taskID, "message": "decrypt started"})

	go func() {
		status, waitErr := rt.orchestrator.WaitTask(taskID, 0)
		if waitErr != nil {
			rt.store.AppendLog("error", "decrypt", waitErr.Error(), nil)
			rt.store.UpdateStatus(func(runtimeStatus *runtimepkg.RuntimeStatus) {
				runtimeStatus.DecryptState = "error"
				runtimeStatus.LastError = waitErr.Error()
			})
			rt.publishEvent("runtime.decrypt.failed", map[string]any{"task_id": taskID, "error": waitErr.Error(), "message": waitErr.Error()})
			return
		}

		switch status.State {
		case ingest.TaskStateSucceeded:
			if rt.onSnapshotReady != nil {
				if err := rt.onSnapshotReady("decrypt task completed"); err != nil {
					rt.store.AppendLog("error", "analysis", err.Error(), map[string]string{"task_id": taskID})
					rt.store.UpdateStatus(func(runtimeStatus *runtimepkg.RuntimeStatus) {
						runtimeStatus.DecryptState = "error"
						runtimeStatus.LastError = err.Error()
					})
					rt.publishEvent("runtime.decrypt.failed", map[string]any{"task_id": taskID, "error": err.Error(), "message": err.Error()})
					return
				}
			}
			now := time.Now().UTC().Format(time.RFC3339)
			rt.store.AppendLog("info", "decrypt", "decrypt task completed", map[string]string{"task_id": taskID})
			rt.store.UpdateStatus(func(runtimeStatus *runtimepkg.RuntimeStatus) {
				runtimeStatus.DecryptState = "ready"
				runtimeStatus.LastError = ""
				runtimeStatus.LastDecryptAt = &now
				rt.applyDataFreshness(runtimeStatus)
			})
			rt.publishEvent("runtime.decrypt.finished", map[string]any{"task_id": taskID, "message": "decrypt finished"})
		case ingest.TaskStateStopped:
			rt.store.AppendLog("info", "decrypt", "decrypt task stopped", map[string]string{"task_id": taskID})
			rt.store.UpdateStatus(func(runtimeStatus *runtimepkg.RuntimeStatus) {
				runtimeStatus.DecryptState = "idle"
			})
			rt.publishEvent("runtime.decrypt.stopped", map[string]any{"task_id": taskID, "message": "decrypt stopped"})
		default:
			errMsg := status.Error
			if errMsg == "" {
				errMsg = string(status.State)
			}
			rt.store.AppendLog("error", "decrypt", errMsg, map[string]string{"task_id": taskID})
			rt.store.UpdateStatus(func(runtimeStatus *runtimepkg.RuntimeStatus) {
				runtimeStatus.DecryptState = "error"
				runtimeStatus.LastError = errMsg
			})
			rt.publishEvent("runtime.decrypt.failed", map[string]any{"task_id": taskID, "error": errMsg, "message": errMsg})
		}
	}()

	return taskID, nil
}

func (rt *systemRuntime) startBuiltinStage(stageOpts ingest.StageOptions) (string, error) {
	task := rt.store.StartTask("", runtimepkg.TaskKindDecrypt, "Builtin data stage", stageOpts.SourceDir+" -> "+stageOpts.TargetDir)
	taskID := task.ID

	rt.store.UpdateStatus(func(status *runtimepkg.RuntimeStatus) {
		status.DecryptState = "running"
		status.LastError = ""
	})
	rt.store.AppendLog("info", "decrypt", "started builtin stage task "+taskID, map[string]string{
		"source_dir": stageOpts.SourceDir,
		"target_dir": stageOpts.TargetDir,
	})
	rt.publishEvent("runtime.decrypt.started", map[string]any{"task_id": taskID, "message": "builtin stage started"})

	go func() {
		result, err := ingest.StageWeChatData(stageOpts)
		if err != nil {
			rt.store.FinishTask(taskID, runtimepkg.TaskStatusFailed, "", err.Error())
			rt.store.AppendLog("error", "decrypt", err.Error(), map[string]string{"task_id": taskID})
			rt.store.UpdateStatus(func(runtimeStatus *runtimepkg.RuntimeStatus) {
				runtimeStatus.DecryptState = "error"
				runtimeStatus.LastError = err.Error()
			})
			rt.publishEvent("runtime.decrypt.failed", map[string]any{"task_id": taskID, "error": err.Error(), "message": err.Error()})
			return
		}
		if rt.onSnapshotReady != nil {
			if err := rt.onSnapshotReady("builtin stage completed"); err != nil {
				rt.store.FinishTask(taskID, runtimepkg.TaskStatusFailed, "", err.Error())
				rt.store.AppendLog("error", "analysis", err.Error(), map[string]string{"task_id": taskID})
				rt.store.UpdateStatus(func(runtimeStatus *runtimepkg.RuntimeStatus) {
					runtimeStatus.DecryptState = "error"
					runtimeStatus.LastError = err.Error()
				})
				rt.publishEvent("runtime.decrypt.failed", map[string]any{"task_id": taskID, "error": err.Error(), "message": err.Error()})
				return
			}
		}

		now := time.Now().UTC().Format(time.RFC3339)
		rt.store.FinishTask(taskID, runtimepkg.TaskStatusSucceeded, fmt.Sprintf("staged %d files", len(result.CopiedFiles)), "")
		rt.store.AppendLog("info", "decrypt", fmt.Sprintf("builtin stage completed with %d files", len(result.CopiedFiles)), map[string]string{"task_id": taskID})
		rt.store.UpdateStatus(func(runtimeStatus *runtimepkg.RuntimeStatus) {
			runtimeStatus.DecryptState = "ready"
			runtimeStatus.LastError = ""
			runtimeStatus.LastDecryptAt = &now
			rt.applyDataFreshness(runtimeStatus)
		})
		rt.publishEvent("runtime.decrypt.finished", map[string]any{
			"task_id":      taskID,
			"message":      "builtin stage finished",
			"copied_files": result.CopiedFiles,
		})
	}()

	return taskID, nil
}

func (rt *systemRuntime) registerRoutes(api *gin.RouterGroup) {
	api.GET("/events", rt.handleSSE)
	api.GET("/system/runtime", func(c *gin.Context) {
		c.JSON(http.StatusOK, rt.store.Status())
	})
	api.GET("/system/config-check", func(c *gin.Context) {
		c.JSON(http.StatusOK, rt.inspectConfigCheck(decryptStartRequest{}))
	})
	api.GET("/system/tasks", func(c *gin.Context) {
		limit := parsePositiveInt(c.DefaultQuery("limit", "100"), 100)
		c.JSON(http.StatusOK, gin.H{"items": rt.listTasks(limit)})
	})
	api.GET("/system/logs", func(c *gin.Context) {
		limit := parsePositiveInt(c.DefaultQuery("limit", "200"), 200)
		taskID := strings.TrimSpace(c.Query("task_id"))
		if taskID == "" {
			c.JSON(http.StatusOK, gin.H{"items": rt.listRuntimeLogs(limit)})
			return
		}
		items, ok := rt.listTaskLogs(taskID, limit)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	})
	api.GET("/system/changes", func(c *gin.Context) {
		status := rt.store.Status()
		sinceRevision := parseInt64(c.Query("since_revision"), 0)
		hasNewerRevision := status.DataRevision > sinceRevision
		items := []systemLogItem{}
		if sinceRevision <= 0 || hasNewerRevision {
			items = rt.listRuntimeLogs(20)
		}
		payload := gin.H{
			"data_revision":      status.DataRevision,
			"since_revision":     sinceRevision,
			"has_newer_revision": hasNewerRevision,
			"pending_changes":    status.PendingChanges,
			"last_reindex_at":    status.LastReindexAt,
			"last_change_reason": status.LastChangeReason,
			"last_error":         status.LastError,
			"items":              items,
		}
		if rt.syncManager != nil {
			payload["sync"] = rt.syncManager.Status()
		}
		c.JSON(http.StatusOK, payload)
	})
	api.POST("/system/decrypt/start", rt.handleStartDecrypt)
	api.POST("/system/decrypt/stop", rt.handleStopDecrypt)
	api.POST("/system/reindex", rt.handleReindex)
}

func (rt *systemRuntime) handleStartDecrypt(c *gin.Context) {
	var body decryptStartRequest
	_ = c.ShouldBindJSON(&body)
	result, err := rt.validateDecryptStart(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             err.Error(),
			"config_check":      result.ConfigCheck,
			"suggested_actions": result.ConfigCheck.SuggestedActions,
		})
		return
	}
	if body.AutoRefresh != nil && *body.AutoRefresh {
		if err := rt.ensureSyncStarted(); err != nil {
			rt.store.AppendLog("warn", "sync", err.Error(), nil)
		}
	}
	taskID, err := rt.startDecrypt(result.Options)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"task_id": taskID, "status": "started"})
}

func (rt *systemRuntime) handleStopDecrypt(c *gin.Context) {
	var body struct {
		TaskID string `json:"task_id"`
	}
	_ = c.ShouldBindJSON(&body)

	taskID := strings.TrimSpace(body.TaskID)
	if taskID == "" {
		statuses := rt.orchestrator.ListStatuses()
		for _, status := range statuses {
			if status.State == ingest.TaskStateRunning || status.State == ingest.TaskStateStopping {
				taskID = status.ID
				break
			}
		}
	}

	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no running decrypt task"})
		return
	}
	status, ok := rt.orchestrator.Status(taskID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task not found"})
		return
	}
	if status.State != ingest.TaskStateRunning && status.State != ingest.TaskStateStopping {
		c.JSON(http.StatusBadRequest, gin.H{"error": "decrypt task is not running"})
		return
	}
	rt.store.UpdateStatus(func(status *runtimepkg.RuntimeStatus) {
		status.DecryptState = "stopping"
	})
	rt.publishEvent("runtime.decrypt.stopping", map[string]any{"task_id": taskID, "message": "decrypt stopping"})
	if err := rt.orchestrator.StopTask(taskID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"task_id": taskID, "status": "stopping"})
}

func (rt *systemRuntime) handleReindex(c *gin.Context) {
	var body struct {
		From int64 `json:"from"`
		To   int64 `json:"to"`
	}
	_ = c.ShouldBindJSON(&body)
	if rt.reindex == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "reindex not configured"})
		return
	}
	rt.reindex(body.From, body.To)
	c.JSON(http.StatusOK, gin.H{"status": "indexing"})
}

func (rt *systemRuntime) handleSSE(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	ch := rt.store.Hub().Subscribe(c.Request.Context())
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			payload, err := json.Marshal(rt.ssePayload(event))
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
			c.Writer.Flush()
		case <-heartbeat.C:
			_, _ = fmt.Fprint(c.Writer, ": heartbeat\n\n")
			c.Writer.Flush()
		}
	}
}

func (rt *systemRuntime) ssePayload(event runtimepkg.RuntimeEvent) map[string]any {
	payload := map[string]any{
		"id":      event.ID,
		"type":    event.Type,
		"at":      event.Timestamp.UTC().Format(time.RFC3339),
		"payload": event.Payload,
	}
	switch value := event.Payload.(type) {
	case string:
		payload["message"] = value
	case map[string]any:
		if message, ok := value["message"].(string); ok && message != "" {
			payload["message"] = message
		}
		if revision, ok := value["revision"]; ok {
			payload["revision"] = revision
		}
	}
	return payload
}

func (rt *systemRuntime) publishEvent(typ string, payload any) {
	rt.store.Hub().Publish(typ, payload)
}

func (rt *systemRuntime) listTasks(limit int) []systemTaskItem {
	items := make([]systemTaskItem, 0)
	for _, task := range rt.store.Tasks(limit) {
		message := task.Message
		if message == "" {
			message = task.Detail
		}
		if message == "" {
			message = task.Title
		}
		items = append(items, systemTaskItem{
			ID:         task.ID,
			Type:       task.Kind,
			Status:     task.Status,
			Message:    message,
			Detail:     task.Detail,
			Error:      task.Error,
			Progress:   task.Progress,
			StartedAt:  firstNonEmpty(task.StartedAt, task.CreatedAt),
			FinishedAt: task.FinishedAt,
			UpdatedAt:  task.UpdatedAt,
		})
	}
	for _, task := range rt.orchestrator.ListStatuses() {
		items = append(items, systemTaskItem{
			ID:             task.ID,
			Type:           "decrypt",
			Status:         string(task.State),
			Message:        firstNonEmpty(task.Error, task.Command),
			Detail:         strings.Join(task.Args, " "),
			Error:          task.Error,
			StartedAt:      formatTaskTime(task.StartedAt),
			FinishedAt:     formatTaskTime(task.FinishedAt),
			UpdatedAt:      formatTaskTime(task.LastUpdateAt),
			WorkDir:        task.WorkDir,
			CommandSummary: summarizeCommand(task.Command, task.Args),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].StartedAt > items[j].StartedAt
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func (rt *systemRuntime) listRuntimeLogs(limit int) []systemLogItem {
	logs := rt.store.Logs(limit)
	items := make([]systemLogItem, 0, len(logs))
	for _, entry := range logs {
		items = append(items, systemLogItem{
			ID:      entry.ID,
			Time:    entry.Timestamp,
			Level:   entry.Level,
			Source:  entry.Source,
			Message: entry.Message,
			Fields:  cloneStringMap(entry.Fields),
		})
	}
	return items
}

func (rt *systemRuntime) listTaskLogs(taskID string, limit int) ([]systemLogItem, bool) {
	logs, ok := rt.orchestrator.Logs(taskID, 0, limit)
	if !ok {
		return nil, false
	}
	items := make([]systemLogItem, 0, len(logs))
	for _, entry := range logs {
		items = append(items, systemLogItem{
			ID:      uint64(entry.Seq),
			Time:    entry.Time.UTC().Format(time.RFC3339),
			Level:   entry.Stream,
			Source:  entry.Stream,
			Stream:  entry.Stream,
			TaskID:  taskID,
			Message: entry.Message,
			Fields: map[string]string{
				"task_id": taskID,
				"stream":  entry.Stream,
			},
		})
	}
	return items, true
}

type decryptStartRequest struct {
	Command         string `json:"command"`
	WorkDir         string `json:"work_dir"`
	SourceDataDir   string `json:"source_data_dir"`
	AnalysisDataDir string `json:"analysis_data_dir"`
	Platform        string `json:"platform"`
	AutoRefresh     *bool  `json:"auto_refresh"`
	WALEnabled      *bool  `json:"wal_enabled"`
}

type resolvedDecryptRequest struct {
	Platform        string
	WorkDir         string
	SourceDataDir   string
	AnalysisDataDir string
	CommandText     string
	AutoRefresh     bool
	WALEnabled      bool
}

func (rt *systemRuntime) resolveDecryptRequest(req decryptStartRequest) resolvedDecryptRequest {
	platform := ingest.NormalizePlatform(firstNonEmpty(req.Platform, rt.cfg.Ingest.Platform, goruntime.GOOS))
	workDir := strings.TrimSpace(req.WorkDir)
	if workDir == "" {
		workDir = rt.cfg.Ingest.WorkDir
	}
	sourceDataDir := strings.TrimSpace(req.SourceDataDir)
	if sourceDataDir == "" {
		sourceDataDir = rt.cfg.Ingest.SourceDataDir
	}
	analysisDataDir := strings.TrimSpace(req.AnalysisDataDir)
	if analysisDataDir == "" {
		analysisDataDir = rt.cfg.Ingest.AnalysisDataDir
	}
	commandText := strings.TrimSpace(req.Command)
	if commandText == "" {
		commandText = defaultDecryptTemplate(rt.cfg, platform)
	}
	autoRefresh := rt.cfg.Sync.Enabled
	if req.AutoRefresh != nil {
		autoRefresh = *req.AutoRefresh
	}
	walEnabled := rt.cfg.Sync.WatchWAL
	if req.WALEnabled != nil {
		walEnabled = *req.WALEnabled
	}
	return resolvedDecryptRequest{
		Platform:        platform,
		WorkDir:         workDir,
		SourceDataDir:   sourceDataDir,
		AnalysisDataDir: analysisDataDir,
		CommandText:     commandText,
		AutoRefresh:     autoRefresh,
		WALEnabled:      walEnabled,
	}
}

func (rt *systemRuntime) inspectConfigCheck(req decryptStartRequest) runtimeConfigCheck {
	resolved := rt.resolveDecryptRequest(req)
	deploymentTarget := detectDeploymentTarget()
	mode := "analysis-only"
	if strings.TrimSpace(resolved.CommandText) != "" {
		mode = "external-command"
	} else if strings.TrimSpace(resolved.SourceDataDir) != "" || strings.TrimSpace(resolved.AnalysisDataDir) != "" || rt.cfg.Ingest.Enabled {
		mode = "manual-stage"
	}

	analysisDir := inspectDataDir(resolved.AnalysisDataDir, "")
	sourceDir := inspectDataDir(resolved.SourceDataDir, resolved.AnalysisDataDir)
	workDir := inspectWorkDir(resolved.WorkDir)
	sameDir := sourceDir.Path != "" && analysisDir.Path != "" && filepath.Clean(sourceDir.Path) == filepath.Clean(analysisDir.Path)
	if sameDir {
		sourceDir.SameAsAnalysis = true
		sourceDir.Ready = false
		sourceDir.Issues = append(sourceDir.Issues, "source_data_dir 和 analysis_data_dir 不能指向同一目录")
	}

	decrypt := capabilityValidation{
		Supported: true,
		Enabled:   rt.cfg.Decrypt.Enabled,
		Provider:  firstNonEmpty(strings.TrimSpace(rt.cfg.Decrypt.Provider), "builtin"),
	}
	syncStatus := capabilityValidation{
		Supported: rt.syncManager != nil,
		Enabled:   rt.cfg.Sync.Enabled,
	}
	if syncStatus.Supported && deploymentTarget == "docker" && mode == "manual-stage" {
		syncStatus.Supported = false
		syncStatus.Issues = append(syncStatus.Issues, "Docker 手动同步模式不支持容器内 watcher/auto refresh")
	}
	if !syncStatus.Supported && rt.syncManager == nil {
		syncStatus.Issues = append(syncStatus.Issues, "sync manager 未配置")
	}

	switch mode {
	case "analysis-only":
		decrypt.Supported = false
		decrypt.Warnings = append(decrypt.Warnings, "当前为 analysis-only，仅分析 analysis 目录，不执行同步")
	case "manual-stage":
		if sourceDir.Path == "" {
			decrypt.Supported = false
			decrypt.Issues = append(decrypt.Issues, "未配置 source_data_dir 标准目录")
		}
		if !sourceDir.Exists && sourceDir.Path != "" {
			decrypt.Supported = false
			decrypt.Issues = append(decrypt.Issues, "source_data_dir 不存在")
		}
		if sourceDir.Exists && !sourceDir.StandardLayout {
			decrypt.Supported = false
			decrypt.Issues = append(decrypt.Issues, "source_data_dir 不是标准目录（需要 contact/message，可选 sns）")
		}
		if sourceDir.SameAsAnalysis {
			decrypt.Supported = false
			decrypt.Issues = append(decrypt.Issues, "source_data_dir 和 analysis_data_dir 不能相同")
		}
		if analysisDir.Path == "" {
			decrypt.Supported = false
			decrypt.Issues = append(decrypt.Issues, "未配置 analysis_data_dir")
		}
		if !workDir.Writable {
			decrypt.Supported = false
			decrypt.Issues = append(decrypt.Issues, "work_dir 不可写")
		}
		if deploymentTarget == "docker" {
			decrypt.Warnings = append(decrypt.Warnings, "Docker 首期正式模式为手动同步标准目录，不推荐容器内自动刷新")
		}
	case "external-command":
		if strings.TrimSpace(resolved.CommandText) == "" {
			decrypt.Supported = false
			decrypt.Issues = append(decrypt.Issues, "未配置外部解密命令")
		}
		if resolved.WorkDir != "" && !workDir.Writable {
			decrypt.Supported = false
			decrypt.Issues = append(decrypt.Issues, "work_dir 不可写")
		}
	}

	sns := snsValidation{}
	if sourceDir.HasSNS {
		sns.Detected = true
		sns.Ready = sourceDir.StandardLayout
		sns.DBPath = sourceDir.SNSDBPath
	} else if analysisDir.HasSNS {
		sns.Detected = true
		sns.Ready = analysisDir.Ready
		sns.DBPath = analysisDir.SNSDBPath
	}
	if !sns.Detected && sourceDir.Path != "" {
		sns.Issues = append(sns.Issues, "未检测到 sns/sns.db（可选）")
	}
	check := runtimeConfigCheck{
		DeploymentTarget: deploymentTarget,
		Mode:             mode,
		AnalysisDir:      analysisDir,
		SourceDir:        sourceDir,
		WorkDir:          workDir,
		Decrypt:          decrypt,
		Sync:             syncStatus,
		SNS:              sns,
		SuggestedActions: buildSuggestedActions(deploymentTarget, mode, sourceDir, analysisDir, workDir, decrypt, syncStatus),
	}
	check.Issues = dedupeStrings(append(append(append(append([]string{}, analysisDir.Issues...), sourceDir.Issues...), workDir.Issues...), append(decrypt.Issues, append(syncStatus.Issues, sns.Issues...)...)...))
	check.Warnings = dedupeStrings(append(append(append([]string{}, decrypt.Warnings...), syncStatus.Warnings...), sns.Warnings...))
	return check
}

func (rt *systemRuntime) validateDecryptStart(req decryptStartRequest) (decryptValidationResult, error) {
	check := rt.inspectConfigCheck(req)
	if !check.Decrypt.Supported {
		return decryptValidationResult{ConfigCheck: check}, fmt.Errorf(firstNonEmpty(check.Decrypt.Issues...))
	}
	resolved := rt.resolveDecryptRequest(req)
	if req.AutoRefresh != nil && *req.AutoRefresh && !check.Sync.Supported &&
		check.DeploymentTarget == "docker" && check.Mode == "manual-stage" {
		return decryptValidationResult{ConfigCheck: check}, fmt.Errorf(firstNonEmpty(check.Sync.Issues...))
	}
	opts, err := rt.buildDecryptStartOptions(req)
	if err != nil {
		return decryptValidationResult{ConfigCheck: check}, err
	}
	if resolved.AutoRefresh && rt.syncManager == nil &&
		check.DeploymentTarget == "docker" && check.Mode == "manual-stage" {
		return decryptValidationResult{ConfigCheck: check}, fmt.Errorf("sync manager not configured")
	}
	return decryptValidationResult{Options: opts, ConfigCheck: check}, nil
}

func (rt *systemRuntime) buildDecryptStartOptions(req decryptStartRequest) (ingest.StartOptions, error) {
	resolved := rt.resolveDecryptRequest(req)

	if resolved.WorkDir != "" {
		if err := os.MkdirAll(resolved.WorkDir, 0o755); err != nil {
			return ingest.StartOptions{}, fmt.Errorf("prepare work dir failed: %w", err)
		}
	}

	if resolved.CommandText == "" {
		if resolved.SourceDataDir == "" || resolved.AnalysisDataDir == "" {
			return ingest.StartOptions{}, fmt.Errorf("builtin staging requires both source_data_dir and analysis_data_dir")
		}
		if filepath.Clean(resolved.SourceDataDir) == filepath.Clean(resolved.AnalysisDataDir) {
			return ingest.StartOptions{}, fmt.Errorf("builtin staging requires source_data_dir and analysis_data_dir to be different")
		}
		rt.store.UpdateStatus(func(status *runtimepkg.RuntimeStatus) {
			status.EngineType = resolved.Platform
		})
		return ingest.StartOptions{
			Name: "decrypt",
			BuiltinStage: &ingest.StageOptions{
				SourceDir:   resolved.SourceDataDir,
				TargetDir:   resolved.AnalysisDataDir,
				PreserveWAL: resolved.WALEnabled || rt.cfg.Decrypt.PreserveWAL,
			},
			WorkDir: resolved.WorkDir,
		}, nil
	}
	opts, err := ingest.ResolveCommandSpec(ingest.CommandSpec{
		Platform:        resolved.Platform,
		CommandTemplate: resolved.CommandText,
		SourceDataDir:   resolved.SourceDataDir,
		AnalysisDataDir: resolved.AnalysisDataDir,
		WorkDir:         resolved.WorkDir,
		AutoRefresh:     resolved.AutoRefresh,
		WALEnabled:      resolved.WALEnabled,
	})
	if err != nil {
		return ingest.StartOptions{}, err
	}

	rt.store.UpdateStatus(func(status *runtimepkg.RuntimeStatus) {
		status.EngineType = resolved.Platform
	})
	return opts, nil
}

func defaultDecryptTemplate(cfg *config.Config, platform string) string {
	switch platform {
	case "windows":
		if strings.TrimSpace(cfg.Decrypt.WindowsCommand) != "" {
			return strings.TrimSpace(cfg.Decrypt.WindowsCommand)
		}
	case "macos":
		if strings.TrimSpace(cfg.Decrypt.MacCommand) != "" {
			return strings.TrimSpace(cfg.Decrypt.MacCommand)
		}
	default:
		if strings.TrimSpace(cfg.Decrypt.LinuxCommand) != "" {
			return strings.TrimSpace(cfg.Decrypt.LinuxCommand)
		}
	}
	if strings.TrimSpace(cfg.Decrypt.Provider) == "builtin" {
		return ""
	}
	return strings.TrimSpace(cfg.Decrypt.Provider)
}

func detectDeploymentTarget() string {
	if value := strings.TrimSpace(os.Getenv("WELINK_DEPLOYMENT_TARGET")); value != "" {
		return value
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "docker"
	}
	return "host"
}

func inspectDataDir(path string, analysisPath string) directoryValidation {
	result := directoryValidation{
		Path: strings.TrimSpace(path),
	}
	if result.Path == "" {
		result.Issues = append(result.Issues, "未配置目录")
		return result
	}
	info, err := os.Stat(result.Path)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("目录不存在: %s", result.Path))
		return result
	}
	if !info.IsDir() {
		result.Issues = append(result.Issues, fmt.Sprintf("路径不是目录: %s", result.Path))
		return result
	}
	result.Exists = true
	result.Readable = true
	if analysisPath != "" && filepath.Clean(result.Path) == filepath.Clean(analysisPath) {
		result.SameAsAnalysis = true
	}

	contactPath := filepath.Join(result.Path, "contact", "contact.db")
	if info, err := os.Stat(contactPath); err == nil && !info.IsDir() {
		result.ContactDBPath = contactPath
		result.HasContact = true
	}
	messageDir := filepath.Join(result.Path, "message")
	result.MessageDirPath = messageDir
	if entries, err := os.ReadDir(messageDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := strings.ToLower(entry.Name())
			if strings.HasPrefix(name, "message_") && strings.HasSuffix(name, ".db") {
				result.MessageDBCount++
				result.HasMessage = true
			}
		}
	}
	snsPath := filepath.Join(result.Path, "sns", "sns.db")
	if info, err := os.Stat(snsPath); err == nil && !info.IsDir() {
		result.SNSDBPath = snsPath
		result.HasSNS = true
	}

	if result.ContactDBPath == "" {
		result.Issues = append(result.Issues, "缺少 contact/contact.db")
	}
	if result.MessageDBCount == 0 {
		result.Issues = append(result.Issues, "缺少 message/message_*.db")
	}
	if !result.HasSNS {
		result.Warnings = append(result.Warnings, "未检测到 sns/sns.db（可选）")
	}
	result.StandardLayout = result.ContactDBPath != "" && result.MessageDBCount > 0
	result.Ready = result.StandardLayout && !result.SameAsAnalysis
	return result
}

func inspectWorkDir(path string) directoryValidation {
	result := directoryValidation{
		Path: strings.TrimSpace(path),
	}
	if result.Path == "" {
		result.Issues = append(result.Issues, "未配置 work_dir")
		return result
	}
	if err := os.MkdirAll(result.Path, 0o755); err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("创建目录失败: %v", err))
		return result
	}
	info, err := os.Stat(result.Path)
	if err != nil || !info.IsDir() {
		result.Issues = append(result.Issues, fmt.Sprintf("work_dir 不存在或不是目录: %s", result.Path))
		return result
	}
	result.Exists = true
	result.Readable = true

	probe, err := os.CreateTemp(result.Path, ".welink-write-check-*")
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("目录不可写: %v", err))
		return result
	}
	result.Writable = true
	_ = probe.Close()
	_ = os.Remove(probe.Name())
	result.Ready = true
	return result
}

func buildSuggestedActions(
	deploymentTarget, mode string,
	source directoryValidation,
	analysis directoryValidation,
	work directoryValidation,
	decrypt capabilityValidation,
	sync capabilityValidation,
) []string {
	actions := make([]string, 0, 6)
	if deploymentTarget == "docker" {
		actions = append(actions, "Docker 推荐流程：容器外准备标准目录后，在系统页点击“校验并同步标准目录”")
	}
	if source.Path == "" {
		actions = append(actions, "配置 source_data_dir 指向标准目录（contact/contact.db + message/message_*.db，可选 sns/sns.db）")
	} else if !source.StandardLayout {
		actions = append(actions, "把 source_data_dir 改为标准目录根，而不是 xwechat_files 原始根目录")
	}
	if source.SameAsAnalysis {
		actions = append(actions, "将 source_data_dir 与 analysis_data_dir 分开挂载，避免读写同目录")
	}
	if analysis.Path == "" {
		actions = append(actions, "配置 analysis_data_dir 作为 WeLink 分析目录")
	}
	if !work.Writable {
		actions = append(actions, "修复 work_dir 映射或权限，确保容器内可写")
	}
	if mode == "manual-stage" && deploymentTarget == "docker" {
		actions = append(actions, "Docker 首期不建议启用 auto_refresh/watcher；更新标准目录后手动同步即可")
	}
	if !decrypt.Supported && len(decrypt.Issues) > 0 {
		actions = append(actions, decrypt.Issues[0])
	}
	if !sync.Supported && len(sync.Issues) > 0 {
		actions = append(actions, sync.Issues[0])
	}
	return dedupeStrings(actions)
}

func (rt *systemRuntime) analysisDataDir() string {
	if value := strings.TrimSpace(rt.cfg.Ingest.AnalysisDataDir); value != "" {
		return value
	}
	return strings.TrimSpace(rt.cfg.Data.Dir)
}

func (rt *systemRuntime) applyDataFreshness(status *runtimepkg.RuntimeStatus) {
	messageAt, snsAt := inspectDataFreshness(rt.analysisDataDir())
	status.LastMessageAt = messageAt
	status.LastSNSAt = snsAt
}

func inspectDataFreshness(root string) (*string, *string) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, nil
	}
	var latestMessage time.Time
	if entries, err := os.ReadDir(filepath.Join(root, "message")); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := strings.ToLower(entry.Name())
			if !strings.HasPrefix(name, "message_") || !strings.HasSuffix(name, ".db") {
				continue
			}
			info, err := entry.Info()
			if err == nil && info.ModTime().After(latestMessage) {
				latestMessage = info.ModTime()
			}
		}
	}
	var latestSNS time.Time
	if info, err := os.Stat(filepath.Join(root, "sns", "sns.db")); err == nil && !info.IsDir() {
		latestSNS = info.ModTime()
	}
	return timePtrString(latestMessage), timePtrString(latestSNS)
}

func timePtrString(value time.Time) *string {
	if value.IsZero() {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

func (rt *systemRuntime) shouldStageSourceBeforeReindex() bool {
	return rt.cfg.Decrypt.Enabled &&
		strings.TrimSpace(rt.cfg.Decrypt.Provider) == "builtin" &&
		strings.TrimSpace(rt.cfg.Ingest.SourceDataDir) != "" &&
		strings.TrimSpace(rt.cfg.Ingest.AnalysisDataDir) != "" &&
		filepath.Clean(rt.cfg.Ingest.SourceDataDir) != filepath.Clean(rt.cfg.Ingest.AnalysisDataDir)
}

func (rt *systemRuntime) stageSourceSnapshot() error {
	result, err := ingest.StageWeChatData(ingest.StageOptions{
		SourceDir:   rt.cfg.Ingest.SourceDataDir,
		TargetDir:   rt.cfg.Ingest.AnalysisDataDir,
		PreserveWAL: rt.cfg.Sync.WatchWAL || rt.cfg.Decrypt.PreserveWAL,
	})
	if err != nil {
		rt.store.AppendLog("error", "ingest", err.Error(), nil)
		return err
	}
	rt.store.AppendLog("info", "ingest", fmt.Sprintf("staged %d files from source to analysis", len(result.CopiedFiles)), map[string]string{
		"source_dir": rt.cfg.Ingest.SourceDataDir,
		"target_dir": rt.cfg.Ingest.AnalysisDataDir,
	})
	return nil
}

func (rt *systemRuntime) ensureSyncStarted() error {
	if rt.syncManager == nil {
		return fmt.Errorf("sync manager not configured")
	}
	return rt.syncManager.Start()
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	var value int
	if _, err := fmt.Sscanf(raw, "%d", &value); err != nil || value <= 0 {
		return fallback
	}
	return value
}

func formatTaskTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func parseInt64(raw string, fallback int64) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	var value int64
	if _, err := fmt.Sscanf(raw, "%d", &value); err != nil {
		return fallback
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func summarizeCommand(command string, args []string) string {
	parts := []string{strings.TrimSpace(command)}
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			continue
		}
		parts = append(parts, arg)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}
