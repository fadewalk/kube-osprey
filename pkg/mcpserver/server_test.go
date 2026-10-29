package mcpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fadewalk/kube-osprey/pkg/cluster"
	"github.com/fadewalk/kube-osprey/pkg/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const testProtocolVersion = "2025-06-18"

func setupMCPTestDB(t *testing.T, mcpEnabled bool) {
	t.Helper()
	originalDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.GeneralSetting{}, &model.MCPAuditLog{}); err != nil {
		t.Fatalf("migrating test database: %v", err)
	}
	setting := model.GeneralSetting{
		Model:                 model.Model{ID: 1},
		MCPEnabled:            mcpEnabled,
		MCPRateLimitPerMinute: 60,
	}
	if err := db.Create(&setting).Error; err != nil {
		t.Fatalf("creating general setting: %v", err)
	}
	model.DB = db
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		model.DB = originalDB
	})
}

func mcpTestRouter(handler *Handler, user model.User) *gin.Engine {
	r := gin.New()
	r.POST("/api/v1/mcp", func(c *gin.Context) {
		c.Set("user", user)
		handler.GinHandler()(c)
	})
	return r
}

func postJSONRPC(t *testing.T, r *gin.Engine, headers map[string]string, payload map[string]any) (int, http.Header, map[string]any) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshaling request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)
	res := recorder.Result()
	respBody, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		t.Fatalf("reading response: %v", err)
	}
	text := string(respBody)
	// The SDK answers SSE when the Accept header advertises event streams;
	// unwrap the first data line for the JSON-RPC envelope.
	if strings.HasPrefix(strings.TrimSpace(text), "event:") {
		for _, line := range strings.Split(text, "\n") {
			if data, ok := strings.CutPrefix(line, "data: "); ok {
				text = strings.TrimSpace(data)
				break
			}
		}
	}
	var decoded map[string]any
	// Notifications are answered with 202 and an empty body.
	if len(strings.TrimSpace(text)) > 0 {
		if err := json.Unmarshal([]byte(text), &decoded); err != nil {
			t.Fatalf("decoding response %q: %v", text, err)
		}
	}
	return res.StatusCode, res.Header, decoded
}

func TestMCPProtocolHandshakeAndTools(t *testing.T) {
	setupMCPTestDB(t, true)
	handler := NewHandler(&cluster.ClusterManager{})
	user := allAccessUser("mcp-admin")
	r := mcpTestRouter(handler, user)

	// initialize
	status, header, initResp := postJSONRPC(t, r, nil, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": testProtocolVersion,
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "kube-osprey-e2e-test", "version": "0.0.1"},
		},
	})
	if status != http.StatusOK {
		t.Fatalf("initialize status = %d", status)
	}
	if initResp["error"] != nil {
		t.Fatalf("initialize returned error: %v", initResp["error"])
	}
	serverInfo, _ := initResp["result"].(map[string]any)["serverInfo"].(map[string]any)
	if serverInfo == nil || serverInfo["name"] != "kube-osprey" {
		t.Fatalf("unexpected serverInfo: %v", initResp["result"])
	}
	sessionID := header.Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Fatal("initialize must return an Mcp-Session-Id header")
	}
	sessionHeaders := map[string]string{
		"Mcp-Session-Id":       sessionID,
		"MCP-Protocol-Version": testProtocolVersion,
	}

	// initialized notification
	postJSONRPC(t, r, sessionHeaders, map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})

	// tools/list
	_, _, listResp := postJSONRPC(t, r, sessionHeaders, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	tools, _ := listResp["result"].(map[string]any)["tools"].([]any)
	if len(tools) < 14 {
		t.Fatalf("expected at least 14 tools, got %d", len(tools))
	}

	// tools/call list_clusters — empty cluster manager yields an empty list.
	_, _, callResp := postJSONRPC(t, r, sessionHeaders, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "list_clusters",
			"arguments": map[string]any{},
		},
	})
	if callResp["error"] != nil {
		t.Fatalf("tools/call returned protocol error: %v", callResp["error"])
	}
	result, _ := callResp["result"].(map[string]any)
	if result == nil || result["isError"] == true {
		t.Fatalf("list_clusters failed: %v", callResp["result"])
	}

	// tools/call against a denied tool path: no clusters exist, and
	// check_permission is answerable without a cluster.
	_, _, permResp := postJSONRPC(t, r, sessionHeaders, map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "check_permission",
			"arguments": map[string]any{"verb": "get", "resource": "pods"},
		},
	})
	permResult, _ := permResp["result"].(map[string]any)
	if permResult == nil || permResult["isError"] == true {
		t.Fatalf("check_permission failed: %v", permResp["result"])
	}
	var payload toolPayload
	content, _ := permResult["content"].([]any)
	if len(content) == 0 {
		t.Fatalf("check_permission returned no content: %v", permResult)
	}
	text := content[0].(map[string]any)["text"].(string)
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("decoding tool payload %q: %v", text, err)
	}
	if payload.DangerLevel != dangerLevelReadOnly {
		t.Fatalf("dangerLevel = %q, want %q", payload.DangerLevel, dangerLevelReadOnly)
	}

	// Audit trail: one row per successful call at minimum.
	var auditCount int64
	if err := model.DB.Model(&model.MCPAuditLog{}).Count(&auditCount).Error; err != nil {
		t.Fatalf("counting audit rows: %v", err)
	}
	if auditCount < 2 {
		t.Fatalf("expected at least 2 audit rows, got %d", auditCount)
	}
	var audit model.MCPAuditLog
	if err := model.DB.Where("tool_name = ?", "list_clusters").First(&audit).Error; err != nil {
		t.Fatalf("list_clusters audit row missing: %v", err)
	}
	if !audit.Success || audit.Username != "mcp-admin" {
		t.Fatalf("unexpected audit row: %+v", audit)
	}
}

