package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// ─── MCP JSON-RPC 协议结构 ───────────────────────────────────────────

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// ─── MCP 协议相关类型 ────────────────────────────────────────────────

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Capabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct{}

type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
	Capabilities    Capabilities `json:"capabilities"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type CallToolResult struct {
	Content []TextContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ─── Server 结构体 ───────────────────────────────────────────────────

type Server struct {
	fetch  func(path string, params map[string]string) (string, error)
	postFn func(path string, body map[string]any) (string, error)
}

func NewServer(backendURL string) *Server {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy: nil,
		},
	}
	return &Server{
		fetch: func(path string, params map[string]string) (string, error) {
			return apiGetWithClient(client, backendURL, path, params)
		},
		postFn: func(path string, body map[string]any) (string, error) {
			return apiPostWithClient(client, backendURL, path, body)
		},
	}
}

// ─── WeLink API 调用 ─────────────────────────────────────────────────

func apiGetWithClient(client *http.Client, baseURL string, path string, params map[string]string) (string, error) {
	u, err := url.Parse(baseURL + path)
	if err != nil {
		return "", err
	}
	if len(params) > 0 {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}
	resp, err := client.Get(u.String())
	if err != nil {
		return "", fmt.Errorf("WeLink 后端未启动或无法访问: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func apiPostWithClient(client *http.Client, baseURL string, path string, payload map[string]any) (string, error) {
	u, err := url.Parse(baseURL + path)
	if err != nil {
		return "", err
	}
	if payload == nil {
		payload = map[string]any{}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("请求体编码失败: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, u.String(), strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("WeLink 后端未启动或无法访问: %v", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(respBody), nil
}

func formatJSON(raw string) string {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return raw
	}
	return string(b)
}

// ─── Tool 定义 ───────────────────────────────────────────────────────

var tools = []Tool{
	{
		Name:        "get_contact_stats",
		Description: "获取所有微信联系人的消息统计排名，包括总消息数、对方消息数、我的消息数、首次和最后一次聊天时间。用于回答「我和谁联系最多」、「谁是我最常聊的人」等问题。",
		InputSchema: InputSchema{
			Type: "object",
		},
	},
	{
		Name:        "get_contact_detail",
		Description: "获取某个微信联系人的深度分析，包括每小时/每周消息分布、深夜消息数、红包数、主动发起对话比例、聊天热度（热/温/冷）等。用于分析与某人的关系深度。",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"username": {Type: "string", Description: "联系人的微信 username，如 wxid_xxxxx 或 12345678@chatroom"},
			},
			Required: []string{"username"},
		},
	},
	{
		Name:        "get_contact_wordcloud",
		Description: "获取与某个联系人聊天的高频词汇，用于了解双方经常聊的话题。",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"username":     {Type: "string", Description: "联系人的微信 username"},
				"include_mine": {Type: "string", Description: "是否包含我发送的消息，true 或 false，默认 false"},
			},
			Required: []string{"username"},
		},
	},
	{
		Name:        "get_contact_sentiment",
		Description: "获取与某个联系人聊天的情感趋势分析，按月统计正面/负面/中性消息占比，用于了解关系情感变化。",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"username":     {Type: "string", Description: "联系人的微信 username"},
				"include_mine": {Type: "string", Description: "是否包含我发送的消息，true 或 false，默认 false"},
			},
			Required: []string{"username"},
		},
	},
	{
		Name:        "get_contact_messages",
		Description: "获取与某个联系人某一天的聊天记录。",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"username": {Type: "string", Description: "联系人的微信 username"},
				"date":     {Type: "string", Description: "日期，格式 YYYY-MM-DD，如 2024-03-15"},
			},
			Required: []string{"username", "date"},
		},
	},
	{
		Name:        "get_global_stats",
		Description: "获取微信数据全局统计，包括总好友数、总消息数、最忙的一天、每月消息趋势、24小时热力图、消息类型分布、深夜聊天排行等。用于回答总体社交数据问题。",
		InputSchema: InputSchema{
			Type: "object",
		},
	},
	{
		Name:        "get_groups",
		Description: "获取所有微信群聊列表及其消息统计。",
		InputSchema: InputSchema{
			Type: "object",
		},
	},
	{
		Name:        "get_group_detail",
		Description: "获取某个群聊的深度分析，包括成员发言排名、活跃时间分布、高频词汇等。",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"username": {Type: "string", Description: "群聊的微信 username，格式通常为 xxxxx@chatroom"},
			},
			Required: []string{"username"},
		},
	},
	{
		Name:        "get_stats_by_timerange",
		Description: "按时间范围过滤统计数据，分析特定时期的社交情况。",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"from": {Type: "string", Description: "开始时间，Unix 秒时间戳"},
				"to":   {Type: "string", Description: "结束时间，Unix 秒时间戳"},
			},
			Required: []string{"from", "to"},
		},
	},
	{
		Name:        "get_runtime_status",
		Description: "获取 WeLink 运行时状态，包括解密任务状态、索引状态、revision 与最近错误信息。",
		InputSchema: InputSchema{
			Type: "object",
		},
	},
	{
		Name:        "start_decrypt",
		Description: "启动解密任务。优先调用新的 /api/system/decrypt/start 接口；适用于 Windows 自动解密或 macOS 编排解密。",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"command":           {Type: "string", Description: "覆盖默认解密命令模板（可选）"},
				"source_data_dir":   {Type: "string", Description: "原始数据目录（可选）"},
				"work_dir":          {Type: "string", Description: "解密工作目录（可选）"},
				"analysis_data_dir": {Type: "string", Description: "分析数据目录（可选）"},
				"platform":          {Type: "string", Description: "平台标识（可选：windows/macos）"},
				"auto_refresh":      {Type: "string", Description: "是否自动刷新（可选，true/false）"},
				"wal_enabled":       {Type: "string", Description: "是否启用 WAL 兼容（可选，true/false）"},
			},
		},
	},
	{
		Name:        "stop_decrypt",
		Description: "停止当前解密任务。",
		InputSchema: InputSchema{
			Type: "object",
		},
	},
	{
		Name:        "rebuild_index",
		Description: "触发索引重建，调用 /api/system/reindex。",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"from": {Type: "string", Description: "开始时间，Unix 秒时间戳（可选）"},
				"to":   {Type: "string", Description: "结束时间，Unix 秒时间戳（可选）"},
			},
		},
	},
	{
		Name:        "get_recent_changes",
		Description: "获取近期数据变化信息（revision、最近变更时间、待处理变化等）。",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"since_revision": {Type: "string", Description: "从哪个 revision 开始查询（可选）"},
			},
		},
	},
	{
		Name:        "export_chatlab",
		Description: "导出 ChatLab 标准格式。支持联系人、群聊、搜索结果导出。",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"scope":        {Type: "string", Description: "导出范围（contact/group/search）"},
				"username":     {Type: "string", Description: "联系人或群 username（scope=contact/group 时常用）"},
				"query":        {Type: "string", Description: "搜索关键词（scope=search 时常用）"},
				"date":         {Type: "string", Description: "群聊导出日期（可选，YYYY-MM-DD）"},
				"include_mine": {Type: "string", Description: "是否包含我发送的消息（可选，true/false）"},
				"limit":        {Type: "string", Description: "导出条数上限（可选）"},
			},
			Required: []string{"scope"},
		},
	},
}

// ─── Tool 调用处理 ───────────────────────────────────────────────────

func (s *Server) callTool(name string, args json.RawMessage) CallToolResult {
	argMap, err := parseArgs(args)
	if err != nil {
		return errorResult("参数解析失败: " + err.Error())
	}
	if argMap == nil {
		argMap = map[string]any{}
	}

	var (
		raw string
	)

	switch name {
	case "get_contact_stats":
		raw, err = s.get("/api/contacts/stats", nil)

	case "get_contact_detail":
		username := getStringArg(argMap, "username")
		if username == "" {
			return errorResult("缺少参数: username")
		}
		raw, err = s.get("/api/contacts/detail", map[string]string{"username": username})

	case "get_contact_wordcloud":
		username := getStringArg(argMap, "username")
		if username == "" {
			return errorResult("缺少参数: username")
		}
		params := map[string]string{"username": username}
		if includeMine := getStringArg(argMap, "include_mine"); includeMine != "" {
			params["include_mine"] = includeMine
		}
		raw, err = s.get("/api/contacts/wordcloud", params)

	case "get_contact_sentiment":
		username := getStringArg(argMap, "username")
		if username == "" {
			return errorResult("缺少参数: username")
		}
		params := map[string]string{"username": username}
		if includeMine := getStringArg(argMap, "include_mine"); includeMine != "" {
			params["include_mine"] = includeMine
		}
		raw, err = s.get("/api/contacts/sentiment", params)

	case "get_contact_messages":
		username := getStringArg(argMap, "username")
		date := getStringArg(argMap, "date")
		if username == "" || date == "" {
			return errorResult("缺少参数: username 和 date")
		}
		raw, err = s.get("/api/contacts/messages", map[string]string{
			"username": username,
			"date":     date,
		})

	case "get_global_stats":
		raw, err = s.get("/api/global", nil)

	case "get_groups":
		raw, err = s.get("/api/groups", nil)

	case "get_group_detail":
		username := getStringArg(argMap, "username")
		if username == "" {
			return errorResult("缺少参数: username")
		}
		raw, err = s.get("/api/groups/detail", map[string]string{"username": username})

	case "get_stats_by_timerange":
		from := getStringArg(argMap, "from")
		to := getStringArg(argMap, "to")
		if from == "" || to == "" {
			return errorResult("缺少参数: from 和 to")
		}
		if _, e := strconv.ParseInt(from, 10, 64); e != nil {
			return errorResult("from 必须是 Unix 时间戳（整数）")
		}
		if _, e := strconv.ParseInt(to, 10, 64); e != nil {
			return errorResult("to 必须是 Unix 时间戳（整数）")
		}
		raw, err = s.get("/api/stats/filter", map[string]string{"from": from, "to": to})

	case "get_runtime_status":
		raw, err = s.get("/api/system/runtime", nil)
		if err != nil {
			return errorResult(err.Error())
		}
		if looksLikeMissingEndpoint(raw) {
			return errorResult("后端尚未提供 /api/system/runtime，请先完成 system runtime 接口落地。")
		}

	case "start_decrypt":
		payload := buildDecryptPayload(argMap)
		raw, err = s.post("/api/system/decrypt/start", payload)
		if err != nil {
			return errorResult(err.Error())
		}
		if looksLikeMissingEndpoint(raw) {
			return errorResult("后端尚未提供 /api/system/decrypt/start，请先完成后端 system 接口落地。")
		}

	case "stop_decrypt":
		raw, err = s.post("/api/system/decrypt/stop", map[string]any{})
		if err != nil {
			return errorResult(err.Error())
		}
		if looksLikeMissingEndpoint(raw) {
			return errorResult("后端尚未提供 /api/system/decrypt/stop，请先完成后端 system 接口落地。")
		}

	case "rebuild_index":
		payload := map[string]any{
			"from": getInt64ArgOrDefault(argMap, "from", 0),
			"to":   getInt64ArgOrDefault(argMap, "to", 0),
		}
		raw, err = s.post("/api/system/reindex", payload)
		if err != nil {
			return errorResult(err.Error())
		}
		if looksLikeMissingEndpoint(raw) {
			return errorResult("后端尚未提供 /api/system/reindex，请先完成 system reindex 接口落地。")
		}

	case "get_recent_changes":
		params := map[string]string{}
		if since := getStringArg(argMap, "since_revision"); since != "" {
			params["since_revision"] = since
		}
		raw, err = s.get("/api/system/changes", params)
		if err != nil {
			return errorResult(err.Error())
		}
		if looksLikeMissingEndpoint(raw) {
			return errorResult("后端尚未提供 /api/system/changes，请先完成 system changes 接口落地。")
		}

	case "export_chatlab":
		scope := getStringArg(argMap, "scope")
		if scope == "" {
			return errorResult("缺少参数: scope")
		}
		params := map[string]string{}
		if limit, ok := getOptionalInt64Arg(argMap, "limit"); ok {
			params["limit"] = strconv.FormatInt(limit, 10)
		}
		switch scope {
		case "contact":
			username := getStringArg(argMap, "username")
			if username == "" {
				return errorResult("scope=contact 时缺少参数: username")
			}
			params["username"] = username
			raw, err = s.get("/api/export/chatlab/contact", params)
		case "group":
			username := getStringArg(argMap, "username")
			if username == "" {
				return errorResult("scope=group 时缺少参数: username")
			}
			params["username"] = username
			if date := getStringArg(argMap, "date"); date != "" {
				params["date"] = date
			}
			raw, err = s.get("/api/export/chatlab/group", params)
		case "search":
			query := getStringArg(argMap, "query")
			if query == "" {
				query = getStringArg(argMap, "q")
			}
			if query == "" {
				return errorResult("scope=search 时缺少参数: query")
			}
			params["q"] = query
			if includeMine, ok := getOptionalBoolArg(argMap, "include_mine"); ok {
				params["include_mine"] = strconv.FormatBool(includeMine)
			}
			raw, err = s.get("/api/export/chatlab/search", params)
		default:
			return errorResult("scope 必须是 contact/group/search")
		}
		if err != nil {
			return errorResult(err.Error())
		}
		if looksLikeMissingEndpoint(raw) {
			return errorResult("后端尚未提供 ChatLab 导出接口，请先完成导出接口落地。")
		}

	default:
		return errorResult("未知工具: " + name)
	}

	if err != nil {
		return errorResult(err.Error())
	}

	return CallToolResult{
		Content: []TextContent{{Type: "text", Text: formatJSON(raw)}},
	}
}

func parseArgs(args json.RawMessage) (map[string]any, error) {
	if len(args) > 0 {
		var argMap map[string]any
		if err := json.Unmarshal(args, &argMap); err != nil {
			return nil, err
		}
		return argMap, nil
	}
	return map[string]any{}, nil
}

func (s *Server) get(path string, params map[string]string) (string, error) {
	if s.fetch == nil {
		return "", errors.New("GET 客户端未初始化")
	}
	return s.fetch(path, params)
}

func (s *Server) post(path string, payload map[string]any) (string, error) {
	if s.postFn == nil {
		return "", errors.New("POST 客户端未初始化")
	}
	return s.postFn(path, payload)
}

func getStringArg(argMap map[string]any, key string) string {
	value, ok := argMap[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func getOptionalInt64Arg(argMap map[string]any, key string) (int64, bool) {
	value := getStringArg(argMap, key)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func getOptionalBoolArg(argMap map[string]any, key string) (bool, bool) {
	value, ok := argMap[key]
	if !ok || value == nil {
		return false, false
	}
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		parsed, err := strconv.ParseBool(strings.TrimSpace(fmt.Sprint(v)))
		if err != nil {
			return false, false
		}
		return parsed, true
	}
}

func getInt64ArgOrDefault(argMap map[string]any, key string, fallback int64) int64 {
	if value, ok := getOptionalInt64Arg(argMap, key); ok {
		return value
	}
	return fallback
}

func buildDecryptPayload(argMap map[string]any) map[string]any {
	payload := map[string]any{}
	for _, key := range []string{
		"source_data_dir",
		"work_dir",
		"analysis_data_dir",
		"platform",
	} {
		if value := getStringArg(argMap, key); value != "" {
			payload[key] = value
		}
	}
	for _, key := range []string{"auto_refresh", "wal_enabled"} {
		if value, ok := getOptionalBoolArg(argMap, key); ok {
			payload[key] = value
		}
	}
	if command := getStringArg(argMap, "command"); command != "" {
		payload["command"] = command
	}
	return payload
}

func looksLikeMissingEndpoint(raw string) bool {
	text := strings.ToLower(strings.TrimSpace(raw))
	if text == "" {
		return false
	}
	return strings.Contains(text, "404 page not found") ||
		strings.Contains(text, "\"code\":404") ||
		strings.Contains(text, "\"status\":404") ||
		strings.Contains(text, "\"error\":\"not found\"") ||
		strings.Contains(text, "\"message\":\"not found\"")
}

func wrapNoticeJSON(notice string, raw string) string {
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return fmt.Sprintf(`{"notice":%q,"raw":%q}`, notice, raw)
	}
	out, err := json.Marshal(map[string]any{
		"notice": notice,
		"data":   decoded,
	})
	if err != nil {
		return raw
	}
	return string(out)
}

func errorResult(msg string) CallToolResult {
	return CallToolResult{
		Content: []TextContent{{Type: "text", Text: msg}},
		IsError: true,
	}
}

// ─── MCP 消息处理 ────────────────────────────────────────────────────

func (s *Server) handle(req Request) *Response {
	resp := &Response{JSONRPC: "2.0", ID: req.ID}

	switch req.Method {
	case "initialize":
		resp.Result = InitializeResult{
			ProtocolVersion: "2024-11-05",
			ServerInfo:      ServerInfo{Name: "welink-mcp", Version: "1.0.0"},
			Capabilities:    Capabilities{Tools: &ToolsCapability{}},
		}

	case "tools/list":
		resp.Result = ListToolsResult{Tools: tools}

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &Error{Code: -32602, Message: "Invalid params"}
			return resp
		}
		result := s.callTool(params.Name, params.Arguments)
		resp.Result = result

	case "notifications/initialized":
		return nil

	case "ping":
		resp.Result = struct{}{}

	default:
		resp.Error = &Error{Code: -32601, Message: "Method not found: " + req.Method}
	}

	return resp
}

// ─── 主循环（stdio JSON-RPC）────────────────────────────────────────

func main() {
	welinkURL := os.Getenv("WELINK_URL")
	if welinkURL == "" {
		welinkURL = "http://localhost:8080"
	}
	srv := NewServer(welinkURL)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			encoder.Encode(Response{
				JSONRPC: "2.0",
				Error:   &Error{Code: -32700, Message: "Parse error"},
			})
			continue
		}

		// Notification（无 id）不回复
		if req.ID == nil && req.Method == "notifications/initialized" {
			continue
		}

		resp := srv.handle(req)
		if resp != nil && req.ID != nil {
			encoder.Encode(resp)
		}
	}
}
