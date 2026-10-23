import { useState } from 'react'
import {
  IconBell,
  IconCheck,
  IconPencil,
  IconPlus,
  IconSend,
  IconTrash,
  IconX,
} from '@tabler/icons-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  createNotificationChannel,
  deleteNotificationChannel,
  testNotificationChannel,
  updateNotificationChannel,
  useNotificationChannels,
  type NotificationChannel,
  type NotificationChannelRequest,
} from '@/lib/api/admin'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
import { DeleteConfirmationDialog } from '@/components/delete-confirmation-dialog'

const CHANNEL_TYPES = [
  'dingtalk',
  'wecom',
  'feishu',
  'slack',
  'webhook',
] as const
const EVENTS = [
  {
    key: 'plan_pending',
    labelKey: 'notifications.events.planPending',
    fallback: 'Plan awaiting approval',
  },
  {
    key: 'plan_applied',
    labelKey: 'notifications.events.planApplied',
    fallback: 'Plan applied',
  },
  {
    key: 'plan_failed',
    labelKey: 'notifications.events.planFailed',
    fallback: 'Plan failed',
  },
  {
    key: 'alert_firing',
    labelKey: 'notifications.events.alertFiring',
    fallback: 'Alert firing',
  },
  {
    key: 'event_warning',
    labelKey: 'notifications.events.eventWarning',
    fallback: 'Warning event',
  },
] as const

interface ChannelForm {
  name: string
  type: string
  endpoint: string
  secret: string
  events: string[]
  enabled: boolean
}

const emptyForm = (): ChannelForm => ({
  name: '',
  type: 'dingtalk',
  endpoint: '',
  secret: '',
  events: ['plan_pending'],
  enabled: true,
})

function channelToForm(channel: NotificationChannel): ChannelForm {
  return {
    name: channel.name,
    type: channel.type,
    endpoint: channel.endpoint,
    secret: '',
    events: channel.events ? channel.events.split(',').filter(Boolean) : [],
    enabled: channel.enabled,
  }
}

