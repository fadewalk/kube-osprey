import { IconActivity } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'

import { apiClient } from '@/lib/api-client'
import { useCluster } from '@/hooks/use-cluster'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

interface HealthCheck {
  id: string
  category: string
  severity: string
  count: number
}

interface HealthReport {
  score: number
  generatedAt: string
  checks: HealthCheck[]
}

const scoreTone = (score: number) =>
  score >= 90
    ? 'bg-emerald-500'
    : score >= 70
      ? 'bg-amber-500'
      : 'bg-destructive'

export function OverviewHealthCard() {
  const { t } = useTranslation()
  const { currentCluster } = useCluster()

  const reportQuery = useQuery({
    queryKey: ['overview-health', currentCluster],
    queryFn: ({ signal }) =>
      apiClient.get<HealthReport>('/health-report', { signal }),
    enabled: !!currentCluster,
    refetchInterval: 120000,
    retry: false,
  })
  const report = reportQuery.data

  const criticalCount = (report?.checks ?? [])
    .filter((check) => check.severity === 'critical')
    .reduce((sum, check) => sum + (check.count || 1), 0)
  const warningCount = (report?.checks ?? [])
    .filter((check) => check.severity === 'warning')
    .reduce((sum, check) => sum + (check.count || 1), 0)

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center justify-between text-base">
          <span className="flex items-center gap-2">
            <IconActivity className="h-4 w-4" />
            {t('overview.health.title')}
          </span>
          <Link
            to="/diagnostics"
            className="text-xs font-normal text-muted-foreground hover:text-foreground"
          >
            {t('overview.health.viewDiagnostics')}
          </Link>
        </CardTitle>
      </CardHeader>
      <CardContent>
        {reportQuery.isLoading ? (
          <p className="text-sm text-muted-foreground">
            {t('common.messages.loading')}
          </p>
        ) : !report ? (
          <p className="text-sm text-muted-foreground">
            {t('overview.health.unavailable')}
          </p>
        ) : (
          <div className="flex items-center gap-4">
            <div
              className={`flex h-14 w-14 shrink-0 items-center justify-center rounded-full text-lg font-bold text-white ${scoreTone(report.score)}`}
            >
              {report.score}
            </div>
            <div className="text-sm">
              <p>{t('overview.health.score')}</p>
              <p className="text-muted-foreground">
                {report.checks.length === 0
                  ? t('overview.health.allHealthy')
                  : t('overview.health.riskSummary', {
                      critical: criticalCount,
                      warning: warningCount,
                    })}
              </p>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
