package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"welink/backend/config"
	"welink/backend/ingest"
	"welink/backend/pkg/db"
	"welink/backend/pkg/seed"
	runtimepkg "welink/backend/runtime"
	syncmgr "welink/backend/sync"

	"github.com/gin-gonic/gin"
)

func TestBuildDecryptStartOptionsBuiltinStage(t *testing.T) {
	cfg := &config.Config{}
	cfg.Ingest.Platform = "windows"
	cfg.Ingest.SourceDataDir = "/tmp/source"
	cfg.Ingest.AnalysisDataDir = "/tmp/analysis"
	cfg.Ingest.WorkDir = t.TempDir()
	cfg.Decrypt.Provider = "builtin"

	rt := newSystemRuntime(cfg)
	opts, err := rt.buildDecryptStartOptions(decryptStartRequest{})
	if err != nil {
		t.Fatalf("build decrypt options: %v", err)
	}
	if opts.BuiltinStage == nil {
		t.Fatal("expected builtin stage options")
	}
	if opts.BuiltinStage.SourceDir != "/tmp/source" || opts.BuiltinStage.TargetDir != "/tmp/analysis" {
		t.Fatalf("unexpected builtin stage dirs: %+v", opts.BuiltinStage)
	}
}

func TestBuildDecryptStartOptionsBuiltinStageRejectsSameDir(t *testing.T) {
	cfg := &config.Config{}
	cfg.Ingest.Platform = "windows"
	cfg.Ingest.SourceDataDir = "/tmp/shared"
	cfg.Ingest.AnalysisDataDir = "/tmp/shared"
	cfg.Ingest.WorkDir = t.TempDir()
	cfg.Decrypt.Provider = "builtin"

	rt := newSystemRuntime(cfg)
	_, err := rt.buildDecryptStartOptions(decryptStartRequest{})
	if err == nil {
		t.Fatal("expected same-dir builtin stage to fail")
	}
	if !strings.Contains(err.Error(), "to be different") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSystemConfigCheckEndpointReportsLayoutAndSNS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sourceDir := t.TempDir()
	analysisDir := t.TempDir()
	workDir := t.TempDir()
	mustWriteFile(t, filepath.Join(sourceDir, "contact", "contact.db"), "contact")
	mustWriteFile(t, filepath.Join(sourceDir, "message", "message_0.db"), "message")
	mustWriteFile(t, filepath.Join(sourceDir, "sns", "sns.db"), "sns")

	cfg := &config.Config{}
	cfg.Ingest.SourceDataDir = sourceDir
	cfg.Ingest.AnalysisDataDir = analysisDir
	cfg.Ingest.WorkDir = workDir
	cfg.Decrypt.Provider = "builtin"
	cfg.Sync.Enabled = false
	cfg.Decrypt.Enabled = false
	cfg.Ingest.Enabled = false

	rt := newSystemRuntime(cfg)
	server := newTestSystemServer(rt)
	defer server.Close()

	var payload map[string]any
	fetchJSON(t, server.URL+"/api/system/config-check", &payload)

	if mode, _ := payload["mode"].(string); mode != "manual-stage" {
		t.Fatalf("expected mode manual-stage, got %#v", payload["mode"])
	}
	if deployment, _ := payload["deployment_target"].(string); strings.TrimSpace(deployment) == "" {
		t.Fatalf("expected deployment target, got %#v", payload["deployment_target"])
	}
	if canStart, _ := payload["can_start_sync"].(bool); !canStart {
		t.Fatalf("expected can_start_sync=true, got %#v", payload["can_start_sync"])
	}

	source, ok := payload["source_dir"].(map[string]any)
	if !ok {
		t.Fatalf("expected source_dir object, got %#v", payload["source_dir"])
	}
	if standard, _ := source["standard_layout"].(bool); !standard {
		t.Fatalf("expected source_dir standard_layout=true, got %#v", source["standard_layout"])
	}
	if hasSNS, _ := source["has_sns"].(bool); !hasSNS {
		t.Fatalf("expected source_dir has_sns=true, got %#v", source["has_sns"])
	}

	sns, ok := payload["sns"].(map[string]any)
	if !ok {
		t.Fatalf("expected sns object, got %#v", payload["sns"])
	}
	if detected, _ := sns["detected"].(bool); !detected {
		t.Fatalf("expected sns detected=true, got %#v", sns["detected"])
	}
	if primary, _ := payload["primary_issue"].(string); strings.TrimSpace(primary) != "" {
		t.Fatalf("expected empty primary_issue, got %#v", payload["primary_issue"])
	}
	media, ok := payload["media"].(map[string]any)
	if !ok {
		t.Fatalf("expected media object, got %#v", payload["media"])
	}
	if preview, _ := media["preview_state"].(string); preview != "disabled" {
		t.Fatalf("expected preview_state=disabled without msg dir, got %#v", media["preview_state"])
	}
}

func TestInspectMediaConfigReportsPartialWhenV2ImagesNeedAES(t *testing.T) {
	msgDir := t.TempDir()
	v2Path := filepath.Join(msgDir, "attach", "hash", "2026-03", "Img", "sample_t.dat")
	mustWriteBytes(t, v2Path, []byte{0x07, 0x08, 'V', '2', 0x08, 0x07, 0x00})

	media := inspectMediaConfig(msgDir)
	if media.PreviewState != "partial" {
		t.Fatalf("expected partial preview state, got %+v", media)
	}
	if !media.V2ImagesDetected {
		t.Fatalf("expected v2_images_detected=true, got %+v", media)
	}
	if media.ImageAESKeyConfigured {
		t.Fatalf("expected image_aes_key_configured=false, got %+v", media)
	}
}

