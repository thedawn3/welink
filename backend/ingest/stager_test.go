package ingest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStageWeChatDataCopiesExpectedFiles(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()

	mustWriteFile(t, filepath.Join(source, "contact", "contact.db"), "contact")
	mustWriteFile(t, filepath.Join(source, "message", "message_0.db"), "db0")
	mustWriteFile(t, filepath.Join(source, "message", "message_0.db-wal"), "wal0")
	mustWriteFile(t, filepath.Join(source, "message", "message_0.db-shm"), "shm0")
	mustWriteFile(t, filepath.Join(source, "message", "resource.db"), "ignore")

	result, err := StageWeChatData(StageOptions{
		SourceDir:   source,
		TargetDir:   target,
		PreserveWAL: true,
	})
	if err != nil {
		t.Fatalf("stage wechat data: %v", err)
	}
	if len(result.CopiedFiles) != 4 {
		t.Fatalf("expected 4 copied files, got %d (%v)", len(result.CopiedFiles), result.CopiedFiles)
	}

	for _, rel := range []string{
		filepath.Join("contact", "contact.db"),
		filepath.Join("message", "message_0.db"),
		filepath.Join("message", "message_0.db-wal"),
		filepath.Join("message", "message_0.db-shm"),
	} {
		if _, err := os.Stat(filepath.Join(target, rel)); err != nil {
			t.Fatalf("expected staged file %s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(target, "message", "resource.db")); !os.IsNotExist(err) {
		t.Fatalf("expected resource.db to be ignored, got err=%v", err)
	}
}

func TestStageWeChatDataWithoutWALSkipsWALFiles(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()

	mustWriteFile(t, filepath.Join(source, "contact", "contact.db"), "contact")
	mustWriteFile(t, filepath.Join(source, "message", "message_1.db"), "db1")
	mustWriteFile(t, filepath.Join(source, "message", "message_1.db-wal"), "wal1")

	result, err := StageWeChatData(StageOptions{
		SourceDir:   source,
		TargetDir:   target,
		PreserveWAL: false,
	})
	if err != nil {
		t.Fatalf("stage wechat data: %v", err)
	}
	if len(result.CopiedFiles) != 2 {
		t.Fatalf("expected 2 copied files, got %d (%v)", len(result.CopiedFiles), result.CopiedFiles)
	}
	if _, err := os.Stat(filepath.Join(target, "message", "message_1.db-wal")); !os.IsNotExist(err) {
		t.Fatalf("expected wal file to be skipped, got err=%v", err)
	}
}

func TestStageWeChatDataCopiesOptionalSNSDB(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()

	mustWriteFile(t, filepath.Join(source, "contact", "contact.db"), "contact")
	mustWriteFile(t, filepath.Join(source, "message", "message_2.db"), "db2")
	mustWriteFile(t, filepath.Join(source, "sns", "sns.db"), "sns")

	result, err := StageWeChatData(StageOptions{
		SourceDir: source,
		TargetDir: target,
	})
	if err != nil {
		t.Fatalf("stage wechat data: %v", err)
	}
	if len(result.CopiedFiles) != 3 {
		t.Fatalf("expected 3 copied files, got %d (%v)", len(result.CopiedFiles), result.CopiedFiles)
	}
	if _, err := os.Stat(filepath.Join(target, "sns", "sns.db")); err != nil {
		t.Fatalf("expected sns/sns.db to be copied: %v", err)
	}
}

func TestStageWeChatDataAllowsContactOnlySource(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()

	mustWriteFile(t, filepath.Join(source, "contact", "contact.db"), "contact")

	result, err := StageWeChatData(StageOptions{
		SourceDir: source,
		TargetDir: target,
	})
	if err != nil {
		t.Fatalf("stage wechat data: %v", err)
	}
	if len(result.CopiedFiles) != 1 || result.CopiedFiles[0] != filepath.Join("contact", "contact.db") {
		t.Fatalf("unexpected copied files: %#v", result.CopiedFiles)
	}
}

func TestStageWeChatDataValidatesInputsAndContent(t *testing.T) {
	target := t.TempDir()

	if _, err := StageWeChatData(StageOptions{TargetDir: target}); err == nil {
		t.Fatal("expected source dir validation error")
	}
	if _, err := StageWeChatData(StageOptions{SourceDir: t.TempDir()}); err == nil {
		t.Fatal("expected target dir validation error")
	}

	emptySource := t.TempDir()
	if _, err := StageWeChatData(StageOptions{
		SourceDir: emptySource,
		TargetDir: target,
	}); err == nil {
		t.Fatal("expected missing data error")
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
