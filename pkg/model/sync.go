package model

import "time"

// SyncObject is one mirrored Kubernetes object of the sync engine. The sync
// engine writes here out of band; reads serve lists, search and topology for
// whitelisted resource types when the engine is enabled.
type SyncObject struct {
	ID uint `json:"id" gorm:"primarykey"`

	ClusterName string `json:"clusterName" gorm:"type:varchar(100);not null;uniqueIndex:uniq_sync_objects,priority:1"`
	UID         string `json:"uid" gorm:"type:varchar(64);not null;uniqueIndex:uniq_sync_objects,priority:2"`

	Group    string `json:"group" gorm:"type:varchar(253);not null;default:''"`
	Version  string `json:"version" gorm:"type:varchar(20);not null;default:''"`
	Kind     string `json:"kind" gorm:"type:varchar(64);not null;index"`
	Resource string `json:"resource" gorm:"type:varchar(64);not null;index:idx_sync_objects_lookup,priority:1"`

	Namespace string `json:"namespace" gorm:"type:varchar(100);not null;default:'';index:idx_sync_objects_lookup,priority:2"`
	Name      string `json:"name" gorm:"type:varchar(253);not null;index:idx_sync_objects_lookup,priority:3"`
	// SearchName is lower(Name), the LIKE target of global search.
	SearchName string `json:"searchName" gorm:"type:varchar(253);not null;index"`

	LabelsJSON string `json:"labelsJson" gorm:"type:text"`
	// OwnerUIDs is a JSON array of the ownerReferences UIDs, the basis of
	// topology queries.
	OwnerUIDs         string    `json:"ownerUids" gorm:"type:text"`
	ResourceVersion   string    `json:"resourceVersion" gorm:"type:varchar(40);not null;default:''"`
	CreationTimestamp time.Time `json:"creationTimestamp"`

	ObjectJSON string `json:"objectJson" gorm:"type:text"`

	// DeletedAt marks objects whose DELETED watch event arrived; a retention
	// job physically removes them later.
	DeletedAt *time.Time `json:"deletedAt" gorm:"index"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

func (SyncObject) TableName() string { return "sync_objects" }

// Sync task lifecycle statuses.
const (
	SyncTaskRunning = "RUNNING"
	SyncTaskError   = "ERROR"
	SyncTaskStopped = "STOPPED"
)

// SyncTask tracks the sync state per (cluster, resource) reflector.
type SyncTask struct {
	Model
	ClusterName string `json:"clusterName" gorm:"type:varchar(100);not null;uniqueIndex:uniq_sync_tasks,priority:1"`
	Resource    string `json:"resource" gorm:"type:varchar(64);not null;uniqueIndex:uniq_sync_tasks,priority:2"`

	Status      string     `json:"status" gorm:"type:varchar(20);not null;default:'RUNNING'"`
	ObjectCount int64      `json:"objectCount"`
	LastError   string     `json:"lastError" gorm:"type:text"`
	LastSyncAt  *time.Time `json:"lastSyncAt"`
}

func (SyncTask) TableName() string { return "sync_tasks" }