func TestInspectMediaConfigReportsReadyWhenAESConfigured(t *testing.T) {
	t.Setenv("WELINK_IMAGE_AES_KEY", "1234567890abcdef")
	msgDir := t.TempDir()
	v2Path := filepath.Join(msgDir, "attach", "hash", "2026-03", "Img", "sample.dat")
	mustWriteBytes(t, v2Path, []byte{0x07, 0x08, 'V', '2', 0x08, 0x07, 0x00})

	media := inspectMediaConfig(msgDir)
	if media.PreviewState != "ready" {
		t.Fatalf("expected ready preview state, got %+v", media)
	}
	if !media.ImageAESKeyConfigured {
		t.Fatalf("expected image_aes_key_configured=true, got %+v", media)
	}
	if media.ImageAESKeySource != "env:WELINK_IMAGE_AES_KEY" {
		t.Fatalf("expected image_aes_key_source from env, got %+v", media)
	}
	if media.ImageKeyMode != "single" || media.ImageKeyCount != 1 {
		t.Fatalf("expected single key mode, got %+v", media)
	}
}

func TestInspectMediaConfigReadsAESKeyFromFile(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "image-key.txt")
	if err := os.WriteFile(keyFile, []byte("1234567890abcdef\n"), 0o644); err != nil {
		t.Fatalf("write key file: %v", err)
	}
	t.Setenv("WELINK_IMAGE_AES_KEY_FILE", keyFile)

	msgDir := t.TempDir()
	v2Path := filepath.Join(msgDir, "attach", "hash", "2026-03", "Img", "sample.dat")
	mustWriteBytes(t, v2Path, []byte{0x07, 0x08, 'V', '2', 0x08, 0x07, 0x00})

	media := inspectMediaConfig(msgDir)
	if media.PreviewState != "ready" {
		t.Fatalf("expected ready preview state from file key, got %+v", media)
	}
	if !media.ImageAESKeyConfigured {
		t.Fatalf("expected image_aes_key_configured=true from file, got %+v", media)
	}
	if media.ImageAESKeySource != "file:"+keyFile {
		t.Fatalf("expected image_aes_key_source=file, got %+v", media)
	}
}

func TestInspectMediaConfigReadsImageKeyMapFromWechatDecryptDir(t *testing.T) {
	toolDir := t.TempDir()
	mapPath := filepath.Join(toolDir, "image_keys.json")
	if err := os.WriteFile(mapPath, []byte("{\n  \"00112233445566778899aabbccddeeff\": \"1234567890abcdef1234567890abcdef\"\n}\n"), 0o644); err != nil {
		t.Fatalf("write key map: %v", err)
	}
	t.Setenv("WELINK_WECHAT_DECRYPT_DIR", toolDir)
	t.Setenv("WELINK_WECHAT_DECRYPT_MOUNT_DIR", toolDir)

	msgDir := t.TempDir()
	v2Path := filepath.Join(msgDir, "attach", "hash", "2026-03", "Img", "sample.dat")
	mustWriteBytes(t, v2Path, []byte{0x07, 0x08, 'V', '2', 0x08, 0x07, 0x00})

	media := inspectMediaConfig(msgDir)
	if media.PreviewState != "ready" {
		t.Fatalf("expected ready preview state from key map, got %+v", media)
	}
	if !media.ImageAESKeyConfigured {
		t.Fatalf("expected image_aes_key_configured=true from key map, got %+v", media)
	}
	if media.ImageKeyMode != "map" || media.ImageKeyCount != 1 {
		t.Fatalf("expected key map mode, got %+v", media)
	}
	if media.WechatDecryptDir != toolDir {
		t.Fatalf("expected wechat_decrypt_dir=%q, got %+v", toolDir, media)
	}
	if media.ImageAESKeySource != "file:"+mapPath {
		t.Fatalf("expected image_aes_key_source=file:%s, got %+v", mapPath, media)
	}
}

func TestHandleStartDecryptRejectsInvalidSourceLayout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	cfg.Ingest.SourceDataDir = t.TempDir()
	cfg.Ingest.AnalysisDataDir = t.TempDir()
	cfg.Ingest.WorkDir = t.TempDir()
	cfg.Decrypt.Provider = "builtin"
	cfg.Sync.Enabled = false

	rt := newSystemRuntime(cfg)
	server := newTestSystemServer(rt)
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/system/decrypt/start", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post start decrypt: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(raw))
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	msg, _ := payload["error"].(string)
	if !strings.Contains(msg, "source 目录不是标准目录") {
		t.Fatalf("expected source layout error, got %#v", payload["error"])
	}
	if configCheck, ok := payload["config_check"].(map[string]any); ok {
		if primary, _ := configCheck["primary_issue"].(string); !strings.Contains(primary, "source 目录不是标准目录") {
			t.Fatalf("expected primary_issue, got %#v", configCheck["primary_issue"])
		}
	}
}

