package model

import (
	"time"

	"k8s.io/klog/v2"
)

// MCPAuditLog records every MCP tool call for traceability: who called what,
// on which cluster, with what outcome. Args are stored as a truncated summary.
type MCPAuditLog struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"createdAt" gorm:"index"`

	UserID   uint   `json:"userId" gorm:"not null;index"`
	Username string `json:"username" gorm:"type:varchar(255)"`

	ToolName    string `json:"toolName" gorm:"type:varchar(100);not null;index"`
	ClusterName string `json:"clusterName" gorm:"type:varchar(100);not null;index"`
	ArgsSummary string `json:"argsSummary" gorm:"type:text"`

	DurationMS   int64  `json:"durationMs"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage" gorm:"type:text"`
}

func (MCPAuditLog) TableName() string {
	return "mcp_audit_logs"
}

// RecordMCPAudit persists an MCP call record; failures to log are surfaced as
// warnings only and never break the tool call itself.
func RecordMCPAudit(entry MCPAuditLog) {
	if DB == nil {
		return
	}
	if err := DB.Create(&entry).Error; err != nil {
		klog.Warningf("Failed to record MCP audit log: %v", err)
	}
}
