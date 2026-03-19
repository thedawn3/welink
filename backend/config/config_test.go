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
