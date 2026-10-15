import { useMemo, useState } from 'react'
import {
  IconBox,
  IconCircleCheckFilled,
  IconLoader,
  IconReload,
} from '@tabler/icons-react'
import { useQueryClient } from '@tanstack/react-query'
import { createColumnHelper } from '@tanstack/react-table'
import { DaemonSet } from 'kubernetes-types/apps/v1'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'

import { restartWorkload } from '@/lib/api'
import { updateWorkloadImage } from '@/lib/api/workload-ops'
import { createSearchFilter } from '@/lib/k8s'
import { formatDate } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ResourceBatchActionDialog } from '@/components/resource-batch-action-dialog'
import {
  ResourceTable,
  type ResourceTableBatchAction,
} from '@/components/resource-table'

const daemonSetSearchFilter = createSearchFilter<DaemonSet>(
  (d) => d.metadata?.name,
  (d) => d.metadata?.namespace
)

const columnHelper = createColumnHelper<DaemonSet>()

export function DaemonSetListPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [isRestartDialogOpen, setIsRestartDialogOpen] = useState(false)
  const [isImageDialogOpen, setIsImageDialogOpen] = useState(false)
  const [batchDaemonSets, setBatchDaemonSets] = useState<DaemonSet[]>([])
  const [batchImage, setBatchImage] = useState('')
  const [batchImageContainer, setBatchImageContainer] = useState('')

  const batchActions = useMemo<ResourceTableBatchAction<DaemonSet>[]>(
    () => [
      {
        id: 'restart',
        label: t('common.actions.restart'),
        icon: <IconReload className="size-4" />,
        onSelect: (daemonSets) => {
          setBatchDaemonSets(daemonSets)
          setIsRestartDialogOpen(true)
        },
      },
      {
        id: 'image',
        label: t('resourceTable.batchActions.updateImage'),
        icon: <IconBox className="size-4" />,
        onSelect: (daemonSets) => {
          setBatchDaemonSets(daemonSets)
          setBatchImage('')
          setBatchImageContainer('')
          setIsImageDialogOpen(true)
        },
      },
    ],
    [t]
  )

  // Define columns for the daemonset table
  const columns = useMemo(
    () => [
      columnHelper.accessor('metadata.name', {
        header: t('common.fields.name'),
        cell: ({ row }) => (
          <div className="font-medium app-link">
            <Link
              to={`/daemonsets/${row.original.metadata!.namespace}/${
                row.original.metadata!.name
              }`}
            >
              {row.original.metadata!.name}
            </Link>
          </div>
        ),
      }),
      columnHelper.accessor('status.desiredNumberScheduled', {
        header: t('common.fields.desired'),
        cell: ({ getValue }) => getValue() || 0,
      }),
      columnHelper.accessor('status.currentNumberScheduled', {
        header: t('common.fields.current'),
        cell: ({ getValue }) => getValue() || 0,
      }),
      columnHelper.accessor('status.numberReady', {
        header: t('common.fields.ready'),
        cell: ({ getValue }) => getValue() || 0,
      }),
      columnHelper.accessor((row) => row.status?.numberAvailable || 0, {
        id: 'status_numberAvailable',
        header: t('common.fields.available'),
        cell: ({ getValue }) => getValue() || 0,
      }),
      columnHelper.accessor(
        (row) => {
          const readyReplicas = row.status?.numberReady || 0
          const replicas = row.status?.desiredNumberScheduled || 0
          if (replicas === 0) return 'Pending'
          return readyReplicas === replicas
            ? t('common.fields.available')
            : t('common.messages.loading')
        },
        {
          id: 'status_conditions',
          header: t('common.fields.status'),
          cell: ({ row, getValue }) => {
            const readyReplicas = row.original.status?.numberReady || 0
            const replicas = row.original.status?.desiredNumberScheduled || 0
            const isAvailable = readyReplicas === replicas
            const status = getValue()
            if (replicas === 0) {
              return (
                <Badge
                  variant="secondary"
                  className="text-muted-foreground px-1.5"
                >
                  Pending
                </Badge>
              )
            }

            return (
              <Badge variant="outline" className="text-muted-foreground px-1.5">
                {isAvailable ? (
                  <IconCircleCheckFilled className="fill-green-500 dark:fill-green-400" />
                ) : (
                  <IconLoader className="animate-spin" />
                )}
                {status}
              </Badge>
            )
          },
        }
      ),
      columnHelper.accessor('metadata.creationTimestamp', {
        header: t('common.fields.created'),
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
    <>
      <ResourceTable
        enableYamlExport
        resourceName={'DaemonSets'}
        columns={columns}
        searchQueryFilter={daemonSetSearchFilter}
        batchActions={batchActions}
      />

      <ResourceBatchActionDialog
        open={isRestartDialogOpen}
        onOpenChange={setIsRestartDialogOpen}
        resources={batchDaemonSets}
        title={t('resourceTable.batchActions.restartResourcesTitle', {
          count: batchDaemonSets.length,
          resource: t('nav.daemonsets'),
        })}
        description={t(
          'resourceTable.batchActions.restartResourcesDescription',
          { resource: t('nav.daemonsets') }
        )}
        actionLabel={t('common.actions.restart')}
        onExecute={(daemonSet) =>
          restartWorkload(
            'daemonsets',
            daemonSet.metadata!.name!,
            daemonSet.metadata!.namespace!
          )
        }
        onComplete={() =>
          queryClient.invalidateQueries({ queryKey: ['daemonsets'] })
        }
        destructive={true}
      />

      <ResourceBatchActionDialog
        open={isImageDialogOpen}
        onOpenChange={setIsImageDialogOpen}
        resources={batchDaemonSets}
        title={t('resourceTable.batchActions.imageResourcesTitle', {
          count: batchDaemonSets.length,
          resource: t('nav.daemonsets'),
        })}
        description={t('resourceTable.batchActions.imageResourcesDescription', {
          resource: t('nav.daemonsets'),
        })}
        actionLabel={t('resourceTable.batchActions.updateImage')}
        executeDisabled={!batchImage.trim()}
        options={(disabled) => (
          <div className="space-y-3">
            <div className="space-y-1">
              <Label htmlFor="batch-image">
                {t('resourceTable.batchActions.imageLabel')}
              </Label>
              <Input
                id="batch-image"
                value={batchImage}
                disabled={disabled}
                onChange={(e) => setBatchImage(e.target.value)}
                className="font-mono text-xs"
                placeholder="nginx:1.27"
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="batch-image-container">
                {t('resourceTable.batchActions.imageContainerLabel')}
              </Label>
              <Input
                id="batch-image-container"
                value={batchImageContainer}
                disabled={disabled}
                onChange={(e) => setBatchImageContainer(e.target.value)}
                className="font-mono text-xs"
                placeholder={t(
                  'resourceTable.batchActions.imageContainerPlaceholder'
                )}
              />
            </div>
          </div>
        )}
        onExecute={(daemonSet) =>
          updateWorkloadImage({
            kind: 'daemonsets',
            namespace: daemonSet.metadata!.namespace!,
            name: daemonSet.metadata!.name!,
            container: batchImageContainer.trim() || undefined,
            image: batchImage.trim(),
          })
        }
        onComplete={() =>
          queryClient.invalidateQueries({ queryKey: ['daemonsets'] })
        }
      />
    </>
  )
}
