package ingest

import (
	"runtime"
	"strings"
	"testing"
)

func TestNormalizePlatform(t *testing.T) {
	if got := NormalizePlatform("darwin"); got != "macos" {
		t.Fatalf("expected macos, got %q", got)
	}
	if got := NormalizePlatform("win"); got != "windows" {
		t.Fatalf("expected windows, got %q", got)
	}
	if got := NormalizePlatform("auto"); got == "" {
		t.Fatal("expected auto platform to resolve")
	}
}

func TestResolveCommandSpecInterpolatesValues(t *testing.T) {
	spec, err := ResolveCommandSpec(CommandSpec{
		Platform:        "linux",
		CommandTemplate: "echo ${source_data_dir} ${analysis_data_dir} ${work_dir} ${auto_refresh} ${wal_enabled}",
		SourceDataDir:   "/src",
		AnalysisDataDir: "/analysis",
		WorkDir:         "/work",
		AutoRefresh:     true,
		WALEnabled:      false,
	})
	if err != nil {
		t.Fatalf("resolve command spec: %v", err)
	}
	if spec.Command != "sh" {
		t.Fatalf("expected sh wrapper, got %q", spec.Command)
	}
	if len(spec.Args) != 2 || spec.Args[0] != "-lc" {
		t.Fatalf("unexpected args: %#v", spec.Args)
	}
	joined := strings.Join(spec.Args, " ")
	for _, want := range []string{"/src", "/analysis", "/work", "true", "false"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected %q in command args: %s", want, joined)
		}
	}
	if spec.Env["WELINK_PLATFORM"] != "linux" {
		t.Fatalf("unexpected env platform: %#v", spec.Env)
	}
}

func TestResolveCommandSpecWindowsWrapper(t *testing.T) {
	spec, err := ResolveCommandSpec(CommandSpec{
		Platform:        "windows",
		CommandTemplate: "echo hello",
	})
	if err != nil {
		t.Fatalf("resolve command spec: %v", err)
	}
	if spec.Command != "cmd" {
		t.Fatalf("expected cmd wrapper, got %q", spec.Command)
	}
	if len(spec.Args) != 2 || spec.Args[0] != "/C" {
		t.Fatalf("unexpected args: %#v", spec.Args)
	}
}

func TestResolveCommandSpecRejectsEmptyCommand(t *testing.T) {
	_, err := ResolveCommandSpec(CommandSpec{
		Platform:        "linux",
		CommandTemplate: "  ",
	})
	if err == nil {
		t.Fatal("expected error for empty command template")
	}
	if !strings.Contains(err.Error(), "no decrypt command configured for linux") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveCommandSpecSetsDefaultNameAndEnv(t *testing.T) {
	spec, err := ResolveCommandSpec(CommandSpec{
		Platform:        "mac",
		CommandTemplate: "echo ${platform}:${auto_refresh}:${wal_enabled}",
		SourceDataDir:   "/input",
		AnalysisDataDir: "/output",
		WorkDir:         "/tmp/work",
		AutoRefresh:     false,
		WALEnabled:      true,
	})
	if err != nil {
		t.Fatalf("resolve command spec: %v", err)
	}
	if spec.Name != "decrypt" {
		t.Fatalf("expected default task name decrypt, got %q", spec.Name)
	}
	if spec.WorkDir != "/tmp/work" {
		t.Fatalf("unexpected workdir: %q", spec.WorkDir)
	}
	if got := strings.Join(spec.Args, " "); !strings.Contains(got, "macos:false:true") {
		t.Fatalf("expected interpolated bool/platform values, got args: %#v", spec.Args)
	}
	if spec.Env["WELINK_PLATFORM"] != "macos" {
		t.Fatalf("unexpected platform env: %q", spec.Env["WELINK_PLATFORM"])
	}
	if spec.Env["WELINK_AUTO_REFRESH"] != "false" || spec.Env["WELINK_WAL_ENABLED"] != "true" {
		t.Fatalf("unexpected bool env values: %#v", spec.Env)
	}
}

func TestNormalizePlatformAutoUsesCurrentOS(t *testing.T) {
	got := NormalizePlatform("")
	switch runtime.GOOS {
	case "windows":
		if got != "windows" {
			t.Fatalf("expected windows, got %q", got)
		}
	case "darwin":
		if got != "macos" {
			t.Fatalf("expected macos, got %q", got)
		}
	default:
		if got != "linux" {
			t.Fatalf("expected linux, got %q", got)
		}
	}
}
