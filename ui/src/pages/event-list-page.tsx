import { useCallback, useMemo, useState } from 'react'
import { createColumnHelper } from '@tanstack/react-table'
import { Event } from 'kubernetes-types/core/v1'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'

import { createSearchFilter } from '@/lib/k8s'
import { formatDate } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ResourceTable } from '@/components/resource-table'

const eventSearchFilter = createSearchFilter<Event>(
  (e) => e.metadata?.name,
  (e) => e.reason,
  (e) => e.message,
  (e) => e.type,
  (e) => e.involvedObject?.name,
  (e) => e.involvedObject?.kind
)

// An event counts as alive while its last occurrence is inside this window;
// beyond it the API server TTL has usually removed it, hence "dead".
const ALIVE_WINDOW_MS = 30 * 60 * 1000

type AliveFilter = 'all' | 'alive' | 'dead'

function eventLastSeen(event: Event): number {
  const seen =
    event.lastTimestamp ||
    event.eventTime ||
    event.series?.lastObservedTime ||
    event.metadata?.creationTimestamp
  if (!seen) return 0
  return new Date(seen).getTime()
}

function matchesAliveFilter(event: Event, filter: AliveFilter): boolean {
  if (filter === 'all') return true
  const lastSeen = eventLastSeen(event)
  const isAlive = lastSeen > Date.now() - ALIVE_WINDOW_MS
  return filter === 'alive' ? isAlive : !isAlive
}

const columnHelper = createColumnHelper<Event>()

export function EventListPage() {
  const { t } = useTranslation()
  const [aliveFilter, setAliveFilter] = useState<AliveFilter>('alive')

  const eventFilter = useCallback(
    (event: Event, query: string) =>
      eventSearchFilter(event, query) && matchesAliveFilter(event, aliveFilter),
    [aliveFilter]
  )

  const aliveFilterToolbar = useMemo(
    () => (
      <Select
        value={aliveFilter}
        onValueChange={(value) => setAliveFilter(value as AliveFilter)}
      >
        <SelectTrigger
          size="sm"
          className="text-muted-foreground w-40"
          aria-label={t('events.aliveFilter')}
        >
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="alive">{t('events.alive')}</SelectItem>
          <SelectItem value="dead">{t('events.dead')}</SelectItem>
          <SelectItem value="all">{t('common.values.all', 'All')}</SelectItem>
        </SelectContent>
      </Select>
    ),
    [aliveFilter, t]
  )

  const columns = useMemo(
    () => [
      columnHelper.accessor(
        (row) =>
          `${row.involvedObject?.kind || ''} ${row.involvedObject?.name || ''}`,
        {
          id: 'involvedObject',
          header: t('events.involvedObject', 'Involved Object'),
          enableColumnFilter: true,
          cell: ({ row }) => {
            const obj = row.original.involvedObject
            if (!obj) return '-'

            const kind = obj.kind || ''
            const name = obj.name || ''
            const namespace = obj.namespace

            const resourcePath = kind.toLowerCase() + 's'
            const link = namespace
              ? `/${resourcePath}/${namespace}/${name}`
              : `/${resourcePath}/${name}`

            return (
              <div className="flex flex-col gap-0.5">
                <div className="text-xs text-muted-foreground">{kind}</div>
                <Link to={link} className="font-medium app-link text-sm">
                  {name}
                </Link>
              </div>
            )
          },
        }
      ),
      columnHelper.accessor((row) => row.type, {
        id: 'type',
        header: t('common.fields.type'),
        enableColumnFilter: true,
        cell: ({ getValue }) => {
          const type = getValue() || ''
          const variant = type === 'Normal' ? 'default' : 'destructive'
          return <Badge variant={variant}>{type}</Badge>
        },
      }),
      columnHelper.accessor((row) => row.reason, {
        id: 'reason',
        header: t('common.fields.reason'),
        cell: ({ getValue }) => (
          <div className="font-medium">{getValue() || '-'}</div>
        ),
      }),
      columnHelper.accessor((row) => row.message, {
        id: 'message',
        header: t('common.fields.message'),
        cell: ({ getValue }) => (
          <div className="text-sm max-w-md whitespace-pre-wrap break-words">
            {getValue() || '-'}
          </div>
        ),
      }),
      columnHelper.accessor((row) => row.reportingComponent, {
        id: 'source',
        header: t('common.fields.source'),
        cell: ({ getValue }) => (
          <div className="text-muted-foreground text-sm max-w-sm whitespace-pre-wrap break-words">
            {getValue() || '-'}
          </div>
        ),
      }),
      columnHelper.accessor((row) => row.count ?? 1, {
        id: 'count',
        header: t('common.fields.count'),
        cell: ({ getValue }) => {
          const count = getValue() || 1
          return <span className="text-muted-foreground text-sm">{count}</span>
        },
      }),
      columnHelper.accessor(
        (row) =>
          row.lastTimestamp || row.eventTime || row.metadata?.creationTimestamp,
        {
          id: 'lastSeen',
          header: t('common.fields.lastSeen'),
          cell: ({ getValue }) => {
            const dateStr = formatDate(getValue() || '')
            return (
              <span className="text-muted-foreground text-sm">{dateStr}</span>
            )
          },
        }
      ),
    ],
    [t]
  )

  return (
    <ResourceTable<Event>
      resourceName="Events"
      columns={columns}
      clusterScope={false}
      searchQueryFilter={eventFilter}
      extraToolbars={[aliveFilterToolbar]}
    />
  )
}
