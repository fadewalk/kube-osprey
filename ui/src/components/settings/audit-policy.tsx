import { useEffect, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { updateGeneralSetting, useGeneralSetting } from '@/lib/api/admin'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

// Modules the audit middleware can classify; keep in sync with
// pkg/middleware/operation_audit.go.
const KNOWN_MODULES = [
  'account',
  'alertmanager',
  'changeplans',
  'clusters',
  'ldap',
  'oauth',
  'plugins',
  'resources',
  'roles',
  'settings',
  'templates',
  'webhook',
  'workloads',
] as const

export function AuditPolicy() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { data } = useGeneralSetting()

  const [retentionDays, setRetentionDays] = useState<string>('30')
  const [skipped, setSkipped] = useState<Set<string>>(new Set())

  useEffect(() => {
    if (!data) return
    setRetentionDays(String(data.auditRetentionDays ?? 30))
    setSkipped(
      new Set(
        (data.auditSkippedModules ?? '')
          .split(',')
          .map((m) => m.trim())
          .filter(Boolean)
      )
    )
  }, [data])

  const mutation = useMutation({
    mutationFn: () =>
      updateGeneralSetting({
        auditRetentionDays: Number(retentionDays) || 30,
        auditSkippedModules: [...skipped].join(','),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['general-setting'] })
      toast.success(t('auditPolicy.saved', 'Audit policy saved'))
    },
    onError: (err: Error) => toast.error(err.message),
  })

  const toggleModule = (name: string, on: boolean) => {
    setSkipped((prev) => {
      const next = new Set(prev)
      if (on) {
        next.add(name)
      } else {
        next.delete(name)
      }
      return next
    })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('auditPolicy.title', 'Audit policy')}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid max-w-xs gap-1">
          <Label htmlFor="audit-retention-days">
            {t('auditPolicy.retentionDays', 'Log retention (days)')}
          </Label>
          <Input
            id="audit-retention-days"
            type="number"
            min={1}
            max={3650}
            value={retentionDays}
            onChange={(e) => setRetentionDays(e.target.value)}
          />
          <p className="text-muted-foreground text-xs">
            {t(
              'auditPolicy.retentionHint',
              'Older operation logs are purged automatically (1–3650 days).'
            )}
          </p>
        </div>

        <div className="space-y-2">
          <Label>{t('auditPolicy.skippedModules', 'Skipped modules')}</Label>
          <p className="text-muted-foreground text-xs">
            {t(
              'auditPolicy.skippedModulesHint',
              'Changes in the selected modules are not written to the audit log.'
            )}
          </p>
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4">
            {KNOWN_MODULES.map((name) => (
              <label
                key={name}
                className="flex items-center justify-between rounded-md border px-3 py-2 text-sm"
              >
                <span>{name}</span>
                <Switch
                  checked={skipped.has(name)}
                  onCheckedChange={(on) => toggleModule(name, on)}
                />
              </label>
            ))}
          </div>
        </div>

        <Button onClick={() => mutation.mutate()} disabled={mutation.isPending}>
          {t('auditPolicy.save', 'Save policy')}
        </Button>
      </CardContent>
    </Card>
  )
}
