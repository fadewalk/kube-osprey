import { useMemo } from 'react'
import {
  IconAlertTriangle,
  IconArrowRight,
  IconNetwork,
  IconPackage,
  IconServer2,
  IconStack2,
} from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'
import { Link, useParams } from 'react-router-dom'

import { ResourceType } from '@/types/api'
import { useResources } from '@/lib/api'
import { getAge } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

type NamespaceResource =
  | 'deployments'
  | 'statefulsets'
  | 'daemonsets'
  | 'jobs'
  | 'cronjobs'
  | 'pods'
  | 'services'
  | 'ingresses'
  | 'configmaps'
  | 'secrets'
  | 'persistentvolumeclaims'
  | 'events'

interface ResourceRow {
  resource: NamespaceResource
  titleKey: string
}

const categoryRows: {
  titleKey: string
  icon: typeof IconStack2
  rows: ResourceRow[]
}[] = [
  {
    titleKey: 'namespaceDashboard.workloads',
    icon: IconServer2,
    rows: [
      { resource: 'deployments', titleKey: 'nav.deployments' },
      { resource: 'statefulsets', titleKey: 'nav.statefulsets' },
      { resource: 'daemonsets', titleKey: 'nav.daemonsets' },
      { resource: 'jobs', titleKey: 'nav.jobs' },
      { resource: 'cronjobs', titleKey: 'nav.cronjobs' },
    ],
  },
  {
    titleKey: 'namespaceDashboard.network',
    icon: IconNetwork,
    rows: [
      { resource: 'services', titleKey: 'nav.services' },
      { resource: 'ingresses', titleKey: 'nav.ingresses' },
    ],
  },
  {
    titleKey: 'namespaceDashboard.config',
    icon: IconStack2,
    rows: [
      { resource: 'configmaps', titleKey: 'nav.configMaps' },
      { resource: 'secrets', titleKey: 'nav.secrets' },
    ],
  },
  {
    titleKey: 'namespaceDashboard.storage',
    icon: IconPackage,
    rows: [
      {
        resource: 'persistentvolumeclaims',
        titleKey: 'nav.persistentvolumeclaims',
      },
    ],
  },
]

function ResourceCountRow({
  resource,
  titleKey,
  namespace,
}: {
  resource: NamespaceResource
  titleKey: string
  namespace: string
}) {
  const { t } = useTranslation()
  // enabled keeps queries for types the user cannot access from flooding the
  // log with 403s; errors render as an unavailable count instead.
  const { data, error } = useResources(resource as ResourceType, namespace)

  const count = error ? null : (data?.length ?? 0)

  return (
    <Link
      to={`/${resource}?namespace=${encodeURIComponent(namespace)}`}
      className="hover:bg-accent/50 -mx-2 flex items-center justify-between rounded-md px-2 py-1.5 text-sm"
    >
      <span className="truncate">{t(titleKey, { defaultValue: titleKey })}</span>
      <span className="flex items-center gap-1">
        <span className="text-muted-foreground tabular-nums">
          {count ?? '-'}
        </span>
        <IconArrowRight className="text-muted-foreground/60 h-3.5 w-3.5" />
      </span>
    </Link>
  )
}

function PodsRow({ namespace }: { namespace: string }) {
  const { t } = useTranslation()
  const { data: pods } = useResources('pods', namespace)

  const summary = useMemo(() => {
    const items = pods || []
    const running = items.filter(
      (p) => p.status?.phase === 'Running' && !p.metadata?.deletionTimestamp
    ).length
    const pending = items.filter((p) => p.status?.phase === 'Pending').length
    const failed = items.filter(
      (p) =>
        p.status?.phase === 'Failed' ||
        p.metadata?.deletionTimestamp !== undefined
    ).length
    return { total: items.length, running, pending, failed }
  }, [pods])

  return (
    <Link
      to={`/pods?namespace=${encodeURIComponent(namespace)}`}
      className="hover:bg-accent/50 -mx-2 flex items-center justify-between rounded-md px-2 py-1.5 text-sm"
    >
      <span className="truncate">{t('nav.pods', 'Pods')}</span>
      <span className="flex items-center gap-1.5">
        {summary.failed > 0 && (
          <Badge variant="destructive" className="gap-1 px-1.5">
            <IconAlertTriangle className="h-3 w-3" />
            {summary.failed}
          </Badge>
        )}
        {summary.pending > 0 && (
          <Badge variant="outline" className="px-1.5">
            {t('status.pending', 'Pending')} {summary.pending}
          </Badge>
        )}
        <span className="text-muted-foreground tabular-nums">
          {summary.running}/{summary.total}
        </span>
        <IconArrowRight className="text-muted-foreground/60 h-3.5 w-3.5" />
      </span>
    </Link>
  )
}

