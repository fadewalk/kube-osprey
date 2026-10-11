import { useAuth } from '@/contexts/auth-context'
import { IconChecklist, IconKey, IconShieldHalf } from '@tabler/icons-react'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'

import { fetchAlerts } from '@/lib/api/alertmanager'
import { usePendingChangePlanCount } from '@/lib/api/changeplans'
import { useCluster } from '@/hooks/use-cluster'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

interface TodoItem {
  key: string
  label: string
  count: number
  to: string
  tone: 'amber' | 'red'
}

const PASSWORD_EXPIRY_WARN_DAYS = 7

function passwordExpiryDays(expiresAt?: string): number | null {
  if (!expiresAt) return null
  const diffMs = new Date(expiresAt).getTime() - Date.now()
  return Math.ceil(diffMs / 86400000)
}

export function OverviewTodos() {
  const { t } = useTranslation()
  const { user } = useAuth()
  const { currentCluster } = useCluster()

  const { data: pendingPlans } = usePendingChangePlanCount({
    refreshInterval: 60000,
  })

  const alertsQuery = useQuery({
    queryKey: ['overview-alerts', currentCluster],
    queryFn: ({ signal }) => fetchAlerts(signal),
    enabled: !!currentCluster,
    refetchInterval: 60000,
    retry: false,
  })
  // AlertManager not configured (503) or RBAC-denied (403) simply means the
  // todo does not apply for this user; the page must still render.
  const firingAlerts = alertsQuery.data
    ? alertsQuery.data.filter((a) => a.status?.state === 'active').length
    : null

  const items: TodoItem[] = []
  const expiryDays = passwordExpiryDays(user?.passwordExpiresAt)
  if (expiryDays !== null && expiryDays <= PASSWORD_EXPIRY_WARN_DAYS) {
    items.push({
      key: 'password-expiry',
      label:
        expiryDays <= 0
          ? t('overview.todos.passwordExpired')
          : t('overview.todos.passwordExpiring', { days: expiryDays }),
      count: 0,
      to: '/settings',
      tone: 'red',
    })
  }
  if ((pendingPlans ?? 0) > 0) {
    items.push({
      key: 'plans',
      label: t('overview.todos.pendingPlans'),
      count: pendingPlans ?? 0,
      to: '/change-plans',
      tone: 'amber',
    })
  }
  if ((firingAlerts ?? 0) > 0) {
    items.push({
      key: 'alerts',
      label: t('overview.todos.firingAlerts'),
      count: firingAlerts ?? 0,
      to: '/alerts',
      tone: 'red',
    })
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-base">
          <IconChecklist className="h-4 w-4" />
          {t('overview.todos.title')}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {t('overview.todos.empty')}
          </p>
        ) : (
          <div className="flex flex-col gap-2">
            {items.map((item) => (
              <Link
                key={item.key}
                to={item.to}
                className="flex items-center justify-between rounded-md border px-3 py-2 text-sm hover:bg-muted"
              >
                <span className="flex items-center gap-2">
                  {item.key === 'password-expiry' ? (
                    <IconKey className="h-4 w-4 text-destructive" />
                  ) : (
                    <IconShieldHalf
                      className={`h-4 w-4 ${item.tone === 'red' ? 'text-destructive' : 'text-amber-500'}`}
                    />
                  )}
                  {item.label}
                </span>
                {item.count > 0 && (
                  <span
                    className={`font-semibold ${item.tone === 'red' ? 'text-destructive' : 'text-amber-500'}`}
                  >
                    {item.count}
                  </span>
                )}
              </Link>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
