package model

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/zxh326/kite/pkg/common"
	"gorm.io/gorm"
	"k8s.io/klog/v2"
)

const DefaultGeneralAIModel = "gpt-4o-mini"
const DefaultGeneralAnthropicModel = "claude-sonnet-4-5"
const DefaultGeneralKubectlImage = "zzde/kubectl:latest"
const DefaultGeneralNodeTerminalImage = "busybox:latest"
const DefaultGeneralClusterAgentImage = "ghcr.io/kite-org/kite:latest"

const GeneralAIProviderOpenAI = "openai"
const GeneralAIProviderAnthropic = "anthropic"
const DefaultGeneralAIProvider = GeneralAIProviderOpenAI

const (
	KubectlPermissionOperator = "operator"
	KubectlPermissionAdmin    = "admin"
	KubectlPermissionCustom   = "custom"
	// Admin mode must be chosen explicitly; unknown/empty values fall back
	// to the least privileged built-in profile.
	DefaultKubectlPermissionMode = KubectlPermissionOperator
)

func DefaultGeneralNodeTerminalImageValue() string {
	image := strings.TrimSpace(common.NodeTerminalImage)
	if image == "" {
		return DefaultGeneralNodeTerminalImage
	}
	return image
}

func DefaultGeneralKubectlImageValue() string {
	image := strings.TrimSpace(common.KubectlTerminalImage)
	if image == "" {
		return DefaultGeneralKubectlImage
	}
	return image
}

func DefaultGeneralClusterAgentImageValue() string {
	image := strings.TrimSpace(common.ClusterAgentImage)
	if image == "" {
		return DefaultGeneralClusterAgentImage
	}
	return image
}

type GeneralSetting struct {
	Model
	AIAgentEnabled        bool         `json:"aiAgentEnabled" gorm:"column:ai_agent_enabled;type:boolean;not null;default:false"`
	AIProvider            string       `json:"aiProvider" gorm:"column:ai_provider;type:varchar(50);not null;default:'openai'"`
	AIModel               string       `json:"aiModel" gorm:"column:ai_model;type:varchar(255);not null;default:'gpt-4o-mini'"`
	AIAPIKey              SecretString `json:"aiApiKey" gorm:"column:ai_api_key;type:text"`
	AIBaseURL             string       `json:"aiBaseUrl" gorm:"column:ai_base_url;type:varchar(500)"`
	AIMaxTokens           int          `json:"aiMaxTokens" gorm:"column:ai_max_tokens;type:integer;default:16384"`
	KubectlEnabled        bool         `json:"kubectlEnabled" gorm:"column:kubectl_enabled;type:boolean;not null;default:false"`
	KubectlImage          string       `json:"kubectlImage" gorm:"column:kubectl_image;type:varchar(255);not null;default:'zzde/kubectl:latest'"`
	KubectlPermissionMode string       `json:"kubectlPermissionMode" gorm:"column:kubectl_permission_mode;type:varchar(20);not null;default:'operator'"`
	KubectlClusterRole    string       `json:"kubectlClusterRole" gorm:"column:kubectl_cluster_role;type:varchar(253)"`
	NodeTerminalEnabled   bool         `json:"nodeTerminalEnabled" gorm:"column:node_terminal_enabled;type:boolean;not null;default:false"`
	NodeTerminalImage     string       `json:"nodeTerminalImage" gorm:"column:node_terminal_image;type:varchar(255);not null;default:'busybox:latest'"`
	ClusterAgentImage     string       `json:"clusterAgentImage" gorm:"column:cluster_agent_image;type:varchar(255);not null;default:'ghcr.io/kite-org/kite:latest'"`
	PluginCatalogURL      string       `json:"pluginCatalogUrl" gorm:"column:plugin_catalog_url;type:varchar(2048)"`
	MCPEnabled            bool         `json:"mcpEnabled" gorm:"column:mcp_enabled;type:boolean;not null;default:false"`
	MCPRateLimitPerMinute int          `json:"mcpRateLimitPerMinute" gorm:"column:mcp_rate_limit_per_minute;type:integer;not null;default:60"`
	SyncEngineEnabled     bool         `json:"syncEngineEnabled" gorm:"column:sync_engine_enabled;type:boolean;not null;default:false"`
	// SyncEngineResources is a comma-separated whitelist of resource plurals;
	// empty means the built-in default whitelist.
	SyncEngineResources string `json:"syncEngineResources" gorm:"column:sync_engine_resources;type:text"`
	// AIChangePlanMode routes chat-assistant mutations through the change
	// plan approval flow instead of the in-chat single-step confirmation.
	AIChangePlanMode        bool         `json:"aiChangePlanMode" gorm:"column:ai_change_plan_mode;type:boolean;not null;default:false"`
	EnableAnalytics         bool         `json:"enableAnalytics" gorm:"column:enable_analytics;type:boolean;not null;default:false"`
	AnalyticsInstallationID string       `json:"-" gorm:"type:varchar(32)"`
	AnalyticsLastAttemptAt  *time.Time   `json:"-"`
	EnableVersionCheck      bool         `json:"enableVersionCheck" gorm:"column:enable_version_check;type:boolean;not null;default:true"`
	PasswordLoginDisabled   bool         `json:"passwordLoginDisabled" gorm:"column:password_login_disabled;type:boolean;not null;default:false"`
	EnableMFA               bool         `json:"enableMFA" gorm:"column:enable_mfa;type:boolean;not null;default:true"`
	EnablePasskeyLogin      bool         `json:"enablePasskeyLogin" gorm:"column:enable_passkey_login;type:boolean;not null;default:true"`
	LoginPrompt             string       `json:"loginPrompt" gorm:"column:login_prompt;type:text"`
	JWTSecret               SecretString `json:"-" gorm:"column:jwt_secret;type:text"`
	GlobalSidebarPreference string       `json:"-" gorm:"column:global_sidebar_preference;type:text"`
}

func NormalizeGeneralAIProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case GeneralAIProviderAnthropic:
		return GeneralAIProviderAnthropic
	default:
		return GeneralAIProviderOpenAI
	}
}

func NormalizeKubectlPermissionMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case KubectlPermissionAdmin:
		return KubectlPermissionAdmin
	case KubectlPermissionCustom:
		return KubectlPermissionCustom
	default:
		return DefaultKubectlPermissionMode
	}
}

func IsKubectlPermissionModeSupported(mode string) bool {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	return normalized == KubectlPermissionAdmin || normalized == KubectlPermissionOperator || normalized == KubectlPermissionCustom
}

func IsGeneralAIProviderSupported(provider string) bool {
	normalized := strings.ToLower(strings.TrimSpace(provider))
	return normalized == GeneralAIProviderOpenAI || normalized == GeneralAIProviderAnthropic
}

func DefaultGeneralAIModelByProvider(provider string) string {
	switch NormalizeGeneralAIProvider(provider) {
	case GeneralAIProviderAnthropic:
		return DefaultGeneralAnthropicModel
	default:
		return DefaultGeneralAIModel
	}
}

func GetGeneralSetting() (*GeneralSetting, error) {
	var setting GeneralSetting
	err := DB.First(&setting, 1).Error
	if err == nil {
		updates := map[string]interface{}{}
		if setting.AIProvider == "" {
			setting.AIProvider = DefaultGeneralAIProvider
			updates["ai_provider"] = DefaultGeneralAIProvider
		} else {
			normalizedProvider := NormalizeGeneralAIProvider(setting.AIProvider)
			if setting.AIProvider != normalizedProvider {
				setting.AIProvider = normalizedProvider
				updates["ai_provider"] = normalizedProvider
			}
		}
		if setting.AIModel == "" {
			setting.AIModel = DefaultGeneralAIModelByProvider(setting.AIProvider)
			updates["ai_model"] = setting.AIModel
		}
		if setting.KubectlImage == "" {
			setting.KubectlImage = DefaultGeneralKubectlImageValue()
			updates["kubectl_image"] = setting.KubectlImage
		}
		normalizedMode := NormalizeKubectlPermissionMode(setting.KubectlPermissionMode)
		if setting.KubectlPermissionMode != normalizedMode {
			setting.KubectlPermissionMode = normalizedMode
			updates["kubectl_permission_mode"] = normalizedMode
		}
		if setting.NodeTerminalImage == "" {
			setting.NodeTerminalImage = DefaultGeneralNodeTerminalImageValue()
			updates["node_terminal_image"] = setting.NodeTerminalImage
		}
		if setting.ClusterAgentImage == "" {
			setting.ClusterAgentImage = DefaultGeneralClusterAgentImageValue()
			updates["cluster_agent_image"] = setting.ClusterAgentImage
		}
		if setting.MCPRateLimitPerMinute <= 0 {
			setting.MCPRateLimitPerMinute = common.MCPRateLimitPerMinute
			updates["mcp_rate_limit_per_minute"] = setting.MCPRateLimitPerMinute
		}
		if err := ensureJWTSecret(&setting, updates); err != nil {
			return nil, err
		}
		if len(updates) > 0 {
			if err := DB.Model(&setting).Updates(updates).Error; err != nil {
				return nil, err
			}
		}
		applyRuntimeGeneralSetting(&setting)
		return &setting, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	setting = GeneralSetting{
		Model:                 Model{ID: 1},
		AIAgentEnabled:        false,
		AIProvider:            DefaultGeneralAIProvider,
		AIModel:               DefaultGeneralAIModel,
		AIMaxTokens:           16384,
		KubectlEnabled:        false,
		KubectlImage:          DefaultGeneralKubectlImageValue(),
		KubectlPermissionMode: DefaultKubectlPermissionMode,
		NodeTerminalImage:     DefaultGeneralNodeTerminalImageValue(),
		ClusterAgentImage:     DefaultGeneralClusterAgentImageValue(),
		MCPEnabled:            common.MCPEnabled,
		MCPRateLimitPerMinute: common.MCPRateLimitPerMinute,
		EnableAnalytics:       common.EnableAnalytics,
		EnableVersionCheck:    common.EnableVersionCheck,
		EnableMFA:             true,
		EnablePasskeyLogin:    true,
	}
	if err := ensureJWTSecret(&setting, nil); err != nil {
		return nil, err
	}
	if err := DB.Create(&setting).Error; err != nil {
		return nil, err
	}
	applyRuntimeGeneralSetting(&setting)
	return &setting, nil
}

func UpdateGeneralSetting(updates map[string]interface{}) (*GeneralSetting, error) {
	setting, err := GetGeneralSetting()
	if err != nil {
		return nil, err
	}
	if err := DB.Model(setting).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := DB.First(setting, setting.ID).Error; err != nil {
		return nil, err
	}
	applyRuntimeGeneralSetting(setting)
	return setting, nil
}

func applyRuntimeGeneralSetting(setting *GeneralSetting) {
	if setting == nil {
		return
	}
	common.EnableAnalytics = setting.EnableAnalytics
	common.EnableVersionCheck = setting.EnableVersionCheck
}

func ensureJWTSecret(setting *GeneralSetting, updates map[string]interface{}) error {
	storedSecret := strings.TrimSpace(string(setting.JWTSecret))
	configuredSecret := strings.TrimSpace(common.JwtSecret)

	switch {
	case configuredSecret != "" && configuredSecret != common.DefaultJWTSecret:
		if storedSecret != configuredSecret {
			setting.JWTSecret = SecretString(configuredSecret)
			if updates != nil {
				updates["jwt_secret"] = setting.JWTSecret
			}
		}
		common.JwtSecret = configuredSecret
		return nil
	case storedSecret != "" && storedSecret != common.DefaultJWTSecret:
		common.JwtSecret = storedSecret
		return nil
	default:
		generatedSecret, err := generateJWTSecret()
		if err != nil {
			return err
		}
		setting.JWTSecret = SecretString(generatedSecret)
		common.JwtSecret = generatedSecret
		if updates != nil {
			updates["jwt_secret"] = setting.JWTSecret
		}
		klog.Warningf("JWT secret is using the insecure default value, generated a random secret and stored it in general setting")
		return nil
	}
}

func generateJWTSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