func TestRuntimeStatusIncludesLastMessageAndSNSAt(t *testing.T) {
	analysisDir := t.TempDir()
	mustWriteFile(t, filepath.Join(analysisDir, "contact", "contact.db"), "contact")
	mustWriteFile(t, filepath.Join(analysisDir, "message", "message_0.db"), "message")
	mustWriteFile(t, filepath.Join(analysisDir, "sns", "sns.db"), "sns")

	cfg := &config.Config{}
	cfg.Data.Dir = analysisDir
	cfg.Ingest.AnalysisDataDir = analysisDir
	rt := newSystemRuntime(cfg)

	status := rt.store.Status()
	if status.LastMessageAt == nil || strings.TrimSpace(*status.LastMessageAt) == "" {
		t.Fatalf("expected last_message_at to be set, got %+v", status)
	}
	if status.LastSNSAt == nil || strings.TrimSpace(*status.LastSNSAt) == "" {
		t.Fatalf("expected last_sns_at to be set, got %+v", status)
	}
}

func TestSystemConfigCheckReportsDockerManualStageIssues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("WELINK_DEPLOYMENT_TARGET", "docker")

	cfg := &config.Config{}
	cfg.Ingest.Enabled = true
	cfg.Ingest.SourceDataDir = t.TempDir()
	cfg.Ingest.AnalysisDataDir = t.TempDir()
	cfg.Ingest.WorkDir = t.TempDir()

	rt := newSystemRuntime(cfg)
	server := newTestSystemServer(rt)
	defer server.Close()

	var payload runtimeConfigCheck
	fetchJSON(t, server.URL+"/api/system/config-check", &payload)

	if payload.DeploymentTarget != "docker" {
		t.Fatalf("expected deployment_target=docker, got %#v", payload.DeploymentTarget)
	}
	if payload.Mode != "manual-stage" {
		t.Fatalf("expected mode=manual-stage, got %#v", payload.Mode)
	}
	if payload.SourceDir.StandardLayout {
		t.Fatalf("expected source dir to be non-standard, got %+v", payload.SourceDir)
	}
	if payload.CanStartSync {
		t.Fatalf("expected can_start_sync=false, got %+v", payload)
	}
	if !strings.Contains(payload.PrimaryIssue, "source 目录不是标准目录") {
		t.Fatalf("expected primary issue to mention standard layout, got %+v", payload)
	}
}

func TestSystemConfigCheckReportsAnalysisOnlyAsNeutralMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("WELINK_DEPLOYMENT_TARGET", "docker")
	t.Setenv("WELINK_MODE", "analysis-only")

	analysisDir := t.TempDir()
	mustWriteFile(t, filepath.Join(analysisDir, "contact", "contact.db"), "contact")
	mustWriteFile(t, filepath.Join(analysisDir, "message", "message_0.db"), "message")

	cfg := &config.Config{}
	cfg.Ingest.AnalysisDataDir = analysisDir
	cfg.Ingest.SourceDataDir = t.TempDir()
	cfg.Ingest.WorkDir = t.TempDir()

	rt := newSystemRuntime(cfg)
	server := newTestSystemServer(rt)
	defer server.Close()

	var payload runtimeConfigCheck
	fetchJSON(t, server.URL+"/api/system/config-check", &payload)

	if payload.Mode != "analysis-only" {
		t.Fatalf("expected mode=analysis-only, got %#v", payload.Mode)
	}
	if payload.CanStartSync {
		t.Fatalf("expected can_start_sync=false in analysis-only, got %+v", payload)
	}
	if payload.PrimaryIssue != "" {
		t.Fatalf("expected empty primary_issue in analysis-only, got %#v", payload.PrimaryIssue)
	}
	if strings.TrimSpace(payload.SourceDir.Path) != "" {
		t.Fatalf("expected source_dir path to be hidden in analysis-only, got %+v", payload.SourceDir)
	}
}

func TestHandleStartDecryptRejectsInvalidStandardSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("WELINK_DEPLOYMENT_TARGET", "docker")

	cfg := &config.Config{}
	cfg.Ingest.Enabled = true
	cfg.Ingest.SourceDataDir = t.TempDir()
	cfg.Ingest.AnalysisDataDir = t.TempDir()
	cfg.Ingest.WorkDir = t.TempDir()
	cfg.Decrypt.Provider = "builtin"

	rt := newSystemRuntime(cfg)
	server := newTestSystemServer(rt)
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/system/decrypt/start", bytes.NewReader([]byte(`{"auto_refresh":false}`)))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post decrypt start: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 400, got %d: %s", resp.StatusCode, string(raw))
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if errMsg, _ := payload["error"].(string); !strings.Contains(errMsg, "标准目录") {
		t.Fatalf("expected actionable standard dir error, got %#v", payload["error"])
	}
}

func TestApplyInitialAnalysisStatusPopulatesLatestMessageAndSNSAt(t *testing.T) {
	analysisDir := t.TempDir()
	if err := seed.Generate(analysisDir); err != nil {
		t.Fatalf("seed generate: %v", err)
	}
	snsPath := filepath.Join(analysisDir, "sns", "sns.db")
	if err := os.MkdirAll(filepath.Dir(snsPath), 0o755); err != nil {
		t.Fatalf("mkdir sns dir: %v", err)
	}
	if err := os.WriteFile(snsPath, []byte("sns"), 0o644); err != nil {
		t.Fatalf("write sns db: %v", err)
	}

	cfg := &config.Config{}
	cfg.Ingest.AnalysisDataDir = analysisDir

	rt := newSystemRuntime(cfg)
	rt.applyInitialAnalysisStatus(false, true, 1)

	status := rt.store.Status()
	if status.LastMessageAt == nil || strings.TrimSpace(*status.LastMessageAt) == "" {
		t.Fatalf("expected last_message_at to be populated, got %+v", status)
	}
	if status.LastSNSAt == nil || strings.TrimSpace(*status.LastSNSAt) == "" {
		t.Fatalf("expected last_sns_at to be populated, got %+v", status)
	}
}

