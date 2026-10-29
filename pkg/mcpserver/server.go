// Package mcpserver exposes a Streamable HTTP MCP endpoint at /api/v1/mcp so
// external MCP clients can operate clusters read-only under the caller's
// existing KubeOsprey RBAC. It is disabled unless enabled in the general settings.
package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fadewalk/kube-osprey/pkg/cluster"
	"github.com/fadewalk/kube-osprey/pkg/model"
	"github.com/fadewalk/kube-osprey/pkg/version"
	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/klog/v2"
)

// jsonRPCDisabledCode mirrors the well-known "server capability disabled"
// error so clients surface a readable message instead of a transport error.
const jsonRPCDisabledCode = -32001

type Handler struct {
	cm      *cluster.ClusterManager
	stream  *mcp.StreamableHTTPHandler
	limiter *rateLimiter
}

func NewHandler(cm *cluster.ClusterManager) *Handler {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "kube-osprey",
		Title:   "KubeOsprey Kubernetes Console",
		Version: version.Version,
	}, nil)
	registerTools(server, cm)

	return &Handler{
		cm: cm,
		stream: mcp.NewStreamableHTTPHandler(
			func(*http.Request) *mcp.Server { return server },
			nil,
		),
		limiter: newRateLimiter(),
	}
}

// Enabled reports whether MCP access is turned on in the runtime settings.
func Enabled() bool {
	setting, err := model.GetGeneralSetting()
	if err != nil {
		klog.Warningf("Failed to load general setting for MCP enablement: %v", err)
		return false
	}
	return setting.MCPEnabled
}

// GinHandler adapts the MCP streamable handler into the Gin pipeline: the
// caller is already authenticated by RequireAuth, and the user plus the
// runtime settings are carried into the MCP session via the request context.
func (h *Handler) GinHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := userFromGin(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
			c.Abort()
			return
		}
		if !Enabled() {
			writeJSONRPCDisabled(c.Writer, c.Request.Method)
			c.Abort()
			return
		}
		limit := mcpRateLimit()
		if !h.limiter.allow(user.ID, limit) {
			klog.V(2).Infof("MCP rate limit exceeded for user %s", user.Username)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "MCP rate limit exceeded, retry later",
			})
			c.Abort()
			return
		}
		ctx := context.WithValue(c.Request.Context(), userContextKey{}, user)
		c.Request = c.Request.WithContext(ctx)
		h.stream.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}

func writeJSONRPCDisabled(w http.ResponseWriter, method string) {
	if method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"MCP endpoint is disabled"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      nil,
		"error": map[string]any{
			"code":    jsonRPCDisabledCode,
			"message": "MCP endpoint is disabled",
		},
	}); err != nil {
		klog.Warningf("Failed to encode MCP disabled response: %v", err)
	}
}
