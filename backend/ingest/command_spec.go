package ingest

import (
	"fmt"
	"runtime"
	"strings"
)

type CommandSpec struct {
	Platform        string
	CommandTemplate string
	SourceDataDir   string
	AnalysisDataDir string
	WorkDir         string
	AutoRefresh     bool
	WALEnabled      bool
}

func NormalizePlatform(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "windows", "win":
		return "windows"
	case "macos", "darwin", "mac", "osx":
		return "macos"
	case "linux":
		return "linux"
	case "", "auto":
		switch runtime.GOOS {
		case "windows":
			return "windows"
		case "darwin":
			return "macos"
		default:
			return "linux"
		}
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func ResolveCommandSpec(spec CommandSpec) (StartOptions, error) {
	platform := NormalizePlatform(spec.Platform)
	commandText := strings.TrimSpace(spec.CommandTemplate)
	if commandText == "" {
		return StartOptions{}, fmt.Errorf("no decrypt command configured for %s", platform)
	}

	replacements := map[string]string{
		"${platform}":          platform,
		"${source_data_dir}":   spec.SourceDataDir,
		"${analysis_data_dir}": spec.AnalysisDataDir,
		"${work_dir}":          spec.WorkDir,
		"${auto_refresh}":      boolString(spec.AutoRefresh),
		"${wal_enabled}":       boolString(spec.WALEnabled),
	}
	for old, newValue := range replacements {
		commandText = strings.ReplaceAll(commandText, old, newValue)
	}

	command, args := shellCommand(platform, commandText)
	return StartOptions{
		Name:    "decrypt",
		Command: command,
		Args:    args,
		WorkDir: spec.WorkDir,
		Env: map[string]string{
			"WELINK_PLATFORM":          platform,
			"WELINK_SOURCE_DATA_DIR":   spec.SourceDataDir,
			"WELINK_ANALYSIS_DATA_DIR": spec.AnalysisDataDir,
			"WELINK_WORK_DIR":          spec.WorkDir,
			"WELINK_AUTO_REFRESH":      boolString(spec.AutoRefresh),
			"WELINK_WAL_ENABLED":       boolString(spec.WALEnabled),
		},
	}, nil
}

func shellCommand(platform, commandText string) (string, []string) {
	if platform == "windows" {
		return "cmd", []string{"/C", commandText}
	}
	return "sh", []string{"-lc", commandText}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