func TestBuiltinSourceChangePipelineStagesReloadsAndPublishesSSE(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sourceDir := t.TempDir()
	analysisDir := t.TempDir()

	cfg := &config.Config{}
	cfg.Ingest.SourceDataDir = sourceDir
	cfg.Ingest.AnalysisDataDir = analysisDir
	cfg.Decrypt.Provider = "builtin"
	cfg.Sync.WatchWAL = true
	cfg.Sync.DebounceMs = 60
	cfg.Sync.MaxWaitMs = 200
	cfg.Sync.EventBuffer = 32

	rt := newSystemRuntime(cfg)
	dbMgr, err := db.NewDBManager(analysisDir)
	if err != nil {
		t.Fatalf("new db manager: %v", err)
	}

	var reindexCount atomic.Int32
	onStart, onFinish := rt.bindReindexHooks(func() int { return 1 })
	manager := newTestSyncManager(t, sourceDir, cfg, rt, dbMgr, &reindexCount, onStart, onFinish)
	rt.syncManager = manager
	defer func() { _ = manager.Stop() }()

	server := newTestSystemServer(rt)
	defer server.Close()

	cancelEvents, eventsCh := subscribeSSE(t, server.URL+"/api/events")
	defer cancelEvents()

	if err := manager.Start(); err != nil {
		t.Fatalf("start sync manager: %v", err)
	}
	if err := seed.Generate(sourceDir); err != nil {
		t.Fatalf("seed generate: %v", err)
	}

	seen := make(map[string]int)
	waitForCondition(t, 5*time.Second, func() bool {
		drainSSE(eventsCh, seen)
		return dbMgr.Ready() &&
			reindexCount.Load() > 0 &&
			seen["runtime.revision.detected"] > 0 &&
			seen["runtime.reindex.finished"] > 0
	}, "source change pipeline to finish")

	assertFileExists(t, filepath.Join(analysisDir, "contact", "contact.db"))
	assertFileExists(t, filepath.Join(analysisDir, "message", "message_0.db"))

	status := rt.store.Status()
	if !status.IsInitialized {
		t.Fatalf("expected runtime to be initialized, got %+v", status)
	}
	if status.DataRevision == 0 {
		t.Fatalf("expected data revision to increase, got %+v", status)
	}

	var changes map[string]any
	waitForCondition(t, 5*time.Second, func() bool {
		fetchJSON(t, server.URL+"/api/system/changes", &changes)
		return asFloat64(changes["data_revision"]) >= 1 && asFloat64(changes["pending_changes"]) == 0
	}, "runtime changes to settle")
	if reason, _ := changes["last_change_reason"].(string); strings.TrimSpace(reason) == "" {
		t.Fatalf("expected last_change_reason to be populated, got %#v", changes["last_change_reason"])
	}
	items, ok := changes["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected recent change logs, got %#v", changes["items"])
	}
	if !containsLogMessage(items, "detected change revision") {
		t.Fatalf("expected change log entry in recent items, got %#v", items)
	}
	syncStatus, ok := changes["sync"].(map[string]any)
	if !ok {
		t.Fatalf("expected sync status payload, got %#v", changes["sync"])
	}
	if value := asFloat64(syncStatus["last_revision_seq"]); value < 1 {
		t.Fatalf("expected sync last_revision_seq >= 1, got %#v", syncStatus["last_revision_seq"])
	}
}

func TestBuiltinWALChangePipelineCopiesWALAndReindexes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sourceDir := t.TempDir()
	analysisDir := t.TempDir()
	if err := seed.Generate(sourceDir); err != nil {
		t.Fatalf("seed generate: %v", err)
	}

	cfg := &config.Config{}
	cfg.Ingest.SourceDataDir = sourceDir
	cfg.Ingest.AnalysisDataDir = analysisDir
	cfg.Decrypt.Provider = "builtin"
	cfg.Sync.WatchWAL = true
	cfg.Sync.DebounceMs = 60
	cfg.Sync.MaxWaitMs = 200
	cfg.Sync.EventBuffer = 32

	rt := newSystemRuntime(cfg)
	dbMgr, err := db.NewDBManager(analysisDir)
	if err != nil {
		t.Fatalf("new db manager: %v", err)
	}

	var reindexCount atomic.Int32
	onStart, onFinish := rt.bindReindexHooks(func() int { return 1 })
	manager := newTestSyncManager(t, sourceDir, cfg, rt, dbMgr, &reindexCount, onStart, onFinish)
	rt.syncManager = manager
	defer func() { _ = manager.Stop() }()

	server := newTestSystemServer(rt)
	defer server.Close()
	cancelEvents, eventsCh := subscribeSSE(t, server.URL+"/api/events")
	defer cancelEvents()

	if err := manager.Start(); err != nil {
		t.Fatalf("start sync manager: %v", err)
	}

	seen := make(map[string]int)
	waitForCondition(t, 5*time.Second, func() bool {
		drainSSE(eventsCh, seen)
		return dbMgr.Ready() && reindexCount.Load() > 0
	}, "initial seeded sync to finish")
	waitForCondition(t, 2*time.Second, func() bool {
		return manager.Status().PendingBatches == 0
	}, "initial sync batches to drain")
	drainSSE(eventsCh, seen)

	initialReindex := reindexCount.Load()
	initialRevisionEvents := seen["runtime.revision.detected"]

	walPath := filepath.Join(sourceDir, "message", "message_0.db-wal")
	if err := os.WriteFile(walPath, []byte("wal-change"), 0o644); err != nil {
		t.Fatalf("write wal: %v", err)
	}

	waitForCondition(t, 5*time.Second, func() bool {
		drainSSE(eventsCh, seen)
		_, walErr := os.Stat(filepath.Join(analysisDir, "message", "message_0.db-wal"))
		return reindexCount.Load() > initialReindex &&
			seen["runtime.revision.detected"] > initialRevisionEvents &&
			walErr == nil
	}, "wal change pipeline to finish")

	assertFileExists(t, filepath.Join(analysisDir, "message", "message_0.db-wal"))
}

