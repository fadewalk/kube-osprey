package model

import (
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// OperationLog records every state-changing HTTP request made through the
// console API: who did what, where, and with what outcome. Bodies are
// truncated and sanitized by the audit middleware before reaching here.
type OperationLog struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"createdAt" gorm:"index"`

	UserID   uint   `json:"userId" gorm:"index"`
	Username string `json:"username" gorm:"type:varchar(255);index"`

	Module       string `json:"module" gorm:"type:varchar(50);index"`
	Action       string `json:"action" gorm:"type:varchar(50)"`
	Resource     string `json:"resource" gorm:"type:varchar(100)"`
	ResourceName string `json:"resourceName" gorm:"type:varchar(255)"`
	Cluster      string `json:"cluster" gorm:"type:varchar(100);index"`

	Method     string `json:"method" gorm:"type:varchar(10)"`
	Path       string `json:"path" gorm:"type:varchar(300)"`
	StatusCode int    `json:"statusCode"`
	DurationMS int64  `json:"durationMs"`
	IP         string `json:"ip" gorm:"type:varchar(64)"`
	RequestID  string `json:"requestId" gorm:"type:varchar(64);index"`
	Body       string `json:"body" gorm:"type:text"`
}

const (
	operationLogQueueSize = 512
	// Fallback when the configured audit retention is unreadable.
	defaultOperationLogRetention = 30 * 24 * time.Hour
	maxOperationLogRetention     = 3650 * 24 * time.Hour
)

var (
	operationLogOnce    sync.Once
	operationLogQueue   chan OperationLog
	operationLogStopped chan struct{}
)

// RecordOperationLog queues an entry for asynchronous persistence; it never
// blocks the request path. When the queue is full the entry is dropped and
// counted — auditing must not take the API down.
func RecordOperationLog(entry OperationLog) {
	if DB == nil {
		return
	}
	operationLogOnce.Do(startOperationLogWorker)
	select {
	case operationLogQueue <- entry:
	default:
		klog.Warning("Operation audit queue full, dropping entry")
	}
}

func startOperationLogWorker() {
	operationLogQueue = make(chan OperationLog, operationLogQueueSize)
	operationLogStopped = make(chan struct{})
	go func() {
		for entry := range operationLogQueue {
			if err := DB.Create(&entry).Error; err != nil {
				klog.Warningf("Failed to record operation audit log: %v", err)
			}
		}
		close(operationLogStopped)
	}()
	go func() {
		// Purge stale entries at startup and then on a fixed interval so the
		// table cannot grow unbounded.
		purgeOperationLogs()
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				purgeOperationLogs()
			case <-operationLogStopped:
				return
			}
		}
	}()
}

func purgeOperationLogs() {
	retention := auditLogRetention()
	result := DB.Where("created_at < ?", time.Now().Add(-retention)).Delete(&OperationLog{})
	if result.Error != nil {
		klog.Warningf("Failed to purge operation audit logs: %v", result.Error)
	} else if result.RowsAffected > 0 {
		klog.Infof("Purged %d operation audit logs older than %s", result.RowsAffected, retention)
	}
}

// auditLogRetention returns the configured audit retention. It fails open
// to the default on read errors and clamps the configured value.
func auditLogRetention() time.Duration {
	retention := defaultOperationLogRetention
	if setting, err := GetGeneralSetting(); err == nil && setting.AuditRetentionDays > 0 {
		retention = time.Duration(setting.AuditRetentionDays) * 24 * time.Hour
		if retention > maxOperationLogRetention {
			retention = maxOperationLogRetention
		}
	}
	return retention
}

// ListOperationLogs returns a page of operation logs with optional filters.
func ListOperationLogs(page, size int, username, module, cluster, method string) ([]OperationLog, int64, error) {
	query := DB.Model(&OperationLog{})
	if username != "" {
		query = query.Where("username = ?", username)
	}
	if module != "" {
		query = query.Where("module = ?", module)
	}
	if cluster != "" {
		query = query.Where("cluster = ?", cluster)
	}
	if method != "" {
		query = query.Where("method = ?", method)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var logs []OperationLog
	err := query.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&logs).Error
	return logs, total, err
}
