import { useMemo } from 'react'
import { createColumnHelper } from '@tanstack/react-table'
import { Job } from 'kubernetes-types/batch/v1'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'

import { createSearchFilter } from '@/lib/k8s'
import { formatDate } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { ResourceTable } from '@/components/resource-table'

const jobSearchFilter = createSearchFilter<Job>(
  (j) => j.metadata?.name,
  (j) => j.metadata?.namespace
)

const columnHelper = createColumnHelper<Job>()

export function JobListPage() {
  const { t } = useTranslation()

  // Define columns for the job table
  const columns = useMemo(
    () => [
      columnHelper.accessor('metadata.name', {
        header: 'Name',
        cell: ({ row }) => (
          <div className="font-medium app-link">
            <Link
              to={`/jobs/${row.original.metadata!.namespace}/${
                row.original.metadata!.name
              }`}
            >
              {row.original.metadata!.name}
            </Link>
          </div>
        ),
      }),
      columnHelper.accessor(
        (row) => {
          const conditions = row.status?.conditions || []
          if (
            conditions.some(
              (condition) =>
                condition.type === 'Complete' && condition.status === 'True'
            )
          ) {
            return 'Complete'
          }
          if (
            conditions.some(
              (condition) =>
                condition.type === 'Failed' && condition.status === 'True'
            )
          ) {
            return 'Failed'
          }
          return 'Running'
        },
        {
          id: 'status_conditions',
          header: 'Status',
          cell: ({ getValue }) => {
            const status = getValue()
            const variant =
              status === 'Complete'
                ? 'default'
                : status === 'Failed'
                  ? 'destructive'
                  : 'secondary'

            return <Badge variant={variant}>{status}</Badge>
          },
        }
      ),
      columnHelper.accessor((row) => row.status?.succeeded || 0, {
        id: 'completions',
        header: 'Completions',
        cell: ({ row }) => {
          const status = row.original.status
          const succeeded = status?.succeeded || 0
          const completions = row.original.spec?.completions || 1
          return `${succeeded}/${completions}`
        },
      }),
      columnHelper.accessor('status.startTime', {
        header: 'Started',
        cell: ({ getValue }) => {
          const startTime = getValue()
          if (!startTime) return '-'

          const dateStr = formatDate(startTime)

          return (
            <span className="text-muted-foreground text-sm">{dateStr}</span>
          )
        },
      }),
      columnHelper.accessor('status.completionTime', {
        header: 'Completed',
        cell: ({ getValue }) => {
          const completionTime = getValue()
          if (!completionTime) return '-'

          const dateStr = formatDate(completionTime)

          return (
            <span className="text-muted-foreground text-sm">{dateStr}</span>
          )
        },
      }),
      columnHelper.accessor('metadata.creationTimestamp', {
        header: 'Created',
        cell: ({ getValue }) => {
          const dateStr = formatDate(getValue() || '')

          return (
            <span className="text-muted-foreground text-sm">{dateStr}</span>
          )
        },
      }),
      columnHelper.display({
        id: 'logs',
        header: t('common.fields.logs'),
        cell: ({ row }) => {
          const selector = Object.entries(
            row.original.spec?.selector?.matchLabels ?? {}
          )
            .map(([key, value]) => `${key}=${value}`)
            .join(',')
          return (
            <Link
              to={`/logs?namespace=${row.original.metadata!.namespace}&selector=${encodeURIComponent(selector)}`}
              className="text-muted-foreground text-sm hover:text-foreground"
              title={t('logCenter.linkTitle')}
            >
              {t('common.fields.logs')}
            </Link>
          )
        },
      }),
    ],
    [t]
  )

  return (
    <ResourceTable
      enableYamlExport
      resourceName="Jobs"
      columns={columns}
      searchQueryFilter={jobSearchFilter}
    />
  )
}