func TestSystemChangesSinceRevisionSuppressesUnchangedLogItems(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	rt := newSystemRuntime(cfg)
	rt.store.AppendLog("info", "sync", "detected change revision rev-1", nil)
	rt.store.UpdateStatus(func(status *runtimepkg.RuntimeStatus) {
		status.DataRevision = 3
		status.PendingChanges = 0
	})

	server := newTestSystemServer(rt)
	defer server.Close()

	var changes map[string]any
	fetchJSON(t, server.URL+"/api/system/changes?since_revision=3", &changes)
	if value := asFloat64(changes["since_revision"]); value != 3 {
		t.Fatalf("expected since_revision=3, got %#v", changes["since_revision"])
	}
	if newer, _ := changes["has_newer_revision"].(bool); newer {
		t.Fatalf("expected has_newer_revision=false, got %#v", changes["has_newer_revision"])
	}
	items, ok := changes["items"].([]any)
	if !ok {
		t.Fatalf("expected items array, got %#v", changes["items"])
	}
	if len(items) != 0 {
		t.Fatalf("expected no log items when revision is unchanged, got %#v", items)
	}

	fetchJSON(t, server.URL+"/api/system/changes?since_revision=2", &changes)
	if newer, _ := changes["has_newer_revision"].(bool); !newer {
		t.Fatalf("expected has_newer_revision=true, got %#v", changes["has_newer_revision"])
	}
	items, ok = changes["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected recent log items when revision advanced, got %#v", changes["items"])
	}
}

func TestSystemTasksAndLogsRoutesReturnItemsWrappers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rt := newSystemRuntime(&config.Config{})
	task := rt.store.StartTask("", runtimepkg.TaskKindDecrypt, "Decrypt demo", "from source")
	rt.store.AppendLog("info", "sync", "detected change revision rev-1", nil)

	server := newTestSystemServer(rt)
	defer server.Close()

	var tasksPayload map[string]any
	fetchJSON(t, server.URL+"/api/system/tasks?limit=10", &tasksPayload)
	items, ok := tasksPayload["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected wrapped task items, got %#v", tasksPayload)
	}
	firstTask, _ := items[0].(map[string]any)
	if gotType, _ := firstTask["type"].(string); gotType != runtimepkg.TaskKindDecrypt {
		t.Fatalf("expected decrypt task type, got %#v", firstTask["type"])
	}
	if gotStatus, _ := firstTask["status"].(string); gotStatus != runtimepkg.TaskStatusRunning {
		t.Fatalf("expected running task status, got %#v", firstTask["status"])
	}
	if _, ok := firstTask["detail"].(string); !ok {
		t.Fatalf("expected detail field, got %#v", firstTask["detail"])
	}
	if _, ok := firstTask["updated_at"].(string); !ok {
		t.Fatalf("expected updated_at field, got %#v", firstTask["updated_at"])
	}

	var logsPayload map[string]any
	fetchJSON(t, server.URL+"/api/system/logs?limit=10", &logsPayload)
	logItems, ok := logsPayload["items"].([]any)
	if !ok || len(logItems) == 0 {
		t.Fatalf("expected wrapped log items, got %#v", logsPayload)
	}
	firstLog, _ := logItems[0].(map[string]any)
	if gotSource, _ := firstLog["source"].(string); gotSource != "sync" {
		t.Fatalf("expected sync log source, got %#v", firstLog["source"])
	}
	if _, ok := firstLog["id"].(float64); !ok {
		t.Fatalf("expected log id, got %#v", firstLog["id"])
	}
	if _, ok := firstLog["time"].(string); !ok {
		t.Fatalf("expected log time, got %#v", firstLog["time"])
	}

	resp, err := http.Get(server.URL + "/api/system/logs?task_id=" + task.ID)
	if err != nil {
		t.Fatalf("get task logs: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404 for unknown orchestrator task logs, got %d: %s", resp.StatusCode, string(raw))
	}
}

func TestHandleReindexRouteInvokesConfiguredRange(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rt := newSystemRuntime(&config.Config{})
	var gotFrom, gotTo int64
	rt.reindex = func(from, to int64) {
		gotFrom = from
		gotTo = to
	}

	server := newTestSystemServer(rt)
	defer server.Close()

	body := bytes.NewBufferString(`{"from":1700000000,"to":1710000000}`)
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/system/reindex", body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post reindex: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(raw))
	}
	if gotFrom != 1700000000 || gotTo != 1710000000 {
		t.Fatalf("unexpected reindex range: from=%d to=%d", gotFrom, gotTo)
	}
}

