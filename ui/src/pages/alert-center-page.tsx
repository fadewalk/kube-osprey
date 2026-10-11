import { useMemo, useState } from 'react'
import {
  IconBellOff,
  IconPlus,
  IconRefresh,
  IconTrash,
  IconX,
} from '@tabler/icons-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  createSilence,
  deleteSilence,
  fetchAlerts,
  fetchSilences,
  type AlertManagerAlert,
  type AlertManagerMatcher,
  type AlertManagerSilence,
} from '@/lib/api/alertmanager'
import { formatDate } from '@/lib/utils'
import { useCluster } from '@/hooks/use-cluster'
import { usePageTitle } from '@/hooks/use-page-title'
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'

const severityClass = (severity: string) => {
  if (severity === 'critical') return 'bg-destructive text-white'
  if (severity === 'warning') return 'bg-amber-500 text-white'
  return 'bg-muted text-muted-foreground'
}

function alertSummary(alert: AlertManagerAlert): string {
  return (
    alert.annotations?.summary ||
    alert.annotations?.message ||
    alert.labels?.alertname ||
    alert.fingerprint ||
    ''
  )
}

function isSilenced(alert: AlertManagerAlert): boolean {
  return (alert.status?.silencedBy?.length ?? 0) > 0
}

function toLocalInputValue(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

interface MatcherRow {
  name: string
  operator: '=' | '!=' | '=~' | '!~'
  value: string
}

function toApiMatcher(row: MatcherRow): AlertManagerMatcher {
  const isRegex = row.operator === '=~' || row.operator === '!~'
  return {
    name: row.name.trim(),
    value: row.value,
    isRegex,
    operator: row.operator === '=' ? undefined : row.operator,
  }
}

export function AlertCenterPage() {
  const { t } = useTranslation()
  const { currentCluster } = useCluster()
  usePageTitle(t('nav.alertCenter'))
  const queryClient = useQueryClient()

  const [severityFilter, setSeverityFilter] = useState('all')
  const [stateFilter, setStateFilter] = useState('all')
  const [detailAlert, setDetailAlert] = useState<AlertManagerAlert | null>(null)
  const [silenceTarget, setSilenceTarget] = useState<AlertManagerAlert | null>(
    null
  )

  const alertsQuery = useQuery({
    queryKey: ['alertmanager-alerts', currentCluster],
    queryFn: ({ signal }) => fetchAlerts(signal),
    enabled: !!currentCluster,
    refetchInterval: 30000,
    retry: false,
  })
  const silencesQuery = useQuery({
    queryKey: ['alertmanager-silences', currentCluster],
    queryFn: ({ signal }) => fetchSilences(signal),
    enabled: !!currentCluster,
    refetchInterval: 60000,
    retry: false,
  })

  const alerts = useMemo(() => alertsQuery.data ?? [], [alertsQuery.data])
  const filteredAlerts = useMemo(() => {
    return alerts
      .filter(
        (a) =>
          severityFilter === 'all' ||
          (a.labels?.severity ?? 'info') === severityFilter
      )
      .filter((a) => {
        if (stateFilter === 'all') return true
        if (stateFilter === 'active')
          return a.status?.state === 'active' && !isSilenced(a)
        return a.status?.state !== 'active' || isSilenced(a)
      })
      .sort(
        (a, b) =>
          new Date(b.startsAt).getTime() - new Date(a.startsAt).getTime()
      )
  }, [alerts, severityFilter, stateFilter])

  const stats = useMemo(
    () => ({
      firing: alerts.filter((a) => a.status?.state === 'active').length,
      critical: alerts.filter(
        (a) => a.status?.state === 'active' && a.labels?.severity === 'critical'
      ).length,
      warning: alerts.filter(
        (a) => a.status?.state === 'active' && a.labels?.severity === 'warning'
      ).length,
      silenced: alerts.filter(isSilenced).length,
    }),
    [alerts]
  )

  const notConfigured =
    /not configured/i.test(alertsQuery.error?.message ?? '') ||
    /not configured/i.test(silencesQuery.error?.message ?? '')

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteSilence(id),
    onSuccess: () => {
      toast.success(t('alertCenter.silences.deleted'))
      queryClient.invalidateQueries({ queryKey: ['alertmanager-silences'] })
      queryClient.invalidateQueries({ queryKey: ['alertmanager-alerts'] })
    },
    onError: (error: Error) => toast.error(error.message),
  })

  if (notConfigured) {
    return (
      <div className="space-y-4">
        <PageHeader />
        <Card>
          <CardContent className="flex flex-col items-center gap-3 py-10 text-center">
            <IconBellOff className="h-10 w-10 text-muted-foreground" />
            <p className="text-sm font-medium">
              {t('alertCenter.notConfigured')}
            </p>
            <p className="max-w-md text-sm text-muted-foreground">
              {t('alertCenter.configureHint')}
            </p>
            <Button
              variant="outline"
              onClick={() => window.open('/settings', '_self')}
            >
              {t('alertCenter.goToSettings')}
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <PageHeader />

      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        <StatCard
          label={t('alertCenter.stats.firing')}
          value={stats.firing}
          tone="destructive"
        />
        <StatCard
          label={t('alertCenter.stats.critical')}
          value={stats.critical}
          tone="destructive"
        />
        <StatCard
          label={t('alertCenter.stats.warning')}
          value={stats.warning}
          tone="warning"
        />
        <StatCard
          label={t('alertCenter.stats.silenced')}
          value={stats.silenced}
          tone="muted"
        />
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <CardTitle className="text-base">{t('alertCenter.title')}</CardTitle>
          <div className="flex items-center gap-2">
            <Select value={severityFilter} onValueChange={setSeverityFilter}>
              <SelectTrigger className="w-36">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">
                  {t('alertCenter.filters.severityAll')}
                </SelectItem>
                <SelectItem value="critical">
                  {t('alertCenter.filters.critical')}
                </SelectItem>
                <SelectItem value="warning">
                  {t('alertCenter.filters.warning')}
                </SelectItem>
                <SelectItem value="info">
                  {t('alertCenter.filters.info')}
                </SelectItem>
              </SelectContent>
            </Select>
            <Select value={stateFilter} onValueChange={setStateFilter}>
              <SelectTrigger className="w-36">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">
                  {t('alertCenter.filters.stateAll')}
                </SelectItem>
                <SelectItem value="active">
                  {t('alertCenter.filters.active')}
                </SelectItem>
                <SelectItem value="suppressed">
                  {t('alertCenter.filters.suppressed')}
                </SelectItem>
              </SelectContent>
            </Select>
            <Button
              variant="outline"
              size="icon"
              onClick={() => {
                alertsQuery.refetch()
                silencesQuery.refetch()
              }}
            >
              <IconRefresh className="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {alertsQuery.isLoading ? (
            <p className="py-6 text-center text-sm text-muted-foreground">
              {t('common.messages.loading')}
            </p>
          ) : filteredAlerts.length === 0 ? (
            <p className="py-6 text-center text-sm text-muted-foreground">
              {t('alertCenter.noAlerts')}
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-24">
                    {t('alertCenter.table.severity')}
                  </TableHead>
                  <TableHead className="w-28">
                    {t('alertCenter.table.state')}
                  </TableHead>
                  <TableHead>{t('alertCenter.table.summary')}</TableHead>
                  <TableHead className="w-56">
                    {t('alertCenter.table.age')}
                  </TableHead>
                  <TableHead className="w-24 text-right">
                    {t('alertCenter.table.actions')}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredAlerts.map((alert) => {
                  const severity = alert.labels?.severity ?? 'info'
                  const silenced = isSilenced(alert)
                  return (
                    <TableRow
                      key={alert.fingerprint || JSON.stringify(alert.labels)}
                    >
                      <TableCell>
                        <span
                          className={`inline-flex rounded-md px-2 py-0.5 text-xs font-medium ${severityClass(severity)}`}
                        >
                          {severity}
                        </span>
                      </TableCell>
                      <TableCell>
                        <Badge variant={silenced ? 'secondary' : 'outline'}>
                          {silenced
                            ? t('alertCenter.alertState.suppressed')
                            : t('alertCenter.alertState.active')}
                        </Badge>
                      </TableCell>
                      <TableCell className="max-w-md">
                        <button
                          type="button"
                          className="truncate text-left text-sm hover:underline"
                          onClick={() => setDetailAlert(alert)}
                          title={alertSummary(alert)}
                        >
                          {alertSummary(alert)}
                        </button>
                        {alert.labels?.alertname ? (
                          <span className="ml-2 font-mono text-xs text-muted-foreground">
                            {alert.labels.alertname}
                          </span>
                        ) : null}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {formatDate(alert.startsAt, true)}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setSilenceTarget(alert)}
                        >
                          <IconBellOff className="h-4 w-4" />
                          {t('alertCenter.silence')}
                        </Button>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <SilencesCard
        silencesQuery={silencesQuery}
        onDelete={(id) => {
          if (window.confirm(t('alertCenter.silences.confirmDelete'))) {
            deleteMutation.mutate(id)
          }
        }}
        onCreate={() => setSilenceTarget({} as AlertManagerAlert)}
      />

      <AlertDetailDialog
        alert={detailAlert}
        onClose={() => setDetailAlert(null)}
      />

      <CreateSilenceDialog
        key={silenceTarget?.fingerprint || 'blank'}
        target={silenceTarget}
        onClose={() => setSilenceTarget(null)}
        onCreated={() => {
          queryClient.invalidateQueries({ queryKey: ['alertmanager-silences'] })
          queryClient.invalidateQueries({ queryKey: ['alertmanager-alerts'] })
        }}
      />
    </div>
  )
}

function PageHeader() {
  const { t } = useTranslation()
  return (
    <div>
      <h1 className="text-lg font-extrabold">{t('nav.alertCenter')}</h1>
      <p className="text-sm text-muted-foreground">
        {t('alertCenter.description')}
      </p>
    </div>
  )
}

function StatCard({
  label,
  value,
  tone,
}: {
  label: string
  value: number
  tone: 'destructive' | 'warning' | 'muted'
}) {
  const toneClass =
    tone === 'destructive'
      ? 'text-destructive'
      : tone === 'warning'
        ? 'text-amber-500'
        : 'text-muted-foreground'
  return (
    <Card>
      <CardContent className="px-4 py-3">
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className={`text-2xl font-bold ${toneClass}`}>{value}</p>
      </CardContent>
    </Card>
  )
}

function AlertDetailDialog({
  alert,
  onClose,
}: {
  alert: AlertManagerAlert | null
  onClose: () => void
}) {
  const { t } = useTranslation()
  return (
    <Dialog open={!!alert} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-[640px]">
        <DialogHeader>
          <DialogTitle>{t('alertCenter.details.title')}</DialogTitle>
        </DialogHeader>
        {alert ? (
          <div className="space-y-3 text-sm">
            <div>
              <p className="font-medium">{alertSummary(alert)}</p>
              {alert.annotations?.description ? (
                <p className="mt-1 text-muted-foreground">
                  {alert.annotations.description}
                </p>
              ) : null}
            </div>
            <KeyValueBlock
              label={t('alertCenter.table.age')}
              value={formatDate(alert.startsAt, true)}
            />
            {alert.generatorURL ? (
              <KeyValueBlock
                label={t('alertCenter.details.generator')}
                value={alert.generatorURL}
                link
              />
            ) : null}
            {alert.fingerprint ? (
              <KeyValueBlock
                label={t('alertCenter.details.fingerprint')}
                value={alert.fingerprint}
              />
            ) : null}
            <LabelledList
              label={t('alertCenter.details.labels')}
              entries={alert.labels}
            />
            <LabelledList
              label={t('alertCenter.details.annotations')}
              entries={alert.annotations}
            />
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  )
}

function KeyValueBlock({
  label,
  value,
  link,
}: {
  label: string
  value: string
  link?: boolean
}) {
  return (
    <div>
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      {link ? (
        <a
          href={value}
          target="_blank"
          rel="noreferrer"
          className="break-all text-sm text-primary hover:underline"
        >
          {value}
        </a>
      ) : (
        <p className="break-all font-mono text-xs">{value}</p>
      )}
    </div>
  )
}

function LabelledList({
  label,
  entries,
}: {
  label: string
  entries?: Record<string, string>
}) {
  const items = Object.entries(entries ?? {})
  if (items.length === 0) return null
  return (
    <div>
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <div className="mt-1 space-y-1 rounded-md border p-2">
        {items.map(([key, value]) => (
          <div key={key} className="flex gap-2 text-xs">
            <span className="font-mono font-medium">{key}</span>
            <span className="break-all text-muted-foreground">{value}</span>
          </div>
        ))}
      </div>
    </div>
  )
}

function SilencesCard({
  silencesQuery,
  onDelete,
  onCreate,
}: {
  silencesQuery: { data?: AlertManagerSilence[]; isLoading: boolean }
  onDelete: (id: string) => void
  onCreate: () => void
}) {
  const { t } = useTranslation()
  const silences = silencesQuery.data ?? []
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0">
        <CardTitle className="text-base">
          {t('alertCenter.silences.title')}
        </CardTitle>
        <Button size="sm" onClick={onCreate}>
          <IconPlus className="h-4 w-4" />
          {t('alertCenter.silences.new')}
        </Button>
      </CardHeader>
      <CardContent>
        {silencesQuery.isLoading ? (
          <p className="py-4 text-center text-sm text-muted-foreground">
            {t('common.messages.loading')}
          </p>
        ) : silences.length === 0 ? (
          <p className="py-4 text-center text-sm text-muted-foreground">
            {t('alertCenter.silences.empty')}
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-24">
                  {t('alertCenter.table.state')}
                </TableHead>
                <TableHead>{t('alertCenter.silences.matchers')}</TableHead>
                <TableHead className="w-56">
                  {t('alertCenter.silences.endsAt')}
                </TableHead>
                <TableHead className="w-40">
                  {t('alertCenter.silences.createdBy')}
                </TableHead>
                <TableHead className="w-16 text-right">
                  {t('alertCenter.table.actions')}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {silences.map((silence) => (
                <TableRow key={silence.id}>
                  <TableCell>
                    <Badge
                      variant={
                        silence.status?.state === 'active'
                          ? 'default'
                          : silence.status?.state === 'pending'
                            ? 'secondary'
                            : 'outline'
                      }
                    >
                      {silence.status?.state}
                    </Badge>
                  </TableCell>
                  <TableCell className="max-w-sm">
                    <p
                      className="truncate font-mono text-xs"
                      title={matchersToText(silence.matchers)}
                    >
                      {matchersToText(silence.matchers)}
                    </p>
                    {silence.comment ? (
                      <p
                        className="truncate text-xs text-muted-foreground"
                        title={silence.comment}
                      >
                        {silence.comment}
                      </p>
                    ) : null}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {formatDate(silence.endsAt, true)}
                  </TableCell>
                  <TableCell className="truncate text-sm text-muted-foreground">
                    {silence.createdBy || '-'}
                  </TableCell>
                  <TableCell className="text-right">
                    {silence.status?.state !== 'expired' ? (
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => onDelete(silence.id)}
                      >
                        <IconTrash className="h-4 w-4 text-destructive" />
                      </Button>
                    ) : null}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}

function matchersToText(matchers: AlertManagerMatcher[]): string {
  return (matchers ?? [])
    .map(
      (m) => `${m.name}${m.operator || (m.isRegex ? '=~' : '=')}"${m.value}"`
    )
    .join(', ')
}

function CreateSilenceDialog({
  target,
  onClose,
  onCreated,
}: {
  target: AlertManagerAlert | null
  onClose: () => void
  onCreated: () => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const isOpen = !!target
  const prefill: MatcherRow[] = target?.labels?.alertname
    ? [
        { name: 'alertname', operator: '=', value: target.labels.alertname },
        ...(target.labels.namespace
          ? [
              {
                name: 'namespace',
                operator: '=' as const,
                value: target.labels.namespace,
              },
            ]
          : []),
      ]
    : [{ name: 'alertname', operator: '=', value: '' }]

  const [rows, setRows] = useState<MatcherRow[]>(prefill)
  const [endsAt, setEndsAt] = useState(() =>
    toLocalInputValue(new Date(Date.now() + 4 * 3600 * 1000))
  )
  const [comment, setComment] = useState('')

  const mutation = useMutation({
    mutationFn: () =>
      createSilence({
        matchers: rows.map(toApiMatcher),
        endsAt: new Date(endsAt).toISOString(),
        comment,
      }),
    onSuccess: () => {
      toast.success(t('alertCenter.silences.created'))
      queryClient.invalidateQueries({ queryKey: ['alertmanager-silences'] })
      queryClient.invalidateQueries({ queryKey: ['alertmanager-alerts'] })
      onCreated()
      onClose()
    },
    onError: (error: Error) => toast.error(error.message),
  })

  const canSubmit =
    rows.length > 0 &&
    rows.every((r) => r.name.trim() && r.value) &&
    comment.trim().length > 0 &&
    !Number.isNaN(new Date(endsAt).getTime()) &&
    new Date(endsAt).getTime() > Date.now()

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-[560px]">
        <DialogHeader>
          <DialogTitle>{t('alertCenter.silences.new')}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <p className="text-sm text-muted-foreground">
            {t('alertCenter.silences.matchersHint')}
          </p>
          <div className="space-y-2">
            {rows.map((row, index) => (
              <div key={index} className="flex items-center gap-2">
                <Input
                  className="w-40"
                  placeholder={t('alertCenter.silences.matcherName')}
                  value={row.name}
                  onChange={(e) =>
                    setRows((prev) =>
                      prev.map((r, i) =>
                        i === index ? { ...r, name: e.target.value } : r
                      )
                    )
                  }
                />
                <Select
                  value={row.operator}
                  onValueChange={(op) =>
                    setRows((prev) =>
                      prev.map((r, i) =>
                        i === index
                          ? { ...r, operator: op as MatcherRow['operator'] }
                          : r
                      )
                    )
                  }
                >
                  <SelectTrigger className="w-20">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="=">=</SelectItem>
                    <SelectItem value="!=">!=</SelectItem>
                    <SelectItem value="=~">=~</SelectItem>
                    <SelectItem value="!~">!~</SelectItem>
                  </SelectContent>
                </Select>
                <Input
                  placeholder={t('alertCenter.silences.matcherValue')}
                  value={row.value}
                  onChange={(e) =>
                    setRows((prev) =>
                      prev.map((r, i) =>
                        i === index ? { ...r, value: e.target.value } : r
                      )
                    )
                  }
                />
                <Button
                  variant="ghost"
                  size="icon"
                  disabled={rows.length <= 1}
                  onClick={() =>
                    setRows((prev) => prev.filter((_, i) => i !== index))
                  }
                >
                  <IconX className="h-4 w-4" />
                </Button>
              </div>
            ))}
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                setRows((prev) => [
                  ...prev,
                  { name: '', operator: '=', value: '' },
                ])
              }
            >
              <IconPlus className="h-4 w-4" />
              {t('alertCenter.silences.addMatcher')}
            </Button>
          </div>
          <div className="space-y-2">
            <Label htmlFor="silence-ends-at">
              {t('alertCenter.silences.endsAt')}
            </Label>
            <Input
              id="silence-ends-at"
              type="datetime-local"
              value={endsAt}
              onChange={(e) => setEndsAt(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="silence-comment">
              {t('alertCenter.silences.comment')}
            </Label>
            <Textarea
              id="silence-comment"
              rows={2}
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              placeholder={t('alertCenter.silences.commentPlaceholder')}
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t('common.actions.cancel')}
          </Button>
          <Button
            disabled={!canSubmit || mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            {t('alertCenter.silences.submit')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
