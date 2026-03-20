package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"welink/backend/config"
	"welink/backend/pkg/db"
	"welink/backend/pkg/seed"
	"welink/backend/service"

	"github.com/gin-gonic/gin"
)

func TestExportChatLabRoutesReturn503WhenAnalysisNotReady(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	api := router.Group("/api")
	registerExportRoutes(api, &service.ContactService{}, func() bool { return false })

	req := httptest.NewRequest(http.MethodGet, "/api/export/chatlab/contact?username=alice_wx", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "analysis data not ready") {
		t.Fatalf("expected not ready error, got %s", rec.Body.String())
	}
}

func TestExportChatLabGroupInfersDateAndDownloadHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	contactSvc := newSeededExportContactService(t)
	router := gin.New()
	api := router.Group("/api")
	registerExportRoutes(api, contactSvc, func() bool { return true })

	req := httptest.NewRequest(http.MethodGet, "/api/export/chatlab/group?username=teamwork2024@chatroom&download=true", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	disposition := rec.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "attachment;") || !strings.Contains(disposition, ".chatlab.json") {
		t.Fatalf("expected attachment filename, got %q", disposition)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	fileName, _ := payload["file_name"].(string)
	if !strings.Contains(fileName, "teamwork2024_chatroom-") || !strings.HasSuffix(fileName, ".chatlab.json") {
		t.Fatalf("expected inferred group filename, got %q", fileName)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data envelope, got %#v", payload["data"])
	}
	meta, ok := data["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected chatlab meta, got %#v", data["meta"])
	}
	if gotType, _ := meta["type"].(string); gotType != "group" {
		t.Fatalf("expected group export type, got %#v", meta["type"])
	}
	summary, ok := payload["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected summary envelope, got %#v", payload["summary"])
	}
	if scope, _ := summary["scope"].(string); scope != "group" {
		t.Fatalf("expected scope=group, got %#v", summary["scope"])
	}
	if conv, _ := summary["conversation_name"].(string); conv == "" {
		t.Fatalf("expected conversation_name, got %#v", summary["conversation_name"])
	}
	if _, ok := summary["message_count"].(float64); !ok {
		t.Fatalf("expected message_count number, got %#v", summary["message_count"])
	}
	if _, ok := summary["member_count"].(float64); !ok {
		t.Fatalf("expected member_count number, got %#v", summary["member_count"])
	}
}

func TestExportChatLabPostSupportsQueryAliasQ(t *testing.T) {
	gin.SetMode(gin.TestMode)

	contactSvc := newSeededExportContactService(t)
	router := gin.New()
	api := router.Group("/api")
	registerExportRoutes(api, contactSvc, func() bool { return true })

	body := []byte(`{"scope":"search","q":"哈哈","include_mine":"true","limit":20}`)
	req := httptest.NewRequest(http.MethodPost, "/api/export/chatlab", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	fileName, _ := payload["file_name"].(string)
	if !strings.Contains(fileName, "search-哈哈") {
		t.Fatalf("expected search filename to include query, got %q", fileName)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data envelope, got %#v", payload["data"])
	}
	meta, ok := data["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected chatlab meta, got %#v", data["meta"])
	}
	if gotName, _ := meta["name"].(string); gotName != "搜索结果" {
		t.Fatalf("expected search export name, got %#v", meta["name"])
	}
	summary, ok := payload["summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected summary envelope, got %#v", payload["summary"])
	}
	if scope, _ := summary["scope"].(string); scope != "search" {
		t.Fatalf("expected scope=search, got %#v", summary["scope"])
	}
	if q, _ := summary["query"].(string); q != "哈哈" {
		t.Fatalf("expected query 哈哈, got %#v", summary["query"])
	}
	if includeMine, _ := summary["include_mine"].(bool); !includeMine {
		t.Fatalf("expected include_mine true, got %#v", summary["include_mine"])
	}
	if limit, _ := summary["limit"].(float64); limit != 20 {
		t.Fatalf("expected limit 20, got %#v", summary["limit"])
	}
}

func newSeededExportContactService(t *testing.T) *service.ContactService {
	t.Helper()

	root := t.TempDir()
	if err := seed.Generate(root); err != nil {
		t.Fatalf("seed generate: %v", err)
	}
	dbMgr, err := db.NewDBManager(root)
	if err != nil {
		t.Fatalf("new db manager: %v", err)
	}

	cfg := &config.Config{}
	contactSvc := service.NewContactService(dbMgr, cfg)
	contactSvc.Reinitialize(0, 0)
	waitForCondition(t, 5*time.Second, func() bool {
		status := contactSvc.GetStatus()
		initialized, _ := status["is_initialized"].(bool)
		return initialized
	}, "seeded contact service initialization")
	return contactSvc
}
