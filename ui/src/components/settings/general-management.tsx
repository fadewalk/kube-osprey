import { useEffect, useState } from 'react'
import {
  IconApi,
  IconDatabase,
  IconKey,
  IconLink,
  IconMessage,
  IconPalette,
  IconPuzzle,
  IconRobot,
  IconSettings,
  IconTerminal2,
} from '@tabler/icons-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'

import {
  GeneralSettingUpdateRequest,
  KubectlPermissionMode,
  updateGeneralSetting,
  useGeneralSetting,
} from '@/lib/api'
import { translateError } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

const DEFAULT_MODEL = 'gpt-4o-mini'
const DEFAULT_ANTHROPIC_MODEL = 'claude-sonnet-5-5'
const DEFAULT_KUBECTL_IMAGE = 'zzde/kubectl:latest'
const DEFAULT_NODE_TERMINAL_IMAGE = 'busybox:latest'
const DEFAULT_CLUSTER_AGENT_IMAGE = 'ghcr.io/kite-org/kite:latest'

interface GeneralSettingsFormData {
  aiAgentEnabled: boolean
  aiProvider: 'openai' | 'anthropic'
  aiModel: string
  aiApiKey: string
  aiApiKeyConfigured: boolean
  aiBaseUrl: string
  aiMaxTokens: number
  aiChangePlanMode: boolean
  kubectlEnabled: boolean
  kubectlImage: string
  kubectlPermissionMode: KubectlPermissionMode
  kubectlClusterRole: string
  nodeTerminalEnabled: boolean
  nodeTerminalImage: string
  clusterAgentImage: string
  pluginCatalogUrl: string
  mcpEnabled: boolean
  mcpRateLimitPerMinute: number
  syncEngineEnabled: boolean
  syncEngineResources: string
  enableVersionCheck: boolean
  loginPrompt: string
  siteTitle: string
  siteLogo: string
  passwordExpireDays: number
  passwordMinLength: number
  passwordRequireUppercase: boolean
  passwordRequireLowercase: boolean
  passwordRequireDigit: boolean
  passwordRequireSpecial: boolean
  passwordHistoryCount: number
}

