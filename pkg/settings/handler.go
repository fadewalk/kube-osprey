package settings

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/fadewalk/kube-osprey/pkg/common"
	"github.com/fadewalk/kube-osprey/pkg/model"
	"github.com/fadewalk/kube-osprey/pkg/syncengine"
	"github.com/gin-gonic/gin"
)

func HandleGetGeneralSetting(c *gin.Context) {
	setting, err := model.GetGeneralSetting()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to load general setting: %v", err)})
		return
	}
	c.JSON(http.StatusOK, generalSettingResponse(setting))
}

func generalSettingResponse(setting *model.GeneralSetting) gin.H {
	hasAIAPIKey := strings.TrimSpace(string(setting.AIAPIKey)) != ""
	return gin.H{
		"aiManaged":                common.IsSectionManaged("ai"),
		"aiAgentEnabled":           setting.AIAgentEnabled,
		"aiProvider":               setting.AIProvider,
		"aiModel":                  setting.AIModel,
		"aiApiKey":                 "",
		"aiApiKeyConfigured":       hasAIAPIKey,
		"aiBaseUrl":                setting.AIBaseURL,
		"aiMaxTokens":              setting.AIMaxTokens,
		"aiChangePlanMode":         setting.AIChangePlanMode,
		"webhookChangePlanMode":    setting.WebhookChangePlanMode,
		"kubectlEnabled":           setting.KubectlEnabled,
		"kubectlImage":             setting.KubectlImage,
		"kubectlPermissionMode":    setting.KubectlPermissionMode,
		"kubectlClusterRole":       setting.KubectlClusterRole,
		"nodeTerminalEnabled":      setting.NodeTerminalEnabled,
		"nodeTerminalImage":        setting.NodeTerminalImage,
		"clusterAgentImage":        setting.ClusterAgentImage,
		"pluginCatalogUrl":         setting.PluginCatalogURL,
		"mcpEnabled":               setting.MCPEnabled,
		"mcpRateLimitPerMinute":    setting.MCPRateLimitPerMinute,
		"syncEngineEnabled":        setting.SyncEngineEnabled,
		"syncEngineResources":      setting.SyncEngineResources,
		"enableVersionCheck":       setting.EnableVersionCheck,
		"passwordLoginDisabled":    setting.PasswordLoginDisabled,
		"enableMFA":                setting.EnableMFA,
		"enablePasskeyLogin":       setting.EnablePasskeyLogin,
		"loginPrompt":              setting.LoginPrompt,
		"siteTitle":                setting.SiteTitle,
		"siteLogo":                 setting.SiteLogo,
		"passwordExpireDays":       setting.PasswordExpireDays,
		"passwordMinLength":        setting.PasswordMinLength,
		"passwordRequireUppercase": setting.PasswordRequireUppercase,
		"passwordRequireLowercase": setting.PasswordRequireLowercase,
		"passwordRequireDigit":     setting.PasswordRequireDigit,
		"passwordRequireSpecial":   setting.PasswordRequireSpecial,
		"passwordHistoryCount":     setting.PasswordHistoryCount,
		"auditRetentionDays":       setting.AuditRetentionDays,
		"auditSkippedModules":      setting.AuditSkippedModules,
	}
}

