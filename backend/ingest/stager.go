package ingest

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type StageOptions struct {
	SourceDir   string
	TargetDir   string
	PreserveWAL bool
}

type StageResult struct {
	CopiedFiles []string
}

func StageWeChatData(opts StageOptions) (StageResult, error) {
	if strings.TrimSpace(opts.SourceDir) == "" {
		return StageResult{}, fmt.Errorf("source dir is required")
	}
	if strings.TrimSpace(opts.TargetDir) == "" {
		return StageResult{}, fmt.Errorf("target dir is required")
	}

	sourceRoot := filepath.Clean(opts.SourceDir)
	targetRoot := filepath.Clean(opts.TargetDir)

	paths, err := collectStagePaths(sourceRoot, opts.PreserveWAL)
	if err != nil {
		return StageResult{}, err
	}

	result := StageResult{CopiedFiles: make([]string, 0, len(paths))}
	for _, rel := range paths {
		src := filepath.Join(sourceRoot, rel)
		dst := filepath.Join(targetRoot, rel)
		if err := copyFile(src, dst); err != nil {
			return StageResult{}, err
		}
		result.CopiedFiles = append(result.CopiedFiles, rel)
	}
	return result, nil
}

func collectStagePaths(root string, preserveWAL bool) ([]string, error) {
	paths := make([]string, 0)
	contactPath := filepath.Join(root, "contact", "contact.db")
	if info, err := os.Stat(contactPath); err == nil && !info.IsDir() {
		paths = append(paths, filepath.Join("contact", "contact.db"))
	}

	messageRoot := filepath.Join(root, "message")
	entries, err := os.ReadDir(messageRoot)
	if err != nil {
		if len(paths) == 0 {
			return nil, fmt.Errorf("message dir not found at %s", messageRoot)
		}
		return paths, nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lower := strings.ToLower(name)
		if strings.HasPrefix(lower, "message_") && strings.HasSuffix(lower, ".db") {
			paths = append(paths, filepath.Join("message", name))
			continue
		}
		if preserveWAL && strings.HasPrefix(lower, "message_") &&
			(strings.HasSuffix(lower, ".db-wal") || strings.HasSuffix(lower, ".db-shm")) {
			paths = append(paths, filepath.Join("message", name))
		}
	}

	snsPath := filepath.Join(root, "sns", "sns.db")
	if info, err := os.Stat(snsPath); err == nil && !info.IsDir() {
		paths = append(paths, filepath.Join("sns", "sns.db"))
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("no contact/message databases found under %s", root)
	}
	return paths, nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("prepare target dir failed: %w", err)
	}

	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file %s failed: %w", src, err)
	}
	defer sourceFile.Close()

	info, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("stat source file %s failed: %w", src, err)
	}

	targetFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create target file %s failed: %w", dst, err)
	}
	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		targetFile.Close()
		return fmt.Errorf("copy %s -> %s failed: %w", src, dst, err)
	}
	if err := targetFile.Close(); err != nil {
		return fmt.Errorf("close target file %s failed: %w", dst, err)
	}
	if err := os.Chtimes(dst, info.ModTime(), info.ModTime()); err != nil {
		return fmt.Errorf("preserve modtime for %s failed: %w", dst, err)
	}
	return os.Chmod(dst, info.Mode())
}