func TestHandleStartDecryptStartsTaskAndUpdatesRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rt := newSystemRuntime(&config.Config{})
	server := newTestSystemServer(rt)
	defer server.Close()

	workDir := t.TempDir()
	body, err := json.Marshal(map[string]any{
		"command":  "printf 'hello from decrypt\\n'",
		"work_dir": workDir,
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/system/decrypt/start", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post start decrypt: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(raw))
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	taskID, _ := payload["task_id"].(string)
	if strings.TrimSpace(taskID) == "" {
		t.Fatalf("expected task_id, got %#v", payload)
	}
	if got, _ := payload["status"].(string); got != "started" {
		t.Fatalf("expected started status, got %#v", payload["status"])
	}

	waitForCondition(t, 3*time.Second, func() bool {
		status, ok := rt.orchestrator.Status(taskID)
		return ok && status.State == ingest.TaskStateSucceeded
	}, "decrypt start task completion")

	waitForCondition(t, 2*time.Second, func() bool {
		return rt.store.Status().DecryptState == "ready"
	}, "runtime decrypt state ready")

	var runtimePayload map[string]any
	fetchJSON(t, server.URL+"/api/system/runtime", &runtimePayload)
	if gotState, _ := runtimePayload["decrypt_state"].(string); gotState != "ready" {
		t.Fatalf("expected decrypt_state=ready, got %#v", runtimePayload["decrypt_state"])
	}
	if gotEngine, _ := runtimePayload["engine_type"].(string); strings.TrimSpace(gotEngine) == "" {
		t.Fatalf("expected engine_type to be set, got %#v", runtimePayload["engine_type"])
	}
}

func TestHandleStartDecryptWithAutoRefreshStartsSyncManager(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	rt := newSystemRuntime(cfg)
	manager, err := syncmgr.NewManager(syncmgr.ManagerOptions{
		Root:         t.TempDir(),
		PollInterval: 25 * time.Millisecond,
		Debounce:     50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new sync manager: %v", err)
	}
	rt.syncManager = manager
	defer func() { _ = manager.Stop() }()

	server := newTestSystemServer(rt)
	defer server.Close()

	workDir := t.TempDir()
	body, err := json.Marshal(map[string]any{
		"command":      "printf 'auto-refresh start\\n'",
		"work_dir":     workDir,
		"auto_refresh": true,
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/system/decrypt/start", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post start decrypt: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(raw))
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	taskID, _ := payload["task_id"].(string)
	if strings.TrimSpace(taskID) == "" {
		t.Fatalf("expected task_id, got %#v", payload)
	}

	waitForCondition(t, 3*time.Second, func() bool {
		status, ok := rt.orchestrator.Status(taskID)
		return ok && status.State == ingest.TaskStateSucceeded && manager.Status().Running
	}, "decrypt auto refresh task completion and sync manager start")

	var changesPayload map[string]any
	fetchJSON(t, server.URL+"/api/system/changes", &changesPayload)
	syncStatus, ok := changesPayload["sync"].(map[string]any)
	if !ok {
		t.Fatalf("expected sync status payload, got %#v", changesPayload["sync"])
	}
	if running, _ := syncStatus["running"].(bool); !running {
		t.Fatalf("expected sync manager running, got %#v", syncStatus)
	}
}

func TestHandleStartDecryptWithAutoRefreshWithoutSyncManagerLogsWarning(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rt := newSystemRuntime(&config.Config{})
	server := newTestSystemServer(rt)
	defer server.Close()

	workDir := t.TempDir()
	body, err := json.Marshal(map[string]any{
		"command":      "printf 'auto-refresh without manager\\n'",
		"work_dir":     workDir,
		"auto_refresh": true,
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/system/decrypt/start", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post start decrypt: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(raw))
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	taskID, _ := payload["task_id"].(string)
	if strings.TrimSpace(taskID) == "" {
		t.Fatalf("expected task_id, got %#v", payload)
	}

	waitForCondition(t, 3*time.Second, func() bool {
		status, ok := rt.orchestrator.Status(taskID)
		return ok && status.State == ingest.TaskStateSucceeded && rt.store.Status().DecryptState == "ready"
	}, "decrypt task to complete without sync manager")

	var runtimePayload map[string]any
	fetchJSON(t, server.URL+"/api/system/runtime", &runtimePayload)
	if gotState, _ := runtimePayload["decrypt_state"].(string); gotState != "ready" {
		t.Fatalf("expected decrypt_state=ready, got %#v", runtimePayload["decrypt_state"])
	}
	if lastDecryptAt, _ := runtimePayload["last_decrypt_at"].(string); strings.TrimSpace(lastDecryptAt) == "" {
		t.Fatalf("expected last_decrypt_at to be set, got %#v", runtimePayload["last_decrypt_at"])
	}

	var changesPayload map[string]any
	fetchJSON(t, server.URL+"/api/system/changes", &changesPayload)
	if _, exists := changesPayload["sync"]; exists {
		t.Fatalf("expected sync payload to be omitted without sync manager, got %#v", changesPayload["sync"])
	}

	var logsPayload map[string]any
	fetchJSON(t, server.URL+"/api/system/logs?limit=20", &logsPayload)
	items, ok := logsPayload["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected runtime log items, got %#v", logsPayload)
	}
	foundWarn := false
	for _, item := range items {
		record, _ := item.(map[string]any)
		level, _ := record["level"].(string)
		source, _ := record["source"].(string)
		message, _ := record["message"].(string)
		if level == "warn" && source == "sync" && strings.Contains(message, "sync manager not configured") {
			foundWarn = true
			break
		}
	}
	if !foundWarn {
		t.Fatalf("expected sync warning log, got %#v", items)
	}
}

func TestHandleStopDecryptWithoutRunningTaskReturnsIdleOK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rt := newSystemRuntime(&config.Config{})
	server := newTestSystemServer(rt)
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/system/decrypt/stop", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post stop decrypt: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(raw))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if got, _ := payload["message"].(string); !strings.Contains(got, "没有进行中的解密任务") {
		t.Fatalf("unexpected error payload: %#v", payload)
	}
}

func TestHandleStopDecryptStopsRunningTask(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rt := newSystemRuntime(&config.Config{})
	server := newTestSystemServer(rt)
	defer server.Close()

	startBody, err := json.Marshal(map[string]any{
		"command":  "sleep 5",
		"work_dir": t.TempDir(),
	})
	if err != nil {
		t.Fatalf("marshal start body: %v", err)
	}
	startReq, err := http.NewRequest(http.MethodPost, server.URL+"/api/system/decrypt/start", bytes.NewReader(startBody))
	if err != nil {
		t.Fatalf("new start request: %v", err)
	}
	startReq.Header.Set("Content-Type", "application/json")
	startResp, err := http.DefaultClient.Do(startReq)
	if err != nil {
		t.Fatalf("post start decrypt: %v", err)
	}
	defer startResp.Body.Close()
	if startResp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(startResp.Body)
		t.Fatalf("expected 200 from start, got %d: %s", startResp.StatusCode, string(raw))
	}

	var startPayload map[string]any
	if err := json.NewDecoder(startResp.Body).Decode(&startPayload); err != nil {
		t.Fatalf("decode start payload: %v", err)
	}
	taskID, _ := startPayload["task_id"].(string)
	if strings.TrimSpace(taskID) == "" {
		t.Fatalf("expected task_id, got %#v", startPayload)
	}

	waitForCondition(t, 2*time.Second, func() bool {
		status, ok := rt.orchestrator.Status(taskID)
		return ok && status.State == ingest.TaskStateRunning
	}, "decrypt task to start running")

	stopReq, err := http.NewRequest(http.MethodPost, server.URL+"/api/system/decrypt/stop", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatalf("new stop request: %v", err)
	}
	stopReq.Header.Set("Content-Type", "application/json")
	stopResp, err := http.DefaultClient.Do(stopReq)
	if err != nil {
		t.Fatalf("post stop decrypt: %v", err)
	}
	defer stopResp.Body.Close()
	if stopResp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(stopResp.Body)
		t.Fatalf("expected 200 from stop, got %d: %s", stopResp.StatusCode, string(raw))
	}

	var stopPayload map[string]any
	if err := json.NewDecoder(stopResp.Body).Decode(&stopPayload); err != nil {
		t.Fatalf("decode stop payload: %v", err)
	}
	if gotID, _ := stopPayload["task_id"].(string); gotID != taskID {
		t.Fatalf("expected stop task_id=%s, got %#v", taskID, stopPayload["task_id"])
	}
	if gotStatus, _ := stopPayload["status"].(string); gotStatus != "stopping" {
		t.Fatalf("expected stopping status, got %#v", stopPayload["status"])
	}

	waitForCondition(t, 3*time.Second, func() bool {
		status, ok := rt.orchestrator.Status(taskID)
		return ok && status.State == ingest.TaskStateStopped && rt.store.Status().DecryptState == "idle"
	}, "decrypt task to stop cleanly")
}

func TestHandleStopDecryptCompletedTaskReturnsIdleOK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rt := newSystemRuntime(&config.Config{})
	server := newTestSystemServer(rt)
	defer server.Close()

	taskID, err := rt.orchestrator.StartTask(ingest.StartOptions{
		Name:    "decrypt",
		Command: "sh",
		Args:    []string{"-lc", "printf 'done\\n'"},
		WorkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("start orchestrator task: %v", err)
	}

	waitForCondition(t, 2*time.Second, func() bool {
		status, ok := rt.orchestrator.Status(taskID)
		return ok && status.State == ingest.TaskStateSucceeded
	}, "completed decrypt task")

	body, err := json.Marshal(map[string]any{"task_id": taskID})
	if err != nil {
		t.Fatalf("marshal stop body: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/system/decrypt/stop", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new stop request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post stop decrypt: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(raw))
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if got, _ := payload["message"].(string); !strings.Contains(got, "没有进行中的解密任务") {
		t.Fatalf("unexpected error payload: %#v", payload)
	}
}

func TestSystemLogsRouteReturnsTaskLogItemsForDecryptTask(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rt := newSystemRuntime(&config.Config{})
	taskID, err := rt.orchestrator.StartTask(ingest.StartOptions{
		Name:    "decrypt",
		Command: "sh",
		Args:    []string{"-lc", "printf 'stdout-line\\n'"},
		WorkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("start orchestrator task: %v", err)
	}

	waitForCondition(t, 3*time.Second, func() bool {
		status, ok := rt.orchestrator.Status(taskID)
		return ok && status.State == ingest.TaskStateSucceeded
	}, "decrypt task logs to be ready")

	server := newTestSystemServer(rt)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/system/logs?task_id=" + taskID + "&limit=20")
	if err != nil {
		t.Fatalf("get task logs: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(raw))
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	items, ok := payload["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected wrapped task log items, got %#v", payload)
	}
	if !containsLogMessage(items, "stdout-line") {
		t.Fatalf("expected stdout log item, got %#v", items)
	}
}

func TestSystemLogsRouteAppliesTaskLogLimitAndPreservesSequenceOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rt := newSystemRuntime(&config.Config{})
	taskID, err := rt.orchestrator.StartTask(ingest.StartOptions{
		Name:    "decrypt",
		Command: "sh",
		Args:    []string{"-lc", "printf 'out-1\\n'; printf 'err-1\\n' >&2; printf 'out-2\\n'"},
		WorkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("start orchestrator task: %v", err)
	}

	waitForCondition(t, 3*time.Second, func() bool {
		status, ok := rt.orchestrator.Status(taskID)
		return ok && status.State == ingest.TaskStateSucceeded
	}, "decrypt task limited logs to be ready")

	server := newTestSystemServer(rt)
	defer server.Close()

	var payload map[string]any
	fetchJSON(t, server.URL+"/api/system/logs?task_id="+taskID+"&limit=3", &payload)
	items, ok := payload["items"].([]any)
	if !ok || len(items) != 3 {
		t.Fatalf("expected exactly 3 task log items, got %#v", payload)
	}

	first, _ := items[0].(map[string]any)
	second, _ := items[1].(map[string]any)
	third, _ := items[2].(map[string]any)
	firstID := asFloat64(first["id"])
	secondID := asFloat64(second["id"])
	thirdID := asFloat64(third["id"])
	if !(firstID < secondID && secondID < thirdID) {
		t.Fatalf("expected ascending log ids, got [%v,%v,%v]", first["id"], second["id"], third["id"])
	}
	firstSource, _ := first["source"].(string)
	secondSource, _ := second["source"].(string)
	thirdSource, _ := third["source"].(string)
	firstMsg, _ := first["message"].(string)
	secondMsg, _ := second["message"].(string)
	thirdMsg, _ := third["message"].(string)
	if firstSource != "system" || !strings.Contains(firstMsg, "starting command") {
		t.Fatalf("expected first log to be starting command, got %#v", first)
	}
	if secondSource != "system" || !strings.Contains(secondMsg, "process started") {
		t.Fatalf("expected second log to be process started, got %#v", second)
	}
	if thirdSource != "stdout" && thirdSource != "stderr" {
		t.Fatalf("expected third log to be first stream entry, got %#v", third)
	}
	if strings.Contains(thirdMsg, "out-2") || strings.Contains(thirdMsg, "task finished") {
		t.Fatalf("expected limit=3 to exclude later logs, got %#v", third)
	}
}

func newTestSyncManager(
	t *testing.T,
	sourceDir string,
	cfg *config.Config,
	rt *systemRuntime,
	dbMgr *db.DBManager,
	reindexCount *atomic.Int32,
	onStart func(int64, int64),
	onFinish func(int64, int64),
) *syncmgr.Manager {
	t.Helper()

	manager, err := syncmgr.NewManager(syncmgr.ManagerOptions{
		Root:         sourceDir,
		Debounce:     time.Duration(cfg.Sync.DebounceMs) * time.Millisecond,
		PollInterval: 25 * time.Millisecond,
		WatchWAL:     cfg.Sync.WatchWAL,
		OnRevision: func(revision syncmgr.Revision) {
			rt.onDataRevision(revision, func() {
				if err := rt.stageSourceSnapshot(); err != nil {
					t.Errorf("stage source snapshot: %v", err)
					return
				}
				if err := dbMgr.Reload(cfg.Ingest.AnalysisDataDir); err != nil {
					t.Errorf("reload db manager: %v", err)
					return
				}
				reindexCount.Add(1)
				onStart(0, 0)
				onFinish(0, 0)
			})
		},
		OnError: func(err error) {
			t.Errorf("sync manager error: %v", err)
		},
	})
	if err != nil {
		t.Fatalf("new sync manager: %v", err)
	}
	return manager
}

func newTestSystemServer(rt *systemRuntime) *httptest.Server {
	router := gin.New()
	api := router.Group("/api")
	rt.registerRoutes(api)
	return httptest.NewServer(router)
}

func subscribeSSE(t *testing.T, url string) (context.CancelFunc, <-chan map[string]any) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new sse request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("open sse stream: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected sse status: %d", resp.StatusCode)
	}

	out := make(chan map[string]any, 64)
	go func() {
		defer close(out)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 256*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var payload map[string]any
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &payload); err != nil {
				continue
			}
			select {
			case out <- payload:
			case <-ctx.Done():
				return
			}
		}
	}()

	return cancel, out
}

func drainSSE(events <-chan map[string]any, seen map[string]int) {
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			typ, _ := event["type"].(string)
			if typ != "" {
				seen[typ]++
			}
		default:
			return
		}
	}
}

func waitForCondition(t *testing.T, timeout time.Duration, cond func() bool, label string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !cond() {
		t.Fatalf("timeout waiting for %s", label)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s to exist: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func mustWriteBytes(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func fetchJSON(t *testing.T, url string, dest any) {
	t.Helper()

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status %d from %s: %s", resp.StatusCode, url, string(body))
	}
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
}

func asFloat64(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}

func containsLogMessage(items []any, needle string) bool {
	for _, item := range items {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		message, _ := record["message"].(string)
		if strings.Contains(message, needle) {
			return true
		}
	}
	return false
}