type UpdateGeneralSettingRequest struct {
	AIAgentEnabled        *bool   `json:"aiAgentEnabled"`
	AIProvider            *string `json:"aiProvider"`
	AIModel               *string `json:"aiModel"`
	AIAPIKey              *string `json:"aiApiKey"`
	AIBaseURL             *string `json:"aiBaseUrl"`
	AIMaxTokens           *int    `json:"aiMaxTokens"`
	KubectlEnabled        *bool   `json:"kubectlEnabled"`
	KubectlImage          *string `json:"kubectlImage"`
	KubectlPermissionMode *string `json:"kubectlPermissionMode"`
	KubectlClusterRole    *string `json:"kubectlClusterRole"`
	NodeTerminalEnabled   *bool   `json:"nodeTerminalEnabled"`
	NodeTerminalImage     *string `json:"nodeTerminalImage"`
	ClusterAgentImage     *string `json:"clusterAgentImage"`
	PluginCatalogURL      *string `json:"pluginCatalogUrl"`
	MCPEnabled            *bool   `json:"mcpEnabled"`
	MCPRateLimitPerMinute *int    `json:"mcpRateLimitPerMinute"`
	AIChangePlanMode      *bool   `json:"aiChangePlanMode"`
	WebhookChangePlanMode *bool   `json:"webhookChangePlanMode"`
	SyncEngineEnabled     *bool   `json:"syncEngineEnabled"`
	SyncEngineResources   *string `json:"syncEngineResources"`
	EnableVersionCheck    *bool   `json:"enableVersionCheck"`
	PasswordLoginDisabled *bool   `json:"passwordLoginDisabled"`
	EnableMFA             *bool   `json:"enableMFA"`
	EnablePasskeyLogin    *bool   `json:"enablePasskeyLogin"`
	LoginPrompt           *string `json:"loginPrompt"`
	SiteTitle             *string `json:"siteTitle"`
	SiteLogo              *string `json:"siteLogo"`

	PasswordExpireDays       *int  `json:"passwordExpireDays"`
	PasswordMinLength        *int  `json:"passwordMinLength"`
	PasswordRequireUppercase *bool `json:"passwordRequireUppercase"`
	PasswordRequireLowercase *bool `json:"passwordRequireLowercase"`
	PasswordRequireDigit     *bool `json:"passwordRequireDigit"`
	PasswordRequireSpecial   *bool `json:"passwordRequireSpecial"`
	PasswordHistoryCount     *int  `json:"passwordHistoryCount"`

	AuditRetentionDays  *int    `json:"auditRetentionDays"`
	AuditSkippedModules *string `json:"auditSkippedModules"`
}