export function GeneralManagement() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { data, isLoading } = useGeneralSetting()
  const isAIManaged = data?.aiManaged ?? false
  const [formData, setFormData] = useState<GeneralSettingsFormData>({
    aiAgentEnabled: false,
    aiProvider: 'openai',
    aiModel: DEFAULT_MODEL,
    aiApiKey: '',
    aiApiKeyConfigured: false,
    aiBaseUrl: '',
    aiMaxTokens: 16384,
    aiChangePlanMode: false,
    kubectlEnabled: false,
    kubectlImage: DEFAULT_KUBECTL_IMAGE,
    kubectlPermissionMode: 'operator',
    kubectlClusterRole: '',
    nodeTerminalEnabled: false,
    nodeTerminalImage: DEFAULT_NODE_TERMINAL_IMAGE,
    clusterAgentImage: DEFAULT_CLUSTER_AGENT_IMAGE,
    pluginCatalogUrl: '',
    mcpEnabled: false,
    mcpRateLimitPerMinute: 60,
    syncEngineEnabled: false,
    syncEngineResources: '',
    enableVersionCheck: true,
    loginPrompt: '',
    siteTitle: '',
    siteLogo: '',
    passwordExpireDays: 0,
    passwordMinLength: 8,
    passwordRequireUppercase: false,
    passwordRequireLowercase: false,
    passwordRequireDigit: false,
    passwordRequireSpecial: false,
    passwordHistoryCount: 0,
  })

  useEffect(() => {
    if (!data) return
    setFormData({
      aiAgentEnabled: data.aiAgentEnabled,
      aiProvider: data.aiProvider || 'openai',
      aiModel: data.aiModel || DEFAULT_MODEL,
      aiApiKey: '',
      aiApiKeyConfigured: data.aiApiKeyConfigured ?? false,
      aiBaseUrl: data.aiBaseUrl || '',
      aiMaxTokens: data.aiMaxTokens || 16384,
      aiChangePlanMode: data.aiChangePlanMode ?? false,
      kubectlEnabled: data.kubectlEnabled ?? false,
      kubectlImage: data.kubectlImage || DEFAULT_KUBECTL_IMAGE,
      kubectlPermissionMode: data.kubectlPermissionMode || 'operator',
      kubectlClusterRole: data.kubectlClusterRole || '',
      nodeTerminalEnabled: data.nodeTerminalEnabled ?? false,
      nodeTerminalImage: data.nodeTerminalImage || DEFAULT_NODE_TERMINAL_IMAGE,
      clusterAgentImage: data.clusterAgentImage || DEFAULT_CLUSTER_AGENT_IMAGE,
      pluginCatalogUrl: data.pluginCatalogUrl,
      mcpEnabled: data.mcpEnabled ?? false,
      mcpRateLimitPerMinute: data.mcpRateLimitPerMinute || 60,
      syncEngineEnabled: data.syncEngineEnabled ?? false,
      syncEngineResources: data.syncEngineResources || '',
      enableVersionCheck: data.enableVersionCheck ?? true,
      loginPrompt: data.loginPrompt || '',
      siteTitle: data.siteTitle || '',
      siteLogo: data.siteLogo || '',
      passwordExpireDays: data.passwordExpireDays ?? 0,
      passwordMinLength: data.passwordMinLength ?? 8,
      passwordRequireUppercase: data.passwordRequireUppercase ?? false,
      passwordRequireLowercase: data.passwordRequireLowercase ?? false,
      passwordRequireDigit: data.passwordRequireDigit ?? false,
      passwordRequireSpecial: data.passwordRequireSpecial ?? false,
      passwordHistoryCount: data.passwordHistoryCount ?? 0,
    })
  }, [data])

  const mutation = useMutation({
    mutationFn: (payload: GeneralSettingUpdateRequest) =>
      updateGeneralSetting(payload),
    onSuccess: (_, payload) => {
      queryClient.invalidateQueries({
        predicate: (query) =>
          query.queryKey[0] === 'general-setting' ||
          query.queryKey[0] === 'ai-status' ||
          query.queryKey[0] === 'bootstrap',
      })
      if (payload.pluginCatalogUrl !== data?.pluginCatalogUrl) {
        queryClient.invalidateQueries({ queryKey: ['plugins', 'catalog'] })
        queryClient.invalidateQueries({ queryKey: ['plugins', 'readme'] })
      }
      toast.success(
        t('generalManagement.messages.updated', 'General settings updated')
      )
    },
    onError: (error) => {
      toast.error(translateError(error, t))
    },
  })

  const handleSave = () => {
    const defaultModel =
      formData.aiProvider === 'anthropic'
        ? DEFAULT_ANTHROPIC_MODEL
        : DEFAULT_MODEL

    if (!isAIManaged && formData.aiAgentEnabled && !formData.aiModel.trim()) {
      toast.error(
        t('generalManagement.errors.modelRequired', 'Model is required')
      )
      return
    }
    if (
      !isAIManaged &&
      formData.aiAgentEnabled &&
      !formData.aiApiKey.trim() &&
      !formData.aiApiKeyConfigured
    ) {
      toast.error(
        t(
          'generalManagement.errors.apiKeyRequired',
          'API Key is required when AI Agent is enabled'
        )
      )
      return
    }
    if (formData.kubectlEnabled && !formData.kubectlImage.trim()) {
      toast.error(
        t(
          'generalManagement.errors.kubectlImageRequired',
          'Kubectl image is required when kubectl is enabled'
        )
      )
      return
    }
    if (
      formData.kubectlEnabled &&
      formData.kubectlPermissionMode === 'custom' &&
      !formData.kubectlClusterRole.trim()
    ) {
      toast.error(
        t(
          'generalManagement.errors.kubectlClusterRoleRequired',
          'A ClusterRole is required in custom permission mode'
        )
      )
      return
    }
    if (!formData.nodeTerminalImage.trim()) {
      toast.error(
        t(
          'generalManagement.errors.nodeTerminalImageRequired',
          'Node terminal image is required'
        )
      )
      return
    }

    const payload: GeneralSettingUpdateRequest = {
      kubectlEnabled: formData.kubectlEnabled,
      kubectlImage: formData.kubectlImage.trim() || DEFAULT_KUBECTL_IMAGE,
      kubectlPermissionMode: formData.kubectlPermissionMode,
      kubectlClusterRole: formData.kubectlClusterRole.trim(),
      nodeTerminalEnabled: formData.nodeTerminalEnabled,
      nodeTerminalImage:
        formData.nodeTerminalImage.trim() || DEFAULT_NODE_TERMINAL_IMAGE,
      clusterAgentImage:
        formData.clusterAgentImage.trim() || DEFAULT_CLUSTER_AGENT_IMAGE,
      pluginCatalogUrl: formData.pluginCatalogUrl.trim(),
      mcpEnabled: formData.mcpEnabled,
      mcpRateLimitPerMinute:
        formData.mcpRateLimitPerMinute > 0
          ? formData.mcpRateLimitPerMinute
          : 60,
      syncEngineEnabled: formData.syncEngineEnabled,
      syncEngineResources: formData.syncEngineResources.trim(),
      enableVersionCheck: formData.enableVersionCheck,
      loginPrompt: formData.loginPrompt.trim(),
      siteTitle: formData.siteTitle.trim(),
      siteLogo: formData.siteLogo.trim(),
      passwordExpireDays: Math.max(0, formData.passwordExpireDays || 0),
      passwordMinLength: Math.max(4, formData.passwordMinLength || 8),
      passwordRequireUppercase: formData.passwordRequireUppercase,
      passwordRequireLowercase: formData.passwordRequireLowercase,
      passwordRequireDigit: formData.passwordRequireDigit,
      passwordRequireSpecial: formData.passwordRequireSpecial,
      passwordHistoryCount: Math.max(0, formData.passwordHistoryCount || 0),
    }
    if (!isAIManaged) {
      payload.aiAgentEnabled = formData.aiAgentEnabled
      payload.aiProvider = formData.aiProvider
      payload.aiModel = formData.aiModel.trim() || defaultModel
      payload.aiBaseUrl = formData.aiBaseUrl.trim()
      payload.aiMaxTokens = formData.aiMaxTokens || 16384
      payload.aiChangePlanMode = formData.aiChangePlanMode
      if (formData.aiApiKey.trim()) {
        payload.aiApiKey = formData.aiApiKey.trim()
      }
    }

    mutation.mutate(payload)
  }

  if (isLoading && !data) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="text-muted-foreground">
          {t('common.messages.loading', 'Loading...')}
        </div>
      </div>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <IconSettings className="h-5 w-5" />
          {t('generalManagement.title', 'General')}
        </CardTitle>
      </CardHeader>

      <CardContent className="space-y-4">
        <div className="rounded-lg border">
          <div className="flex items-center justify-between p-3">
            <div className="space-y-1">
              <Label className="flex items-center gap-2 text-sm font-medium">
                <IconRobot className="h-4 w-4" />
                {t('generalManagement.aiAgent.title', 'AI Agent')}
              </Label>
              <p className="text-xs text-muted-foreground">
                {t(
                  'generalManagement.aiAgent.description',
                  'Enable AI assistant and configure model endpoint.'
                )}
              </p>
              {isAIManaged && (
                <p className="text-xs text-muted-foreground">
                  {t('generalManagement.aiAgent.managedByConfig')}
                </p>
              )}
            </div>
            <Switch
              disabled={isAIManaged}
              checked={formData.aiAgentEnabled}
              onCheckedChange={(checked) =>
                setFormData((prev) => ({ ...prev, aiAgentEnabled: checked }))
              }
            />
          </div>

          {formData.aiAgentEnabled && (
            <div className="space-y-4 border-t p-3">
              <div className="space-y-2">
                <Label htmlFor="general-ai-provider">
                  {t('common.fields.provider', 'Provider')}
                </Label>
                <Select
                  disabled={isAIManaged}
                  value={formData.aiProvider}
                  onValueChange={(value: 'openai' | 'anthropic') =>
                    setFormData((prev) => ({
                      ...prev,
                      aiProvider: value,
                      aiModel:
                        value === 'anthropic'
                          ? prev.aiModel || DEFAULT_ANTHROPIC_MODEL
                          : prev.aiModel || DEFAULT_MODEL,
                    }))
                  }
                >
                  <SelectTrigger id="general-ai-provider">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="openai">OpenAI Compatible</SelectItem>
                    <SelectItem value="anthropic">
                      Anthropic Compatible
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="general-ai-model">
                  {t('generalManagement.aiAgent.form.model', 'Model')}
                </Label>
                <Input
                  id="general-ai-model"
                  disabled={isAIManaged}
                  value={formData.aiModel}
                  onChange={(e) =>
                    setFormData((prev) => ({
                      ...prev,
                      aiModel: e.target.value,
                    }))
                  }
                  placeholder={
                    formData.aiProvider === 'anthropic'
                      ? DEFAULT_ANTHROPIC_MODEL
                      : DEFAULT_MODEL
                  }
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="general-ai-api-key">
                  {t('generalManagement.aiAgent.form.apiKey', 'API Key')}
                </Label>
                <Input
                  id="general-ai-api-key"
                  disabled={isAIManaged}
                  type="password"
                  value={formData.aiApiKey}
                  onChange={(e) =>
                    setFormData((prev) => ({
                      ...prev,
                      aiApiKey: e.target.value,
                    }))
                  }
                  placeholder={
                    formData.aiApiKeyConfigured
                      ? t(
                          'generalManagement.aiAgent.form.apiKeyPlaceholder',
                          'Leave empty to keep current API Key'
                        )
                      : 'sk-...'
                  }
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="general-ai-base-url">
                  {t('generalManagement.aiAgent.form.baseUrl', 'Base URL')}
                </Label>
                <Input
                  id="general-ai-base-url"
                  disabled={isAIManaged}
                  value={formData.aiBaseUrl}
                  onChange={(e) =>
                    setFormData((prev) => ({
                      ...prev,
                      aiBaseUrl: e.target.value,
                    }))
                  }
                  placeholder={
                    formData.aiProvider === 'anthropic'
                      ? 'https://api.anthropic.com'
                      : 'https://api.openai.com/v1'
                  }
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="general-ai-max-tokens">
                  {t('generalManagement.aiAgent.form.maxTokens', 'Max Tokens')}
                </Label>
                <Input
                  id="general-ai-max-tokens"
                  disabled={isAIManaged}
                  type="number"
                  min="1"
                  max="128000"
                  value={formData.aiMaxTokens}
                  onChange={(e) =>
                    setFormData((prev) => ({
                      ...prev,
                      aiMaxTokens: parseInt(e.target.value) || 16384,
                    }))
                  }
                  placeholder="16384"
                />
              </div>

              <div className="flex items-start justify-between gap-3 rounded-md border p-3">
                <div className="space-y-1">
                  <Label
                    htmlFor="general-ai-change-plan-mode"
                    className="text-sm font-medium"
                  >
                    {t(
                      'generalManagement.aiAgent.form.changePlanMode',
                      'Route write operations through approval'
                    )}
                  </Label>
                  <p className="text-xs text-muted-foreground">
                    {t(
                      'generalManagement.aiAgent.form.changePlanModeHint',
                      'Assistant mutations become change plans: dry-run first, then a human approves in Change Plans before anything is applied.'
                    )}
                  </p>
                </div>
                <Switch
                  id="general-ai-change-plan-mode"
                  disabled={isAIManaged}
                  checked={formData.aiChangePlanMode}
                  onCheckedChange={(checked) =>
                    setFormData((prev) => ({
                      ...prev,
                      aiChangePlanMode: checked,
                    }))
                  }
                />
              </div>
            </div>
          )}
        </div>

        <div className="rounded-lg border">
          <div className="flex items-center justify-between p-3">
            <div className="space-y-1">
              <Label className="flex items-center gap-2 text-sm font-medium">
                <IconTerminal2 className="h-4 w-4" />
                {t('generalManagement.kubectl.title', 'Kubectl')}
              </Label>
              <p className="text-xs text-muted-foreground">
                {t(
                  'generalManagement.kubectl.description',
                  'Enable kubectl terminal and configure runtime image.'
                )}
              </p>
            </div>
            <Switch
              checked={formData.kubectlEnabled}
              onCheckedChange={(checked) =>
                setFormData((prev) => ({ ...prev, kubectlEnabled: checked }))
              }
            />
          </div>

          {formData.kubectlEnabled && (
            <div className="space-y-2 border-t p-3">
              <Label htmlFor="general-kubectl-image">
                {t('generalManagement.kubectl.form.image', 'Image')}
              </Label>
              <Input
                id="general-kubectl-image"
                value={formData.kubectlImage}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    kubectlImage: e.target.value,
                  }))
                }
                placeholder={DEFAULT_KUBECTL_IMAGE}
              />

              <div className="space-y-2">
                <Label htmlFor="general-kubectl-permission-mode">
                  {t(
                    'generalManagement.kubectl.form.permissionMode',
                    'Permission mode'
                  )}
                </Label>
                <Select
                  value={formData.kubectlPermissionMode}
                  onValueChange={(value: KubectlPermissionMode) =>
                    setFormData((prev) => ({
                      ...prev,
                      kubectlPermissionMode: value,
                    }))
                  }
                >
                  <SelectTrigger id="general-kubectl-permission-mode">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="operator">
                      {t(
                        'generalManagement.kubectl.form.permissionOperator',
                        'Operator (limited permissions)'
                      )}
                    </SelectItem>
                    <SelectItem value="admin">
                      {t(
                        'generalManagement.kubectl.form.permissionAdmin',
                        'Admin (cluster-admin)'
                      )}
                    </SelectItem>
                    <SelectItem value="custom">
                      {t(
                        'generalManagement.kubectl.form.permissionCustom',
                        'Custom ClusterRole'
                      )}
                    </SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">
                  {t(
                    'generalManagement.kubectl.form.permissionModeDescription',
                    'Each session runs with a temporary ServiceAccount bound to the selected role. Admin grants full cluster-admin access and should only be used for break-glass.'
                  )}
                </p>
              </div>

              {formData.kubectlPermissionMode === 'custom' && (
                <div className="space-y-2">
                  <Label htmlFor="general-kubectl-cluster-role">
                    {t(
                      'generalManagement.kubectl.form.clusterRole',
                      'ClusterRole'
                    )}
                  </Label>
                  <Input
                    id="general-kubectl-cluster-role"
                    value={formData.kubectlClusterRole}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        kubectlClusterRole: e.target.value,
                      }))
                    }
                    placeholder="platform-operator"
                  />
                </div>
              )}
            </div>
          )}
        </div>

        <div className="rounded-lg border">
          <div className="flex items-center justify-between p-3">
            <div className="space-y-1">
              <Label className="flex items-center gap-2 text-sm font-medium">
                <IconTerminal2 className="h-4 w-4" />
                {t('generalManagement.nodeTerminal.title', 'Node Terminal')}
              </Label>
              <p className="text-xs text-muted-foreground">
                {t(
                  'generalManagement.nodeTerminal.description',
                  'Node terminal grants a privileged host shell and is disabled by default.'
                )}
              </p>
            </div>
            <Switch
              checked={formData.nodeTerminalEnabled}
              onCheckedChange={(checked) =>
                setFormData((prev) => ({
                  ...prev,
                  nodeTerminalEnabled: checked,
                }))
              }
            />
          </div>

          {formData.nodeTerminalEnabled && (
            <div className="space-y-2 border-t p-3">
              <Label htmlFor="general-node-terminal-image">
                {t('generalManagement.nodeTerminal.form.image', 'Image')}
              </Label>
              <Input
                id="general-node-terminal-image"
                value={formData.nodeTerminalImage}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    nodeTerminalImage: e.target.value,
                  }))
                }
                placeholder={DEFAULT_NODE_TERMINAL_IMAGE}
              />
              <p className="text-xs text-muted-foreground">
                {t(
                  'generalManagement.nodeTerminal.form.enabledWarning',
                  'Sessions run as a privileged root shell on the node. Only admins can open them; keep this off unless required.'
                )}
              </p>
            </div>
          )}
        </div>

        <div className="rounded-lg border">
          <div className="flex items-center justify-between p-3">
            <div className="space-y-1">
              <Label className="flex items-center gap-2 text-sm font-medium">
                <IconApi className="h-4 w-4" />
                {t('generalManagement.mcp.title', 'MCP Server')}
              </Label>
              <p className="text-xs text-muted-foreground">
                {t(
                  'generalManagement.mcp.description',
                  'Let external MCP clients operate clusters read-only with API Keys under each user’s RBAC.'
                )}
              </p>
            </div>
            <Switch
              checked={formData.mcpEnabled}
              onCheckedChange={(checked) =>
                setFormData((prev) => ({ ...prev, mcpEnabled: checked }))
              }
            />
          </div>

          {formData.mcpEnabled && (
            <div className="space-y-4 border-t p-3">
              <div className="space-y-2">
                <Label htmlFor="general-mcp-rate-limit">
                  {t(
                    'generalManagement.mcp.form.rateLimit',
                    'Rate limit (calls per user per minute)'
                  )}
                </Label>
                <Input
                  id="general-mcp-rate-limit"
                  type="number"
                  min="1"
                  max="10000"
                  value={formData.mcpRateLimitPerMinute}
                  onChange={(e) =>
                    setFormData((prev) => ({
                      ...prev,
                      mcpRateLimitPerMinute: parseInt(e.target.value) || 60,
                    }))
                  }
                  placeholder="60"
                />
              </div>

              <div className="space-y-2 rounded-md bg-muted/50 p-3 text-xs">
                <div className="space-y-1">
                  <span className="font-medium">
                    {t('generalManagement.mcp.form.endpoint', 'Endpoint URL')}
                  </span>
                  <code className="block break-all font-mono text-muted-foreground">
                    {`${window.location.origin}${import.meta.env.BASE_URL}api/v1/mcp`}
                  </code>
                </div>
                <div className="space-y-1">
                  <span className="font-medium">
                    {t(
                      'generalManagement.mcp.form.auth',
                      'Authentication header'
                    )}
                  </span>
                  <code className="block break-all font-mono text-muted-foreground">
                    Authorization: kite&lt;your-api-key&gt;
                  </code>
                </div>
                <p className="text-muted-foreground">
                  {t(
                    'generalManagement.mcp.form.hint',
                    'Every tool call is audited and evaluated against the API key owner’s RBAC permissions.'
                  )}
                </p>
                <Link
                  to="/settings?tab=apikeys"
                  className="inline-block font-medium text-primary underline-offset-4 hover:underline"
                >
                  {t(
                    'generalManagement.mcp.form.createApiKey',
                    'Create an API Key'
                  )}
                </Link>
              </div>
            </div>
          )}
        </div>

        <div className="rounded-lg border">
          <div className="flex items-center justify-between p-3">
            <div className="space-y-1">
              <Label className="flex items-center gap-2 text-sm font-medium">
                <IconDatabase className="h-4 w-4" />
                {t(
                  'generalManagement.syncEngine.title',
                  'Resource Sync Engine'
                )}
              </Label>
              <p className="text-xs text-muted-foreground">
                {t(
                  'generalManagement.syncEngine.description',
                  'Mirror whitelisted resource types into the database so lists, search and topology stay available without touching the API server.'
                )}
              </p>
            </div>
            <Switch
              checked={formData.syncEngineEnabled}
              onCheckedChange={(checked) =>
                setFormData((prev) => ({
                  ...prev,
                  syncEngineEnabled: checked,
                }))
              }
            />
          </div>

          {formData.syncEngineEnabled && (
            <div className="space-y-2 border-t p-3">
              <Label htmlFor="general-sync-resources">
                {t(
                  'generalManagement.syncEngine.form.resources',
                  'Resource whitelist (comma-separated)'
                )}
              </Label>
              <Input
                id="general-sync-resources"
                value={formData.syncEngineResources}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    syncEngineResources: e.target.value,
                  }))
                }
                placeholder="pods, deployments, statefulsets, daemonsets, replicasets, jobs, services, ingresses, configmaps, nodes, namespaces, events"
              />
              <p className="text-xs text-muted-foreground">
                {t(
                  'generalManagement.syncEngine.form.hint',
                  'Leave empty for the default whitelist. Secrets are excluded by default and are stored as-is if added explicitly.'
                )}
              </p>
            </div>
          )}
        </div>

        <div className="rounded-lg border p-3">
          <div className="space-y-1">
            <Label className="flex items-center gap-2 text-sm font-medium">
              <IconLink className="h-4 w-4" />
              {t('generalManagement.clusterAgent.title', 'Cluster Agent Image')}
            </Label>
            <p className="text-xs text-muted-foreground">
              {t(
                'generalManagement.clusterAgent.description',
                'Container image used when generating the Cluster Agent manifest for Cluster Agent clusters.'
              )}
            </p>
          </div>

          <div className="mt-3 space-y-2">
            <Label htmlFor="general-cluster-agent-image">
              {t('generalManagement.clusterAgent.form.image', 'Image')}
            </Label>
            <Input
              id="general-cluster-agent-image"
              value={formData.clusterAgentImage}
              onChange={(e) =>
                setFormData((prev) => ({
                  ...prev,
                  clusterAgentImage: e.target.value,
                }))
              }
              placeholder={DEFAULT_CLUSTER_AGENT_IMAGE}
            />
          </div>
        </div>

        <div className="rounded-lg border p-3">
          <div className="space-y-1">
            <Label className="flex items-center gap-2 text-sm font-medium">
              <IconPuzzle className="h-4 w-4" />
              {t('generalManagement.pluginCatalog.title')}
            </Label>
            <p
              id="general-plugin-catalog-description"
              className="text-xs text-muted-foreground"
            >
              {t('generalManagement.pluginCatalog.description')}
            </p>
          </div>

          <div className="mt-3 space-y-2">
            <Label htmlFor="general-plugin-catalog-url">
              {t('generalManagement.pluginCatalog.url')}
            </Label>
            <Input
              id="general-plugin-catalog-url"
              type="url"
              value={formData.pluginCatalogUrl}
              onChange={(e) =>
                setFormData((prev) => ({
                  ...prev,
                  pluginCatalogUrl: e.target.value,
                }))
              }
              placeholder="https://plugins.kitehq.dev/catalog.json"
              aria-describedby="general-plugin-catalog-description"
            />
          </div>
        </div>

        <div className="rounded-lg border">
          <div className="p-3">
            <Label className="text-sm font-medium">
              {t('generalManagement.runtime.title', 'Runtime')}
            </Label>
            <p className="mt-1 text-xs text-muted-foreground">
              {t(
                'generalManagement.runtime.description',
                'Configure version checking behavior.'
              )}
            </p>
          </div>

          <div className="flex items-center justify-between border-t p-3">
            <Label htmlFor="general-enable-version-check" className="text-sm">
              {t(
                'generalManagement.runtime.form.enableVersionCheck',
                'Enable version check'
              )}
            </Label>
            <Switch
              id="general-enable-version-check"
              checked={formData.enableVersionCheck}
              onCheckedChange={(checked) =>
                setFormData((prev) => ({
                  ...prev,
                  enableVersionCheck: checked,
                }))
              }
            />
          </div>
        </div>

        <div className="rounded-lg border p-3">
          <div className="space-y-1">
            <Label className="flex items-center gap-2 text-sm font-medium">
              <IconPalette className="h-4 w-4" />
              {t('generalManagement.branding.title', 'Branding')}
            </Label>
            <p className="text-xs text-muted-foreground">
              {t(
                'generalManagement.branding.description',
                'Override the site name and logo shown on the login page and sidebar. Leave empty to keep the KubeOsprey defaults.'
              )}
            </p>
          </div>

          <div className="mt-3 grid gap-3 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="general-site-title">
                {t('generalManagement.branding.siteTitle', 'Site Title')}
              </Label>
              <Input
                id="general-site-title"
                value={formData.siteTitle}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    siteTitle: e.target.value,
                  }))
                }
                placeholder={t(
                  'generalManagement.branding.siteTitlePlaceholder',
                  'Leave empty to use "KubeOsprey"'
                )}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="general-site-logo">
                {t('generalManagement.branding.siteLogo', 'Logo URL')}
              </Label>
              <Input
                id="general-site-logo"
                value={formData.siteLogo}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    siteLogo: e.target.value,
                  }))
                }
                placeholder={t(
                  'generalManagement.branding.siteLogoPlaceholder',
                  'Leave empty to use the KubeOsprey logo'
                )}
              />
            </div>
          </div>
        </div>

        <div className="rounded-lg border p-3">
          <div className="space-y-1">
            <Label className="flex items-center gap-2 text-sm font-medium">
              <IconMessage className="h-4 w-4" />
              {t('generalManagement.loginPrompt.title', 'Login Prompt')}
            </Label>
            <p className="text-xs text-muted-foreground">
              {t(
                'generalManagement.loginPrompt.description',
                'Show a custom message on the login page.'
              )}
            </p>
          </div>

          <div className="mt-3 space-y-2">
            <Label htmlFor="general-login-prompt">
              {t('generalManagement.loginPrompt.form.message', 'Message')}
            </Label>
            <Textarea
              id="general-login-prompt"
              value={formData.loginPrompt}
              onChange={(e) =>
                setFormData((prev) => ({
                  ...prev,
                  loginPrompt: e.target.value,
                }))
              }
              placeholder={t(
                'generalManagement.loginPrompt.form.placeholder',
                'Leave empty to hide the login prompt'
              )}
            />
          </div>
        </div>

        <div className="rounded-lg border p-3">
          <div className="space-y-1">
            <Label className="flex items-center gap-2 text-sm font-medium">
              <IconKey className="h-4 w-4" />
              {t('generalManagement.passwordPolicy.title', 'Password Policy')}
            </Label>
            <p className="text-xs text-muted-foreground">
              {t(
                'generalManagement.passwordPolicy.description',
                'Applies to every password set through the console. Expired passwords cannot log in until an administrator resets them.'
              )}
            </p>
          </div>

          <div className="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2">
            <div className="space-y-1">
              <Label htmlFor="general-password-min-length">
                {t(
                  'generalManagement.passwordPolicy.minLength',
                  'Minimum length'
                )}
              </Label>
              <Input
                id="general-password-min-length"
                type="number"
                min={4}
                max={128}
                value={formData.passwordMinLength}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    passwordMinLength: Number(e.target.value),
                  }))
                }
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="general-password-expire-days">
                {t(
                  'generalManagement.passwordPolicy.expireDays',
                  'Expires after (days)'
                )}
              </Label>
              <Input
                id="general-password-expire-days"
                type="number"
                min={0}
                value={formData.passwordExpireDays}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    passwordExpireDays: Number(e.target.value),
                  }))
                }
              />
              <p className="text-xs text-muted-foreground">
                {t(
                  'generalManagement.passwordPolicy.expireDaysHint',
                  '0 disables expiry'
                )}
              </p>
            </div>
            <div className="space-y-1">
              <Label htmlFor="general-password-history">
                {t(
                  'generalManagement.passwordPolicy.history',
                  'Reject last N reused passwords'
                )}
              </Label>
              <Input
                id="general-password-history"
                type="number"
                min={0}
                max={24}
                value={formData.passwordHistoryCount}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    passwordHistoryCount: Number(e.target.value),
                  }))
                }
              />
              <p className="text-xs text-muted-foreground">
                {t(
                  'generalManagement.passwordPolicy.historyHint',
                  '0 disables the reuse check'
                )}
              </p>
            </div>
            <div className="grid grid-cols-2 gap-2">
              {(
                [
                  ['uppercase', 'passwordRequireUppercase'],
                  ['lowercase', 'passwordRequireLowercase'],
                  ['digit', 'passwordRequireDigit'],
                  ['special', 'passwordRequireSpecial'],
                ] as const
              ).map(([key, field]) => (
                <label key={field} className="flex items-center gap-2 text-sm">
                  <Checkbox
                    checked={formData[field]}
                    onCheckedChange={(checked) =>
                      setFormData((prev) => ({
                        ...prev,
                        [field]: checked === true,
                      }))
                    }
                  />
                  {t(`generalManagement.passwordPolicy.${key}`)}
                </label>
              ))}
            </div>
          </div>
        </div>

        <div className="flex justify-end">
          <Button onClick={handleSave} disabled={mutation.isPending}>
            {t('common.actions.save', 'Save')}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
