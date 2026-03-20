package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFirstNonEmptyEnvOrder(t *testing.T) {
	t.Setenv("WELINK_DATA_DIR", "")
	t.Setenv("DATA_DIR", "/tmp/data-dir")

	if got := firstNonEmptyEnv("WELINK_DATA_DIR", "DATA_DIR"); got != "/tmp/data-dir" {
		t.Fatalf("expected fallback DATA_DIR, got %q", got)
	}

	t.Setenv("WELINK_DATA_DIR", "/tmp/welink-data-dir")

	if got := firstNonEmptyEnv("WELINK_DATA_DIR", "DATA_DIR"); got != "/tmp/welink-data-dir" {
		t.Fatalf("expected WELINK_DATA_DIR to win, got %q", got)
	}
}

func TestLoadSupportsWelinkEnvNames(t *testing.T) {
	t.Setenv("WELINK_DATA_DIR", "/tmp/custom-data")
	t.Setenv("WELINK_MSG_DIR", "/tmp/custom-msg")
	t.Setenv("WELINK_BACKEND_PORT", "19080")
	t.Setenv("DATA_DIR", "")
	t.Setenv("MSG_DIR", "")
	t.Setenv("PORT", "")

	cfg := Load("/path/that/does/not/exist.yaml")

	if cfg.Data.Dir != "/tmp/custom-data" {
		t.Fatalf("expected data dir from WELINK_DATA_DIR, got %q", cfg.Data.Dir)
	}
	if cfg.Data.MsgDir != "/tmp/custom-msg" {
		t.Fatalf("expected msg dir from WELINK_MSG_DIR, got %q", cfg.Data.MsgDir)
	}
	if cfg.Server.Port != "19080" {
		t.Fatalf("expected port from WELINK_BACKEND_PORT, got %q", cfg.Server.Port)
	}
}

func TestLoadEnvOverridesYaml(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := []byte("server:\n  port: \"18080\"\ndata:\n  dir: /yaml/data\n  msg_dir: /yaml/msg\n")
	if err := os.WriteFile(configPath, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("WELINK_DATA_DIR", "/env/data")
	t.Setenv("WELINK_MSG_DIR", "/env/msg")
	t.Setenv("WELINK_BACKEND_PORT", "19080")

	cfg := Load(configPath)

	if cfg.Data.Dir != "/env/data" {
		t.Fatalf("expected env data dir to override yaml, got %q", cfg.Data.Dir)
	}
	if cfg.Data.MsgDir != "/env/msg" {
		t.Fatalf("expected env msg dir to override yaml, got %q", cfg.Data.MsgDir)
	}
	if cfg.Server.Port != "19080" {
		t.Fatalf("expected env port to override yaml, got %q", cfg.Server.Port)
	}
}

func TestLoadSupportsRuntimeAndIngestEnv(t *testing.T) {
	t.Setenv("WELINK_INGEST_ENABLED", "true")
	t.Setenv("WELINK_SOURCE_DATA_DIR", "/tmp/source")
	t.Setenv("WELINK_WORK_DIR", "/tmp/work")
	t.Setenv("WELINK_ANALYSIS_DATA_DIR", "/tmp/analysis")
	t.Setenv("WELINK_PLATFORM", "windows")
	t.Setenv("WELINK_SYNC_ENABLED", "true")
	t.Setenv("WELINK_SYNC_WATCH_WAL", "false")
	t.Setenv("WELINK_SYNC_DEBOUNCE_MS", "250")
	t.Setenv("WELINK_SYNC_MAX_WAIT_MS", "1200")
	t.Setenv("WELINK_SYNC_EVENT_BUFFER", "64")
	t.Setenv("WELINK_DECRYPT_ENABLED", "true")
	t.Setenv("WELINK_DECRYPT_AUTO_START", "true")
	t.Setenv("WELINK_DECRYPT_PROVIDER", "builtin")
	t.Setenv("WELINK_DECRYPT_PRESERVE_WAL", "true")
	t.Setenv("WELINK_DECRYPT_TIMEOUT_SECONDS", "90")
	t.Setenv("WELINK_RUNTIME_ENGINE_TYPE", "hybrid")
	t.Setenv("WELINK_RUNTIME_MAX_TASK_RECORDS", "50")
	t.Setenv("WELINK_RUNTIME_MAX_LOG_RECORDS", "500")

	cfg := Load("/path/that/does/not/exist.yaml")
	if !cfg.Ingest.Enabled {
		t.Fatalf("expected ingest enabled")
	}
	if cfg.Ingest.SourceDataDir != "/tmp/source" {
		t.Fatalf("expected source data dir, got %q", cfg.Ingest.SourceDataDir)
	}
	if cfg.Ingest.WorkDir != "/tmp/work" {
		t.Fatalf("expected work dir, got %q", cfg.Ingest.WorkDir)
	}
	if cfg.Ingest.AnalysisDataDir != "/tmp/analysis" {
		t.Fatalf("expected analysis data dir, got %q", cfg.Ingest.AnalysisDataDir)
	}
	if cfg.Ingest.Platform != "windows" {
		t.Fatalf("expected platform windows, got %q", cfg.Ingest.Platform)
	}
	if !cfg.Sync.Enabled || cfg.Sync.WatchWAL {
		t.Fatalf("unexpected sync config: %+v", cfg.Sync)
	}
	if cfg.Sync.DebounceMs != 250 || cfg.Sync.MaxWaitMs != 1200 || cfg.Sync.EventBuffer != 64 {
		t.Fatalf("unexpected sync values: %+v", cfg.Sync)
	}
	if !cfg.Decrypt.Enabled || !cfg.Decrypt.AutoStart || !cfg.Decrypt.PreserveWAL {
		t.Fatalf("unexpected decrypt config: %+v", cfg.Decrypt)
	}
	if cfg.Decrypt.TaskTimeoutSeconds != 90 {
		t.Fatalf("expected decrypt timeout 90, got %d", cfg.Decrypt.TaskTimeoutSeconds)
	}
	if cfg.Runtime.EngineType != "hybrid" {
		t.Fatalf("expected runtime engine type hybrid, got %q", cfg.Runtime.EngineType)
	}
	if cfg.Runtime.MaxTaskRecords != 50 || cfg.Runtime.MaxLogRecords != 500 {
		t.Fatalf("unexpected runtime limits: %+v", cfg.Runtime)
	}
}

func TestLoadFallsBackAnalysisDataDirToDataDir(t *testing.T) {
	t.Setenv("WELINK_DATA_DIR", "/tmp/data")
	t.Setenv("WELINK_ANALYSIS_DATA_DIR", "")
	t.Setenv("WELINK_SOURCE_DATA_DIR", "")

	cfg := Load("/path/that/does/not/exist.yaml")
	if cfg.Ingest.SourceDataDir != "/tmp/data" {
		t.Fatalf("expected source dir fallback to data dir, got %q", cfg.Ingest.SourceDataDir)
	}
	if cfg.Ingest.AnalysisDataDir != "/tmp/data" {
		t.Fatalf("expected analysis dir fallback to data dir, got %q", cfg.Ingest.AnalysisDataDir)
	}
}
