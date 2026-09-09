package model

import (
	"strings"
	"time"

	"sigs.k8s.io/yaml"
)

func RedactResourceYAML(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	var object interface{}
	if err := yaml.Unmarshal([]byte(value), &object); err != nil {
		return "[redacted: invalid resource content]"
	}
	if objectMap, ok := object.(map[string]interface{}); ok {
		if kind, _ := objectMap["kind"].(string); strings.EqualFold(kind, "secret") {
			if _, exists := objectMap["data"]; exists {
				objectMap["data"] = "[REDACTED]"
			}
			if _, exists := objectMap["stringData"]; exists {
				objectMap["stringData"] = "[REDACTED]"
			}
		}
	}
	redactSensitiveValue(object)
	redacted, err := yaml.Marshal(object)
	if err != nil {
		return "[redacted: unable to serialize resource content]"
	}
	return string(redacted)
}

func redactSensitiveValue(value interface{}) {
	switch current := value.(type) {
	case map[string]interface{}:
		for key, item := range current {
			if isSensitiveField(key) {
				current[key] = "[REDACTED]"
				continue
			}
			redactSensitiveValue(item)
		}
	case []interface{}:
		for _, item := range current {
			redactSensitiveValue(item)
		}
	}
}

func isSensitiveField(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
	for _, marker := range []string{"password", "token", "secret", "privatekey", "clientsecret", "kubeconfig", "apikey", "accesskey", "refreshkey"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

type ResourceHistory struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	CreatedAt   time.Time `json:"createdAt" gorm:"index:idx_resource_histories_lookup_with_time,priority:5,sort:desc"`
	UpdatedAt   time.Time `json:"updatedAt"`
	ClusterName string    `json:"clusterName" gorm:"type:varchar(100);not null;index:idx_resource_histories_lookup_with_time,priority:1"`

	ResourceType string `json:"resourceType" gorm:"type:varchar(255);not null;index:idx_resource_histories_lookup_with_time,priority:2"`
	ResourceName string `json:"resourceName" gorm:"type:varchar(255);not null;index:idx_resource_histories_lookup_with_time,priority:3"`
	Namespace    string `json:"namespace" gorm:"type:varchar(100);index:idx_resource_histories_lookup_with_time,priority:4"`

	OperationType   string `json:"operationType" gorm:"type:varchar(50);not null;index"`
	OperationSource string `json:"operationSource" gorm:"type:varchar(20);not null;default:'manual';index"`

	ResourceYAML string `json:"resourceYaml" gorm:"type:text"`
	PreviousYAML string `json:"previousYaml" gorm:"type:text"`

	Success      bool   `json:"success" gorm:"type:boolean"`
	ErrorMessage string `json:"errorMessage" gorm:"type:text"`

	OperatorID uint  `json:"operatorId" gorm:"not null;index"`
	Operator   *User `json:"operator" gorm:"foreignKey:OperatorID;constraint:OnDelete:CASCADE"`
}

func (ResourceHistory) TableName() string {
	return "resource_histories"
}