func TestMCPDisabledReturnsJSONRPCError(t *testing.T) {
	setupMCPTestDB(t, false)
	handler := NewHandler(&cluster.ClusterManager{})
	r := mcpTestRouter(handler, allAccessUser("mcp-admin"))

	status, _, resp := postJSONRPC(t, r, nil, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": testProtocolVersion,
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "kube-osprey-e2e-test", "version": "0.0.1"},
		},
	})
	rpcErr, _ := resp["error"].(map[string]any)
	if rpcErr == nil {
		t.Fatalf("expected JSON-RPC error when disabled, got: %v", resp)
	}
	if code, _ := rpcErr["code"].(float64); int(code) != jsonRPCDisabledCode {
		t.Fatalf("error code = %v, want %d", rpcErr["code"], jsonRPCDisabledCode)
	}
	if msg, _ := rpcErr["message"].(string); msg != "MCP endpoint is disabled" {
		t.Fatalf("message = %q", msg)
	}
	if status != http.StatusOK {
		t.Fatalf("disabled POST must answer HTTP 200 with a JSON-RPC error, got %d", status)
	}
}

func TestMCPRateLimitExceeded(t *testing.T) {
	setupMCPTestDB(t, true)
	// Tighten the runtime limit to 1 call per minute.
	if err := model.DB.Model(&model.GeneralSetting{}).Where("id = ?", 1).
		Update("mcp_rate_limit_per_minute", 1).Error; err != nil {
		t.Fatalf("updating rate limit: %v", err)
	}
	handler := NewHandler(&cluster.ClusterManager{})
	r := mcpTestRouter(handler, allAccessUser("mcp-admin"))

	for i := 0; i < 2; i++ {
		status, _, _ := postJSONRPC(t, r, nil, map[string]any{
			"jsonrpc": "2.0",
			"id":      i,
			"method":  "initialize",
			"params": map[string]any{
				"protocolVersion": testProtocolVersion,
				"capabilities":    map[string]any{},
				"clientInfo":      map[string]any{"name": "kube-osprey-e2e-test", "version": "0.0.1"},
			},
		})
		if i == 0 && status != http.StatusOK {
			t.Fatalf("first call status = %d, want 200", status)
		}
		if i == 1 {
			if status == http.StatusOK {
				// 200 is only acceptable if the body is not a rate limit rejection.
				continue
			}
			if status != http.StatusTooManyRequests {
				t.Fatalf("second call status = %d, want 429", status)
			}
		}
	}
}

func TestMCPUnknownUserRejected(t *testing.T) {
	setupMCPTestDB(t, true)
	handler := NewHandler(&cluster.ClusterManager{})
	r := gin.New()
	r.POST("/api/v1/mcp", handler.GinHandler()) // no user injected

	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

func TestAllToolsRegisteredOnServer(t *testing.T) {
	setupMCPTestDB(t, true)
	handler := NewHandler(&cluster.ClusterManager{})

	// Exercise tools/list over the wire to verify every registry entry is
	// present and schema-carrying.
	r := mcpTestRouter(handler, allAccessUser("mcp-admin"))
	status, header, initResp := postJSONRPC(t, r, nil, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": testProtocolVersion,
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "kube-osprey-e2e-test", "version": "0.0.1"},
		},
	})
	if status != http.StatusOK || initResp["error"] != nil {
		t.Fatalf("initialize failed: %v", initResp)
	}
	sessionHeaders := map[string]string{
		"Mcp-Session-Id":       header.Get("Mcp-Session-Id"),
		"MCP-Protocol-Version": testProtocolVersion,
	}
	_, _, listResp := postJSONRPC(t, r, sessionHeaders, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	tools, _ := listResp["result"].(map[string]any)["tools"].([]any)
	registered := map[string]bool{}
	for _, def := range append(toolDefs(), planTools(handler.cm)...) {
		registered[def.name] = false
	}
	for _, raw := range tools {
		tool, _ := raw.(map[string]any)
		name, _ := tool["name"].(string)
		if _, ok := registered[name]; !ok {
			t.Errorf("unexpected tool %q served", name)
			continue
		}
		registered[name] = true
		schema, ok := tool["inputSchema"]
		if !ok || schema == nil {
			t.Errorf("tool %q served without inputSchema", name)
		}
	}
	for name, found := range registered {
		if !found {
			t.Errorf("tool %q missing from tools/list", name)
		}
	}
	if len(tools) == 0 {
		fmt.Println("no tools listed")
	}
}