func HandleUpdateGeneralSetting(c *gin.Context) { //nolint:gocyclo
	var req UpdateGeneralSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}
	if common.IsSectionManaged("ai") && (req.AIAgentEnabled != nil || req.AIProvider != nil ||
		req.AIModel != nil || req.AIAPIKey != nil || req.AIBaseURL != nil || req.AIMaxTokens != nil) {
		c.JSON(http.StatusForbidden, gin.H{"error": common.ManagedSectionError})
		return
	}
	currentSetting, err := model.GetGeneralSetting()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to load general setting: %v", err)})
		return
	}

	aiProvider := model.NormalizeGeneralAIProvider(currentSetting.AIProvider)
	if req.AIProvider != nil {
		incomingProvider := strings.ToLower(strings.TrimSpace(*req.AIProvider))
		if incomingProvider != "" {
			if !model.IsGeneralAIProviderSupported(incomingProvider) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported aiProvider"})
				return
			}
			aiProvider = model.NormalizeGeneralAIProvider(incomingProvider)
		}
	}

	aiModel := strings.TrimSpace(currentSetting.AIModel)
	if req.AIModel != nil {
		aiModel = strings.TrimSpace(*req.AIModel)
	}
	if aiModel == "" {
		aiModel = model.DefaultGeneralAIModelByProvider(aiProvider)
	}
	aiAPIKey := strings.TrimSpace(string(currentSetting.AIAPIKey))
	shouldUpdateAIAPIKey := false
	if req.AIAPIKey != nil {
		incomingKey := strings.TrimSpace(*req.AIAPIKey)
		if incomingKey != "" {
			aiAPIKey = incomingKey
			shouldUpdateAIAPIKey = true
		}
	}
	aiAgentEnabled := currentSetting.AIAgentEnabled
	if req.AIAgentEnabled != nil {
		aiAgentEnabled = *req.AIAgentEnabled
	}
	if aiAgentEnabled && aiAPIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "aiApiKey is required when aiAgentEnabled is true"})
		return
	}

	kubectlEnabled := currentSetting.KubectlEnabled
	if req.KubectlEnabled != nil {
		kubectlEnabled = *req.KubectlEnabled
	}
	kubectlImage := strings.TrimSpace(currentSetting.KubectlImage)
	if req.KubectlImage != nil {
		kubectlImage = strings.TrimSpace(*req.KubectlImage)
	}
	if kubectlEnabled && req.KubectlImage != nil && strings.TrimSpace(*req.KubectlImage) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kubectlImage is required when kubectlEnabled is true"})
		return
	}
	if kubectlImage == "" {
		kubectlImage = model.DefaultGeneralKubectlImageValue()
	}
	kubectlPermissionMode := model.NormalizeKubectlPermissionMode(currentSetting.KubectlPermissionMode)
	if req.KubectlPermissionMode != nil {
		incomingMode := strings.ToLower(strings.TrimSpace(*req.KubectlPermissionMode))
		if incomingMode != "" {
			if !model.IsKubectlPermissionModeSupported(incomingMode) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported kubectlPermissionMode"})
				return
			}
			kubectlPermissionMode = incomingMode
		}
	}
	kubectlClusterRole := strings.TrimSpace(currentSetting.KubectlClusterRole)
	if req.KubectlClusterRole != nil {
		kubectlClusterRole = strings.TrimSpace(*req.KubectlClusterRole)
	}
	if kubectlPermissionMode == model.KubectlPermissionCustom && kubectlClusterRole == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kubectlClusterRole is required when kubectlPermissionMode is custom"})
		return
	}
	nodeTerminalImage := strings.TrimSpace(currentSetting.NodeTerminalImage)
	if req.NodeTerminalImage != nil {
		nodeTerminalImage = strings.TrimSpace(*req.NodeTerminalImage)
	}
	if nodeTerminalImage == "" {
		nodeTerminalImage = model.DefaultGeneralNodeTerminalImageValue()
	}
	clusterAgentImage := strings.TrimSpace(currentSetting.ClusterAgentImage)
	if req.ClusterAgentImage != nil {
		clusterAgentImage = strings.TrimSpace(*req.ClusterAgentImage)
	}
	if clusterAgentImage == "" {
		clusterAgentImage = model.DefaultGeneralClusterAgentImageValue()
	}

	aiMaxTokens := currentSetting.AIMaxTokens
	if req.AIMaxTokens != nil {
		aiMaxTokens = *req.AIMaxTokens
	}
	if aiMaxTokens <= 0 {
		aiMaxTokens = 16384
	}

	updates := map[string]interface{}{}
	if req.AIAgentEnabled != nil {
		updates["ai_agent_enabled"] = aiAgentEnabled
	}
	if req.AIProvider != nil {
		updates["ai_provider"] = aiProvider
	}
	if req.AIModel != nil {
		updates["ai_model"] = aiModel
	}
	if req.AIBaseURL != nil {
		updates["ai_base_url"] = strings.TrimSpace(*req.AIBaseURL)
	}
	if req.AIMaxTokens != nil {
		updates["ai_max_tokens"] = aiMaxTokens
	}
	if req.KubectlEnabled != nil {
		updates["kubectl_enabled"] = kubectlEnabled
	}
	if req.KubectlImage != nil {
		updates["kubectl_image"] = kubectlImage
	}
	if req.KubectlPermissionMode != nil {
		updates["kubectl_permission_mode"] = kubectlPermissionMode
	}
	if req.KubectlClusterRole != nil {
		updates["kubectl_cluster_role"] = kubectlClusterRole
	}
	if req.NodeTerminalEnabled != nil {
		updates["node_terminal_enabled"] = *req.NodeTerminalEnabled
	}
	if req.NodeTerminalImage != nil {
		updates["node_terminal_image"] = nodeTerminalImage
	}
	if req.ClusterAgentImage != nil {
		updates["cluster_agent_image"] = clusterAgentImage
	}
	if req.PluginCatalogURL != nil {
		address := strings.TrimSpace(*req.PluginCatalogURL)
		if address != "" {
			u, err := url.Parse(address)
			if err != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.Fragment != "" || len(address) > 2048 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "pluginCatalogUrl must be an HTTP or HTTPS URL without credentials or fragments, at most 2048 characters"})
				return
			}
		}
		updates["plugin_catalog_url"] = address
	}
	if req.MCPEnabled != nil {
		updates["mcp_enabled"] = *req.MCPEnabled
	}
	if req.AIChangePlanMode != nil {
		updates["ai_change_plan_mode"] = *req.AIChangePlanMode
	}
	if req.WebhookChangePlanMode != nil {
		updates["webhook_change_plan_mode"] = *req.WebhookChangePlanMode
	}
	if req.SyncEngineEnabled != nil {
		updates["sync_engine_enabled"] = *req.SyncEngineEnabled
	}
	if req.SyncEngineResources != nil {
		updates["sync_engine_resources"] = strings.TrimSpace(*req.SyncEngineResources)
	}
	if req.MCPRateLimitPerMinute != nil {
		limit := *req.MCPRateLimitPerMinute
		if limit <= 0 || limit > 10000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mcpRateLimitPerMinute must be between 1 and 10000"})
			return
		}
		updates["mcp_rate_limit_per_minute"] = limit
	}
	if req.EnableVersionCheck != nil {
		updates["enable_version_check"] = *req.EnableVersionCheck
	}
	if req.LoginPrompt != nil {
		updates["login_prompt"] = strings.TrimSpace(*req.LoginPrompt)
	}
	if req.SiteTitle != nil {
		updates["site_title"] = strings.TrimSpace(*req.SiteTitle)
	}
	if req.SiteLogo != nil {
		updates["site_logo"] = strings.TrimSpace(*req.SiteLogo)
	}
	if req.PasswordLoginDisabled != nil {
		updates["password_login_disabled"] = *req.PasswordLoginDisabled
	}
	if req.EnableMFA != nil {
		updates["enable_mfa"] = *req.EnableMFA
	}
	if req.EnablePasskeyLogin != nil {
		updates["enable_passkey_login"] = *req.EnablePasskeyLogin
	}
	if req.PasswordExpireDays != nil {
		updates["password_expire_days"] = max(0, *req.PasswordExpireDays)
	}
	if req.PasswordMinLength != nil {
		updates["password_min_length"] = max(4, *req.PasswordMinLength)
	}
	if req.PasswordRequireUppercase != nil {
		updates["password_require_uppercase"] = *req.PasswordRequireUppercase
	}
	if req.PasswordRequireLowercase != nil {
		updates["password_require_lowercase"] = *req.PasswordRequireLowercase
	}
	if req.PasswordRequireDigit != nil {
		updates["password_require_digit"] = *req.PasswordRequireDigit
	}
	if req.PasswordRequireSpecial != nil {
		updates["password_require_special"] = *req.PasswordRequireSpecial
	}
	if req.PasswordHistoryCount != nil {
		updates["password_history_count"] = max(0, *req.PasswordHistoryCount)
	}
	if req.AuditRetentionDays != nil {
		days := *req.AuditRetentionDays
		if days < 1 || days > 3650 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "auditRetentionDays must be between 1 and 3650"})
			return
		}
		updates["audit_retention_days"] = days
	}
	if req.AuditSkippedModules != nil {
		updates["audit_skipped_modules"] = normalizeAuditSkippedModules(*req.AuditSkippedModules)
	}
	if shouldUpdateAIAPIKey {
		updates["ai_api_key"] = model.SecretString(aiAPIKey)
	}

	updated, err := model.UpdateGeneralSetting(updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update general setting: %v", err)})
		return
	}
	syncengine.ReloadFromSetting()

	c.JSON(http.StatusOK, generalSettingResponse(updated))
}

// normalizeAuditSkippedModules trims, lowercases and dedupes the module CSV
// so the audit middleware can match against a clean set.
func normalizeAuditSkippedModules(raw string) string {
	fields := strings.Split(raw, ",")
	seen := map[string]struct{}{}
	var kept []string
	for _, f := range fields {
		name := strings.ToLower(strings.TrimSpace(f))
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		kept = append(kept, name)
	}
	return strings.Join(kept, ",")
}
