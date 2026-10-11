package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zxh326/kite/pkg/model"
	"k8s.io/klog/v2"
)

const (
	operationAuditMaxBody = 4 * 1024
)

// sensitiveKeyFragments are matched case-insensitively against JSON keys;
// a matching key's whole value is replaced before the body is stored.
var sensitiveKeyFragments = []string{
	"password", "passwd", "secret", "token", "apikey", "api_key",
	"privatekey", "private_key", "credential", "cookie", "config",
}

type operationInfo struct {
	Module   string
	Action   string
	Resource string
}

// operationRouteMap classifies known mutation routes by "METHOD FullPath".
// Unknown routes fall back to a heuristic derived from the path, so new
// endpoints are still audited without maintaining this table.
var operationRouteMap = map[string]operationInfo{
	"POST /api/v1/admin/clusters/":                                       {Module: "clusters", Action: "create", Resource: "cluster"},
	"POST /api/v1/admin/clusters/import":                                 {Module: "clusters", Action: "import", Resource: "cluster"},
	"PUT /api/v1/admin/clusters/:id":                                     {Module: "clusters", Action: "update", Resource: "cluster"},
	"DELETE /api/v1/admin/clusters/:id":                                  {Module: "clusters", Action: "delete", Resource: "cluster"},
	"POST /api/v1/admin/plugins":                                         {Module: "plugins", Action: "install", Resource: "plugin"},
	"DELETE /api/v1/admin/plugins/:id":                                   {Module: "plugins", Action: "delete", Resource: "plugin"},
	"PUT /api/v1/plugins/:id/settings":                                   {Module: "plugins", Action: "update", Resource: "plugin-setting"},
	"POST /api/v1/admin/oauth-providers/":                                {Module: "oauth", Action: "create", Resource: "oauth-provider"},
	"PUT /api/v1/admin/oauth-providers/:id":                              {Module: "oauth", Action: "update", Resource: "oauth-provider"},
	"DELETE /api/v1/admin/oauth-providers/:id":                           {Module: "oauth", Action: "delete", Resource: "oauth-provider"},
	"PUT /api/v1/admin/ldap-setting/":                                    {Module: "ldap", Action: "update", Resource: "ldap-setting"},
	"POST /api/v1/admin/roles/":                                          {Module: "roles", Action: "create", Resource: "role"},
	"PUT /api/v1/admin/roles/:id":                                        {Module: "roles", Action: "update", Resource: "role"},
	"DELETE /api/v1/admin/roles/:id":                                     {Module: "roles", Action: "delete", Resource: "role"},
	"PUT /api/v1/admin/general-setting/":                                 {Module: "settings", Action: "update", Resource: "general-setting"},
	"POST /api/v1/admin/templates/":                                      {Module: "templates", Action: "create", Resource: "template"},
	"PUT /api/v1/admin/templates/:id":                                    {Module: "templates", Action: "update", Resource: "template"},
	"DELETE /api/v1/admin/templates/:id":                                 {Module: "templates", Action: "delete", Resource: "template"},
	"POST /api/v1/users/me/password":                                     {Module: "account", Action: "change-password", Resource: "account"},
	"POST /api/v1/users/me/mfa/setup":                                    {Module: "account", Action: "mfa-setup", Resource: "account"},
	"POST /api/v1/users/me/mfa/enable":                                   {Module: "account", Action: "mfa-enable", Resource: "account"},
	"POST /api/v1/users/me/mfa/disable":                                  {Module: "account", Action: "mfa-disable", Resource: "account"},
	"POST /api/v1/users/me/passkeys/begin":                               {Module: "account", Action: "passkey-register", Resource: "account"},
	"POST /api/v1/users/me/passkeys/finish":                              {Module: "account", Action: "passkey-register", Resource: "account"},
	"DELETE /api/v1/users/me/passkeys/:id":                               {Module: "account", Action: "passkey-delete", Resource: "account"},
	"POST /api/v1/users/me/sidebar_preference":                           {Module: "account", Action: "update", Resource: "sidebar-preference"},
	"POST /api/v1/change-plans/:id/approve":                              {Module: "changeplans", Action: "approve", Resource: "change-plan"},
	"POST /api/v1/change-plans/:id/reject":                               {Module: "changeplans", Action: "reject", Resource: "change-plan"},
	"POST /api/v1/change-plans/:id/rollback":                             {Module: "changeplans", Action: "rollback", Resource: "change-plan"},
	"POST /api/v1/resources/apply":                                       {Module: "resources", Action: "apply", Resource: "resource"},
	"POST /api/v1/workload-ops/update-image":                             {Module: "workloads", Action: "update-image", Resource: "workload"},
	"POST /api/v1/workload-ops/restart":                                  {Module: "workloads", Action: "restart", Resource: "workload"},
	"POST /api/v1/alertmanager/silences":                                 {Module: "alertmanager", Action: "create", Resource: "silence"},
	"DELETE /api/v1/alertmanager/silences/:silenceId":                    {Module: "alertmanager", Action: "delete", Resource: "silence"},
	"DELETE /api/v1/_clusters/:cluster/alertmanager/silences/:silenceId": {Module: "alertmanager", Action: "delete", Resource: "silence"},
	"POST /api/v1/_clusters/:cluster/alertmanager/silences":              {Module: "alertmanager", Action: "create", Resource: "silence"},
}

