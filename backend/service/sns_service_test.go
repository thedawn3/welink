package service

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestParseTimelineContent(t *testing.T) {
	raw := `<SnsDataItem><TimelineObject><id>123</id><username>wxid_a</username><createTime>1700000000</createTime><contentDesc>hello</contentDesc><LocalExtraInfo><nickname>Alice</nickname></LocalExtraInfo></TimelineObject></SnsDataItem>`
	feedID, username, createdAt, contentText, displayName := parseTimelineContent(raw, "fallback_feed", "fallback_user")
	if feedID != "123" || username != "wxid_a" || createdAt != 1700000000 || contentText != "hello" || displayName != "Alice" {
		t.Fatalf("unexpected parse result: feed=%q user=%q ts=%d content=%q display=%q", feedID, username, createdAt, contentText, displayName)
	}
}

func TestParseSNSBound_DateEndOfDay(t *testing.T) {
	svc := &ContactService{tz: time.FixedZone("CST", 8*3600)}
	from, err := svc.parseSNSBound("2025-01-01", false)
	if err != nil {
		t.Fatalf("parse from failed: %v", err)
	}
	to, err := svc.parseSNSBound("2025-01-01", true)
	if err != nil {
		t.Fatalf("parse to failed: %v", err)
	}
	if to <= from {
		t.Fatalf("expected to > from, got from=%d to=%d", from, to)
	}
}

func TestSearchSNSIndex_DedupAndFallbackContent(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "sns.db"))
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE SnsTopItem_1(
		tid TEXT,
		username TEXT,
		summary TEXT,
		create_time INTEGER,
		last_read_time INTEGER,
		is_read INTEGER
	)`)
	if err != nil {
		t.Fatalf("create table failed: %v", err)
	}
	_, err = db.Exec(`INSERT INTO SnsTopItem_1(tid, username, summary, create_time, last_read_time, is_read) VALUES
		('101', 'wxid_a', '', 1700000000, 1700000100, 1),
		('101', 'wxid_a', '', 1700000000, 1700000100, 1),
		('feed_102', 'wxid_b', '有摘要', 1700000200, 1700000300, 0)`)
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	svc := &ContactService{tz: time.FixedZone("CST", 8*3600)}
	items, err := svc.searchSNSIndex(db, map[string]string{"wxid_a": "A", "wxid_b": "B"}, "", "", 0, 0, 10)
	if err != nil {
		t.Fatalf("searchSNSIndex failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 deduped items, got %d", len(items))
	}
	if items[0].Kind != snsKindIndex {
		t.Fatalf("expected index kind, got %q", items[0].Kind)
	}
	if items[0].Username != "wxid_b" {
		t.Fatalf("expected newer item first, got %q", items[0].Username)
	}
	if items[0].FeedID != "feed_102" {
		t.Fatalf("expected string feed id to be preserved, got %q", items[0].FeedID)
	}
	if items[1].ContentText == "" || items[1].ContentText == items[0].ContentText && items[1].Username == items[0].Username {
		t.Fatalf("unexpected fallback content ordering: %#v", items)
	}
	if items[1].ContentText != "[朋友圈索引记录，正文未同步到 sns.db]" {
		t.Fatalf("expected fallback content text, got %q", items[1].ContentText)
	}
}

func TestSortSNSItems_PostsBeforeIndexAtSameTimestamp(t *testing.T) {
	items := []SnsSearchItem{
		{Kind: snsKindIndex, FeedID: "3", createdAtUnix: 100},
		{Kind: snsKindInteraction, FeedID: "2", createdAtUnix: 100},
		{Kind: snsKindPost, FeedID: "1", createdAtUnix: 100},
		{Kind: snsKindPost, FeedID: "9", createdAtUnix: 101},
	}
	sortSNSItems(items, true)

	if items[0].Kind != snsKindPost || items[0].FeedID != "9" {
		t.Fatalf("expected post bucket first, got %#v", items[0])
	}
	if items[1].Kind != snsKindPost || items[1].FeedID != "1" {
		t.Fatalf("expected remaining post next, got %#v", items)
	}
	if items[2].Kind != snsKindInteraction {
		t.Fatalf("expected interaction before index at same timestamp, got %#v", items)
	}
	if items[3].Kind != snsKindIndex {
		t.Fatalf("expected index last, got %#v", items)
	}
}