function RecentEventsCard({ namespace }: { namespace: string }) {
  const { t } = useTranslation()
  const { data: events } = useResources('events', namespace)

  const recent = useMemo(() => {
    return (events || [])
      .slice()
      .sort(
        (a, b) =>
          new Date(
            b.lastTimestamp || b.eventTime || b.metadata?.creationTimestamp || 0
          ).getTime() -
          new Date(
            a.lastTimestamp || a.eventTime || a.metadata?.creationTimestamp || 0
          ).getTime()
      )
      .slice(0, 8)
  }, [events])

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base">
          {t('namespaceDashboard.recentEvents', 'Recent Events')}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {(recent || []).length === 0 ? (
          <p className="text-muted-foreground py-4 text-center text-sm">
            {t('namespaceDashboard.noEvents', 'No recent events in this namespace.')}
          </p>
        ) : (
          recent.map((event) => (
            <div
              key={event.metadata?.uid}
              className="flex items-start justify-between gap-3 border-b pb-2 text-sm last:border-b-0 last:pb-0"
            >
              <div className="flex min-w-0 flex-col gap-0.5">
                <div className="flex items-center gap-2">
                  {event.type === 'Warning' && (
                    <IconAlertTriangle className="h-3.5 w-3.5 shrink-0 text-amber-500" />
                  )}
                  <span className="font-medium">{event.involvedObject?.name}</span>
                  <span className="text-muted-foreground truncate text-xs">
                    {event.reason}
                  </span>
                </div>
                <span className="text-muted-foreground line-clamp-2 text-xs">
                  {event.message}
                </span>
              </div>
              <span className="text-muted-foreground shrink-0 text-xs">
                {event.count && event.count > 1 ? `×${event.count}` : ''}
              </span>
            </div>
          ))
        )}
        <Link
          to={`/events?namespace=${encodeURIComponent(namespace)}`}
          className="app-link inline-flex items-center gap-1 text-sm"
        >
          {t('namespaceDashboard.viewAllEvents', 'View all events')}
          <IconArrowRight className="h-3.5 w-3.5" />
        </Link>
      </CardContent>
    </Card>
  )
}

export function NamespaceDashboard() {
  const { name = '' } = useParams()
  const { t } = useTranslation()
  const { data: namespaces } = useResources('namespaces')
  const namespace = (namespaces || []).find(
    (ns) => ns.metadata?.name === name
  )

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{name}</h1>
          <p className="text-muted-foreground mt-1 flex items-center gap-3 text-sm">
            {namespace?.status?.phase && (
              <Badge variant="outline">{namespace.status.phase}</Badge>
            )}
            {namespace?.metadata?.creationTimestamp && (
              <span>
                {t('common.messages.timeAgo', {
                  time: getAge(namespace.metadata.creationTimestamp),
                })}
              </span>
            )}
          </p>
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        {categoryRows.map((category) => (
          <Card key={category.titleKey}>
            <CardHeader className="flex flex-row items-center gap-2 pb-2">
              <category.icon className="text-muted-foreground h-4 w-4" />
              <CardTitle className="text-base">
                {t(category.titleKey)}
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-0.5">
              {category.rows.map((row) => (
                <ResourceCountRow
                  key={row.resource}
                  resource={row.resource}
                  titleKey={row.titleKey}
                  namespace={name}
                />
              ))}
              {category.titleKey === 'namespaceDashboard.workloads' && (
                <PodsRow namespace={name} />
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      <RecentEventsCard namespace={name} />
    </div>
  )
}

export default NamespaceDashboard