export function NotificationManagement() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { data: channels = [], isLoading } = useNotificationChannels()

  const [editorOpen, setEditorOpen] = useState(false)
  const [editing, setEditing] = useState<NotificationChannel | null>(null)
  const [form, setForm] = useState<ChannelForm>(emptyForm())
  const [deleting, setDeleting] = useState<NotificationChannel | null>(null)
  const [testingId, setTestingId] = useState<number | null>(null)

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: ['notification-channels'] })

  const saveMutation = useMutation({
    mutationFn: async () => {
      const request: NotificationChannelRequest = {
        name: form.name,
        type: form.type,
        endpoint: form.endpoint,
        events: form.events.join(','),
        enabled: form.enabled,
      }
      if (form.secret.trim()) {
        request.secret = form.secret.trim()
      }
      if (editing) {
        return updateNotificationChannel(editing.id, request)
      }
      return createNotificationChannel(request)
    },
    onSuccess: () => {
      toast.success(
        t('notifications.messages.saved', 'Notification channel saved')
      )
      invalidate()
      setEditorOpen(false)
    },
    onError: (err: Error) => toast.error(err.message),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteNotificationChannel(id),
    onSuccess: () => {
      toast.success(t('notifications.messages.deleted', 'Channel deleted'))
      invalidate()
      setDeleting(null)
    },
    onError: (err: Error) => toast.error(err.message),
  })

  const testMutation = useMutation({
    mutationFn: (id: number) => testNotificationChannel(id),
    onSuccess: () =>
      toast.success(
        t('notifications.messages.testSent', 'Test notification sent')
      ),
    onError: (err: Error) => toast.error(err.message),
  })

  const openCreate = () => {
    setEditing(null)
    setForm(emptyForm())
    setEditorOpen(true)
  }

  const openEdit = (channel: NotificationChannel) => {
    setEditing(channel)
    setForm(channelToForm(channel))
    setEditorOpen(true)
  }

  const toggleEvent = (key: string) => {
    setForm((prev) => ({
      ...prev,
      events: prev.events.includes(key)
        ? prev.events.filter((e) => e !== key)
        : [...prev.events, key],
    }))
  }

  const canSave =
    form.name.trim() !== '' &&
    form.endpoint.trim() !== '' &&
    form.events.length > 0

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span className="flex items-center gap-2">
            <IconBell className="h-5 w-5" />
            {t('notifications.title', 'Notifications')}
          </span>
          <Button size="sm" onClick={openCreate}>
            <IconPlus className="h-4 w-4" />
            {t('notifications.addChannel', 'Add Channel')}
          </Button>
        </CardTitle>
      </CardHeader>

      <CardContent>
        {isLoading ? (
          <p className="text-sm text-muted-foreground">
            {t('common.messages.loading', 'Loading...')}
          </p>
        ) : channels.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {t(
              'notifications.empty',
              'No notification channels configured yet.'
            )}
          </p>
        ) : (
          <div className="space-y-2">
            {channels.map((channel) => (
              <div
                key={channel.id}
                className="flex items-center justify-between rounded-lg border p-3"
              >
                <div className="space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{channel.name}</span>
                    <Badge variant="outline">{channel.type}</Badge>
                    {channel.enabled ? (
                      <Badge variant="default">
                        {t('notifications.enabled', 'enabled')}
                      </Badge>
                    ) : (
                      <Badge variant="secondary">
                        {t('notifications.disabled', 'disabled')}
                      </Badge>
                    )}
                    {channel.lastStatus === 'failed' && (
                      <Badge variant="destructive">
                        {t('notifications.lastFailed', 'last delivery failed')}
                      </Badge>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground break-all">
                    {channel.endpoint}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {channel.events
                      .split(',')
                      .map((e) => e.trim())
                      .filter(Boolean)
                      .map((e) =>
                        t(
                          EVENTS.find((ev) => ev.key === e)?.labelKey ??
                            'notifications.events.other',
                          e
                        )
                      )
                      .join(' · ')}
                    {channel.lastStatus === 'failed' && channel.lastError
                      ? ` — ${channel.lastError}`
                      : ''}
                  </p>
                </div>
                <div className="flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="icon"
                    disabled={
                      testMutation.isPending && testingId === channel.id
                    }
                    onClick={() => {
                      setTestingId(channel.id)
                      testMutation.mutate(channel.id)
                    }}
                    title={t('notifications.test', 'Send test notification')}
                  >
                    <IconSend className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => openEdit(channel)}
                  >
                    <IconPencil className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => setDeleting(channel)}
                  >
                    {deleting?.id === channel.id ? (
                      <IconX className="h-4 w-4" />
                    ) : (
                      <IconTrash className="h-4 w-4" />
                    )}
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>

      {/* Create / edit dialog */}
      <Dialog open={editorOpen} onOpenChange={setEditorOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {editing
                ? t('notifications.editChannel', 'Edit Channel')
                : t('notifications.addChannel', 'Add Channel')}
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="notification-name">
                {t('common.fields.name', 'Name')}
              </Label>
              <Input
                id="notification-name"
                value={form.name}
                onChange={(e) =>
                  setForm((p) => ({ ...p, name: e.target.value }))
                }
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="notification-type">
                {t('notifications.type', 'Type')}
              </Label>
              <Select
                value={form.type}
                onValueChange={(v) => setForm((p) => ({ ...p, type: v }))}
              >
                <SelectTrigger id="notification-type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {CHANNEL_TYPES.map((type) => (
                    <SelectItem key={type} value={type}>
                      {t(`notifications.types.${type}`, type)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="notification-endpoint">
                {t('notifications.endpoint', 'Webhook URL')}
              </Label>
              <Input
                id="notification-endpoint"
                value={form.endpoint}
                onChange={(e) =>
                  setForm((p) => ({ ...p, endpoint: e.target.value }))
                }
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="notification-secret">
                {t('notifications.secret', 'Signing secret (optional)')}
              </Label>
              <Input
                id="notification-secret"
                type="password"
                value={form.secret}
                onChange={(e) =>
                  setForm((p) => ({ ...p, secret: e.target.value }))
                }
                placeholder={t(
                  'notifications.secretPlaceholder',
                  'Leave empty to keep the current secret'
                )}
              />
            </div>

            <div className="space-y-2">
              <Label>{t('notifications.eventsTitle', 'Events')}</Label>
              {EVENTS.map((event) => (
                <div key={event.key} className="flex items-center gap-2">
                  <button
                    type="button"
                    className="flex items-center gap-2"
                    onClick={() => toggleEvent(event.key)}
                  >
                    {form.events.includes(event.key) ? (
                      <IconCheck className="h-4 w-4 text-primary" />
                    ) : (
                      <span className="h-4 w-4 rounded border" />
                    )}
                    <span className="text-sm">
                      {t(event.labelKey, event.fallback)}
                    </span>
                  </button>
                </div>
              ))}
            </div>

            <div className="flex items-center justify-between">
              <Label htmlFor="notification-enabled">
                {t('notifications.enabled', 'Enabled')}
              </Label>
              <Switch
                id="notification-enabled"
                checked={form.enabled}
                onCheckedChange={(checked) =>
                  setForm((p) => ({ ...p, enabled: checked }))
                }
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setEditorOpen(false)}>
              {t('common.actions.cancel', 'Cancel')}
            </Button>
            <Button
              disabled={!canSave || saveMutation.isPending}
              onClick={() => saveMutation.mutate()}
            >
              {editing
                ? t('common.actions.save', 'Save')
                : t('common.actions.create', 'Create')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete confirmation */}
      <DeleteConfirmationDialog
        open={deleting !== null}
        onOpenChange={(open) => !open && setDeleting(null)}
        resourceName={deleting?.name ?? ''}
        resourceType="notification channel"
        onConfirm={() => deleting && deleteMutation.mutate(deleting.id)}
        isDeleting={deleteMutation.isPending}
      />
    </Card>
  )
}
