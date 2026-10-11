import { IconRefresh } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { apiClient } from '@/lib/api-client'
import { formatDate } from '@/lib/utils'
import { useCluster } from '@/hooks/use-cluster'
import { usePageTitle } from '@/hooks/use-page-title'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

interface HealthCheck {
  id: string
  category: string
  severity: string
  count: number
  sample?: string
}

interface HealthReport {
  score: number
  generatedAt: string
  prometheusEnabled: boolean
  checks: HealthCheck[]
  controlPlane: Array<{ name: string; ready: number; total: number }>
  topNodes: Array<{ node: string; cpuPercent: number; memPercent: number }>
  topPods: Array<{
    namespace: string
    pod: string
    cpuCores: number
    memBytes: number
  }>
}

const scoreTone = (score: number) =>
  score >= 90
    ? 'bg-emerald-500'
    : score >= 70
      ? 'bg-amber-500'
      : 'bg-destructive'

function percentBar(percent: number) {
  const clamped = Math.max(0, Math.min(100, percent))
  const tone =
    clamped >= 90
      ? 'bg-destructive'
      : clamped >= 70
        ? 'bg-amber-500'
        : 'bg-emerald-500'
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-24 overflow-hidden rounded-full bg-muted">
        <div
          className={`h-full rounded-full ${tone}`}
          style={{ width: `${clamped}%` }}
        />
      </div>
      <span className="text-xs text-muted-foreground">
        {percent.toFixed(1)}%
      </span>
    </div>
  )
}

export function HealthDiagnosticsPage() {
  const { t } = useTranslation()
  const { currentCluster } = useCluster()
  usePageTitle(t('nav.diagnostics'))

  const reportQuery = useQuery({
    queryKey: ['health-report', currentCluster],
    queryFn: ({ signal }) =>
      apiClient.get<HealthReport>('/health-report', { signal }),
    enabled: !!currentCluster,
    refetchInterval: 60000,
  })
  const report = reportQuery.data

  const checksByCategory = (report?.checks ?? []).reduce<
    Record<string, HealthCheck[]>
  >((acc, check) => {
    ;(acc[check.category] ??= []).push(check)
    return acc
  }, {})

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-extrabold">{t('nav.diagnostics')}</h1>
          <p className="text-sm text-muted-foreground">
            {t('health.description')}
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => reportQuery.refetch()}
        >
          <IconRefresh className="h-4 w-4" />
          {t('common.actions.refresh')}
        </Button>
      </div>

      <Card>
        <CardContent className="flex flex-wrap items-center gap-6 px-4 py-4">
          <div className="flex items-center gap-4">
            <div
              className={`flex h-20 w-20 items-center justify-center rounded-full text-2xl font-bold text-white ${scoreTone(report?.score ?? 100)}`}
            >
              {report?.score ?? '-'}
            </div>
            <div>
              <p className="text-sm font-medium">{t('health.score')}</p>
              <p className="text-xs text-muted-foreground">
                {report ? formatDate(report.generatedAt) : ''}
              </p>
            </div>
          </div>
          <div className="text-sm text-muted-foreground">
            {reportQuery.isLoading
              ? t('common.messages.loading')
              : (report?.checks ?? []).length === 0
                ? t('health.allHealthy')
                : t('health.issuesFound', {
                    count: (report?.checks ?? []).length,
                  })}
          </div>
        </CardContent>
      </Card>

      {(report?.checks ?? []).length > 0 ? (
        <div className="grid gap-3 md:grid-cols-2">
          {Object.entries(checksByCategory).map(([category, checks]) => (
            <Card key={category}>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm">
                  {t(`health.categories.${category}`)}
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-2">
                {checks.map((check) => (
                  <div key={check.id} className="flex items-start gap-2">
                    <Badge
                      variant={
                        check.severity === 'critical'
                          ? 'destructive'
                          : 'secondary'
                      }
                    >
                      {t(`health.severities.${check.severity}`)}
                    </Badge>
                    <div className="min-w-0 flex-1 text-sm">
                      <p>
                        {t(`health.checks.${check.id}`, { count: check.count })}
                      </p>
                      {check.sample ? (
                        <p className="truncate font-mono text-xs text-muted-foreground">
                          {check.sample}
                        </p>
                      ) : null}
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>
          ))}
        </div>
      ) : null}

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm">{t('health.controlPlane')}</CardTitle>
        </CardHeader>
        <CardContent>
          {(report?.controlPlane ?? []).length === 0 ? (
            <p className="py-2 text-sm text-muted-foreground">
              {t('health.controlPlaneEmpty')}
            </p>
          ) : (
            <div className="flex flex-wrap gap-3">
              {(report?.controlPlane ?? []).map((component) => (
                <div
                  key={component.name}
                  className="rounded-md border px-3 py-2 text-sm"
                >
                  <p className="font-mono text-xs">{component.name}</p>
                  <p
                    className={`font-medium ${component.ready < component.total ? 'text-amber-500' : 'text-emerald-500'}`}
                  >
                    {t('health.readyCount', {
                      ready: component.ready,
                      total: component.total,
                    })}
                  </p>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <div className="grid gap-3 lg:grid-cols-2">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">
              {t('health.topNodes.title')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {!report?.prometheusEnabled ? (
              <p className="py-2 text-sm text-muted-foreground">
                {t('health.prometheusRequired')}
              </p>
            ) : (report?.topNodes ?? []).length === 0 ? (
              <p className="py-2 text-sm text-muted-foreground">
                {t('health.noData')}
              </p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('common.fields.name')}</TableHead>
                    <TableHead>CPU</TableHead>
                    <TableHead>{t('health.topNodes.memory')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(report?.topNodes ?? []).map((node) => (
                    <TableRow key={node.node}>
                      <TableCell className="font-mono text-xs">
                        {node.node}
                      </TableCell>
                      <TableCell>
                        {Number.isNaN(node.cpuPercent)
                          ? '-'
                          : percentBar(node.cpuPercent)}
                      </TableCell>
                      <TableCell>
                        {Number.isNaN(node.memPercent)
                          ? '-'
                          : percentBar(node.memPercent)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">
              {t('health.topPods.title')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            {!report?.prometheusEnabled ? (
              <p className="py-2 text-sm text-muted-foreground">
                {t('health.prometheusRequired')}
              </p>
            ) : (report?.topPods ?? []).length === 0 ? (
              <p className="py-2 text-sm text-muted-foreground">
                {t('health.noData')}
              </p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('common.fields.name')}</TableHead>
                    <TableHead>CPU</TableHead>
                    <TableHead>{t('health.topPods.memory')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(report?.topPods ?? []).map((pod) => (
                    <TableRow key={`${pod.namespace}/${pod.pod}`}>
                      <TableCell className="font-mono text-xs">
                        {pod.namespace}/{pod.pod}
                      </TableCell>
                      <TableCell>{pod.cpuCores.toFixed(3)} cores</TableCell>
                      <TableCell>
                        {(pod.memBytes / 1024 / 1024).toFixed(1)} MiB
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
