package model

import (
	"time"
)

// Change plan lifecycle. Transitions:
//
//	CREATED -> PENDING -> (APPROVED -> APPLYING -> APPLIED | FAILED) | REJECTED | EXPIRED
const (
	ChangePlanCreated  = "CREATED"
	ChangePlanPending  = "PENDING"
	ChangePlanApproved = "APPROVED"
	ChangePlanApplying = "APPLYING"
	ChangePlanApplied  = "APPLIED"
	ChangePlanFailed   = "FAILED"
	ChangePlanRejected = "REJECTED"
	ChangePlanExpired  = "EXPIRED"
)

// Change plan step lifecycle:
//
//	PENDING -> DRY_RUN_OK | DRY_RUN_FAILED | DRY_RUN_UNSUPPORTED -> APPLIED | FAILED | SKIPPED
const (
	ChangeStepPending           = "PENDING"
	ChangeStepDryRunOK          = "DRY_RUN_OK"
	ChangeStepDryRunFailed      = "DRY_RUN_FAILED"
	ChangeStepDryRunUnsupported = "DRY_RUN_UNSUPPORTED"
	ChangeStepApplied           = "APPLIED"
	ChangeStepFailed            = "FAILED"
	ChangeStepSkipped           = "SKIPPED"
)

const (
	DangerLevelLow    = "LOW"
	DangerLevelMedium = "MEDIUM"
	DangerLevelHigh   = "HIGH"
)

// DefaultChangePlanTTL is how long a finalized plan stays approvable.
const DefaultChangePlanTTL = 30 * time.Minute

// ChangePlan is an AI-authored batch of mutations that only executes after a
// human approves it in the UI. Writes happen exclusively through
// apply_change_plan; no other path may act on the steps.
type ChangePlan struct {
	Model
	UserID         uint   `json:"userId" gorm:"not null;index"`
	Username       string `json:"username" gorm:"type:varchar(255);not null;default:''"`
	AgentSessionID string `json:"agentSessionId" gorm:"type:varchar(128);not null;default:'';index"`
	ClientName     string `json:"clientName" gorm:"type:varchar(128);not null;default:''"`
	Description    string `json:"description" gorm:"type:text"`
	Status         string `json:"status" gorm:"type:varchar(20);not null;default:'CREATED';index"`
	DangerLevel    string `json:"dangerLevel" gorm:"type:varchar(10);not null;default:'LOW'"`
	// ApprovalToken is presented by the creator when calling apply; it is
	// consumed (cleared) inside the apply transaction so it cannot be replayed.
	ApprovalToken  SecretString     `json:"-" gorm:"type:varchar(64)"`
	ApproverUserID uint             `json:"approverUserId" gorm:"not null;default:0"`
	ApproverName   string           `json:"approverName" gorm:"type:varchar(255);not null;default:''"`
	ApprovedAt     *time.Time       `json:"approvedAt"`
	ExpiresAt      *time.Time       `json:"expiresAt"`
	Steps          []ChangePlanStep `json:"steps" gorm:"foreignKey:PlanID"`
}

func (ChangePlan) TableName() string { return "change_plans" }

type ChangePlanStep struct {
	Model
	PlanID      uint   `json:"planId" gorm:"not null;index"`
	Seq         int    `json:"seq" gorm:"not null"`
	ToolName    string `json:"toolName" gorm:"type:varchar(64);not null"`
	Description string `json:"description" gorm:"type:text"`
	Cluster     string `json:"cluster" gorm:"type:varchar(100);not null;default:''"`
	Namespace   string `json:"namespace" gorm:"type:varchar(100);not null;default:''"`
	Resource    string `json:"resource" gorm:"type:varchar(255);not null;default:''"`
	DangerLevel string `json:"dangerLevel" gorm:"type:varchar(10);not null;default:'MEDIUM'"`
	Status      string `json:"status" gorm:"type:varchar(24);not null;default:'PENDING'"`
	// ArgsJSON holds the original mutation tool arguments (yaml/patch etc.).
	ArgsJSON        string     `json:"argsJson" gorm:"type:text"`
	BeforeJSON      string     `json:"beforeJson" gorm:"type:text"`
	AfterJSON       string     `json:"afterJson" gorm:"type:text"`
	DryRunSupported bool       `json:"dryRunSupported" gorm:"not null;default:true"`
	DryRunStatus    string     `json:"dryRunStatus" gorm:"type:varchar(24);not null;default:'PENDING'"`
	DryRunResponse  string     `json:"dryRunResponse" gorm:"type:text"`
	ApplyStatus     string     `json:"applyStatus" gorm:"type:varchar(24);not null;default:'PENDING'"`
	ApplyResponse   string     `json:"applyResponse" gorm:"type:text"`
	ErrorMessage    string     `json:"errorMessage" gorm:"type:text"`
	AppliedAt       *time.Time `json:"appliedAt"`
}

func (ChangePlanStep) TableName() string { return "change_plan_steps" }

// MaxDangerLevel returns the stronger of two danger levels.
func MaxDangerLevel(a, b string) string {
	rank := map[string]int{DangerLevelLow: 0, DangerLevelMedium: 1, DangerLevelHigh: 2}
	if rank[b] > rank[a] {
		return b
	}
	return a
}