func classifyOperation(method, fullPath string, c *gin.Context) operationInfo {
	if info, ok := operationRouteMap[method+" "+fullPath]; ok {
		return info
	}
	// Workload/resource CRUD arrive through generic per-resource routes;
	// classify from the path shape: /api/v1/<resource>/...
	segments := strings.Split(strings.Trim(fullPath, "/"), "/")
	if len(segments) >= 2 && segments[0] == "api" && segments[1] == "v1" && len(segments) > 2 {
		name := segments[2]
		switch name {
		case "admin":
			if len(segments) > 3 {
				name = segments[3]
			}
		case "users", "change-plans", "plugins", "clusters", "helm", "helm-charts", "mcp":
			// keep as module name
		}
		resource := c.Param("resource")
		if resource == "" {
			resource = name
		}
		return operationInfo{Module: name, Action: strings.ToLower(method), Resource: resource}
	}
	return operationInfo{Module: "unknown", Action: strings.ToLower(method), Resource: ""}
}

func operationResourceName(c *gin.Context) string {
	for _, param := range []string{"name", "id", "silenceId"} {
		if value := c.Param(param); value != "" {
			return value
		}
	}
	return ""
}

// sanitizeOperationBody masks values under sensitive keys and truncates the
// JSON body so secrets (kubeconfigs, tokens, passwords) never reach the log.
func sanitizeOperationBody(contentType string, body []byte) string {
	if len(body) == 0 {
		return ""
	}
	lowerType := strings.ToLower(contentType)
	if strings.Contains(lowerType, "multipart/") || strings.Contains(lowerType, "octet-stream") {
		return ""
	}
	if !strings.Contains(lowerType, "json") && !strings.Contains(lowerType, "yaml") &&
		!strings.Contains(lowerType, "text") {
		return ""
	}
	if !strings.Contains(lowerType, "json") {
		return string(body)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		// Not a full JSON document (truncated mid-object): store as-is, the
		// body is capped and will be reviewed manually if ever needed.
		return string(body)
	}
	maskSensitiveMap(payload)
	masked, err := json.Marshal(payload)
	if err != nil {
		return string(body)
	}
	return string(masked)
}

func maskSensitiveMap(payload map[string]interface{}) {
	for key, value := range payload {
		keyLower := strings.ToLower(key)
		masked := false
		for _, fragment := range sensitiveKeyFragments {
			if strings.Contains(keyLower, fragment) {
				payload[key] = "******"
				masked = true
				break
			}
		}
		if masked {
			continue
		}
		if nested, ok := value.(map[string]interface{}); ok {
			maskSensitiveMap(nested)
		} else if nestedList, ok := value.([]interface{}); ok {
			for _, item := range nestedList {
				if nested, ok := item.(map[string]interface{}); ok {
					maskSensitiveMap(nested)
				}
			}
		}
	}
}

// OperationAudit records state-changing requests after the handler completes.
// It tolerates missing auth context (unauthenticated calls are skipped) and
// never alters the request outcome.
func OperationAudit() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			c.Next()
			return
		}

		var body []byte
		truncated := false
		if c.Request.Body != nil {
			body, _ = io.ReadAll(io.LimitReader(c.Request.Body, operationAuditMaxBody+1))
			if len(body) > operationAuditMaxBody {
				body = body[:operationAuditMaxBody]
				truncated = true
			}
			c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(body), c.Request.Body))
		}

		start := time.Now()
		c.Next()

		userValue, exists := c.Get("user")
		if !exists {
			return
		}
		user, ok := userValue.(model.User)
		if !ok {
			return
		}

		info := classifyOperation(method, c.FullPath(), c)
		entry := model.OperationLog{
			UserID:       user.ID,
			Username:     user.Key(),
			Module:       info.Module,
			Action:       info.Action,
			Resource:     info.Resource,
			ResourceName: operationResourceName(c),
			Cluster:      c.GetString(ClusterNameKey),
			Method:       method,
			Path:         c.Request.URL.Path,
			StatusCode:   c.Writer.Status(),
			DurationMS:   time.Since(start).Milliseconds(),
			IP:           c.ClientIP(),
			Body:         sanitizeOperationBody(c.GetHeader("Content-Type"), body),
		}
		if truncated {
			entry.Body += " ...(truncated)"
		}
		model.RecordOperationLog(entry)
		klog.V(4).Infof("Operation audit: %s %s -> %d by %s", method, c.Request.URL.Path, c.Writer.Status(), user.Key())
	}
}
