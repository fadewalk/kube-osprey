import { useMemo, useState } from 'react'
import { IconArrowsVertical, IconBox, IconReload } from '@tabler/icons-react'
import { useQueryClient } from '@tanstack/react-query'
import { createColumnHelper } from '@tanstack/react-table'
import { Deployment } from 'kubernetes-types/apps/v1'
import { useTranslation } from 'react-i18next'
import { Link, useNavigate } from 'react-router-dom'

import { patchResource, restartWorkload } from '@/lib/api'
import { updateWorkloadImage } from '@/lib/api/workload-ops'
import { createSearchFilter, getDeploymentStatus } from '@/lib/k8s'
import { formatDate } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { DeploymentStatusIcon } from '@/components/deployment-status-icon'
import { DeploymentCreateDialog } from '@/components/editors/deployment-create-dialog'
import { ResourceBatchActionDialog } from '@/components/resource-batch-action-dialog'
import {
  ResourceTable,
  type ResourceTableBatchAction,
} from '@/components/resource-table'

const deploymentSearchFilter = createSearchFilter<Deployment>(
  (d) => d.metadata?.name,
  (d) => d.metadata?.namespace
)

const columnHelper = createColumnHelper<Deployment>()

export function DeploymentListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false)
  const [isRestartDialogOpen, setIsRestartDialogOpen] = useState(false)
  const [isScaleDialogOpen, setIsScaleDialogOpen] = useState(false)
  const [isImageDialogOpen, setIsImageDialogOpen] = useState(false)
  const [batchDeployments, setBatchDeployments] = useState<Deployment[]>([])
  const [scaleReplicas, setScaleReplicas] = useState('1')
  const [batchImage, setBatchImage] = useState('')
  const [batchImageContainer, setBatchImageContainer] = useState('')

  const replicas = Number(scaleReplicas)
  const scaleValid = Number.isInteger(replicas) && replicas >= 0

  const batchActions = useMemo<ResourceTableBatchAction<Deployment>[]>(
    () => [
      {
        id: 'restart',
        label: t('common.actions.restart'),
        icon: <IconReload className="size-4" />,
        onSelect: (deployments) => {
          setBatchDeployments(deployments)
          setIsRestartDialogOpen(true)
        },
      },
      {
        id: 'scale',
        label: t('common.actions.scale'),
        icon: <IconArrowsVertical className="size-4" />,
        onSelect: (deployments) => {
          setBatchDeployments(deployments)
          setScaleReplicas('1')
          setIsScaleDialogOpen(true)
        },
      },
      {
        id: 'image',
        label: t('resourceTable.batchActions.updateImage'),
        icon: <IconBox className="size-4" />,
        onSelect: (deployments) => {
          setBatchDeployments(deployments)
          setBatchImage('')
          setBatchImageContainer('')
          setIsImageDialogOpen(true)
        },
      },
    ],
    [t]
  )

  // Define columns for the deployment table
  const columns = useMemo(
    () => [
      columnHelper.accessor('metadata.name', {
        header: t('common.fields.name'),
        cell: ({ row }) => (
          <div className="font-medium app-link">
            <Link
              to={`/deployments/${row.original.metadata!.namespace}/${
                row.original.metadata!.name
              }`}
            >
              {row.original.metadata!.name}
            </Link>
          </div>
        ),
      }),
      columnHelper.accessor((row) => row.status?.readyReplicas ?? 0, {
        id: 'ready',
        header: t('common.fields.ready'),
        cell: ({ row }) => {
          const status = row.original.status
          const ready = status?.readyReplicas || 0
          const desired = status?.replicas || 0
          return (
            <div>
              {ready} / {desired}
            </div>
          )
        },
      }),
      columnHelper.accessor((row) => getDeploymentStatus(row), {
        id: 'status_conditions',
        header: t('common.fields.status'),
        cell: ({ getValue }) => {
          const status = getValue()
          return (
            <Badge variant="outline" className="text-muted-foreground px-1.5">
              <DeploymentStatusIcon status={status} />
              {status}
            </Badge>
          )
        },
      }),
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

  const handleCreateClick = () => {
    setIsCreateDialogOpen(true)
  }

  const handleCreateSuccess = (deployment: Deployment, namespace: string) => {
    // Navigate to the newly created deployment's detail page
    navigate(`/deployments/${namespace}/${deployment.metadata?.name}`)
  }

  return (
    <>
      <ResourceTable
        enableYamlExport
        resourceName="Deployments"
        columns={columns}
        searchQueryFilter={deploymentSearchFilter}
        showCreateButton={true}
        onCreateClick={handleCreateClick}
        batchActions={batchActions}
      />

      <DeploymentCreateDialog
        open={isCreateDialogOpen}
        onOpenChange={setIsCreateDialogOpen}
        onSuccess={handleCreateSuccess}
      />

      <ResourceBatchActionDialog
        open={isRestartDialogOpen}
        onOpenChange={setIsRestartDialogOpen}
        resources={batchDeployments}
        title={t('resourceTable.batchActions.restartResourcesTitle', {
          count: batchDeployments.length,
          resource: t('nav.deployments'),
        })}
        description={t(
          'resourceTable.batchActions.restartResourcesDescription',
          { resource: t('nav.deployments') }
        )}
        actionLabel={t('common.actions.restart')}
        onExecute={(deployment) =>
          restartWorkload(
            'deployments',
            deployment.metadata!.name!,
            deployment.metadata!.namespace!
          )
        }
        onComplete={() =>
          queryClient.invalidateQueries({ queryKey: ['deployments'] })
        }
        destructive={true}
      />

      <ResourceBatchActionDialog
        open={isScaleDialogOpen}
        onOpenChange={setIsScaleDialogOpen}
        resources={batchDeployments}
        title={t('resourceTable.batchActions.scaleResourcesTitle', {
          count: batchDeployments.length,
          resource: t('nav.deployments'),
        })}
        description={t('resourceTable.batchActions.scaleResourcesDescription', {
          resource: t('nav.deployments'),
        })}
        actionLabel={t('common.actions.scale')}
        executeDisabled={!scaleValid}
        options={(disabled) => (
          <div className="space-y-1">
            <Label htmlFor="batch-scale-replicas">
              {t('resourceTable.batchActions.scaleReplicasLabel')}
            </Label>
            <Input
              id="batch-scale-replicas"
              type="number"
              min={0}
              value={scaleReplicas}
              disabled={disabled}
              onChange={(e) => setScaleReplicas(e.target.value)}
            />
          </div>
        )}
        onExecute={(deployment) =>
          patchResource(
            'deployments',
            deployment.metadata!.name!,
            deployment.metadata!.namespace!,
            {
              spec: { replicas },
            }
          )
        }
        onComplete={() =>
          queryClient.invalidateQueries({ queryKey: ['deployments'] })
        }
      />

      <ResourceBatchActionDialog
        open={isImageDialogOpen}
        onOpenChange={setIsImageDialogOpen}
        resources={batchDeployments}
        title={t('resourceTable.batchActions.imageResourcesTitle', {
          count: batchDeployments.length,
          resource: t('nav.deployments'),
        })}
        description={t('resourceTable.batchActions.imageResourcesDescription', {
          resource: t('nav.deployments'),
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
        onExecute={(deployment) =>
          updateWorkloadImage({
            kind: 'deployments',
            namespace: deployment.metadata!.namespace!,
            name: deployment.metadata!.name!,
            container: batchImageContainer.trim() || undefined,
            image: batchImage.trim(),
          })
        }
        onComplete={() =>
          queryClient.invalidateQueries({ queryKey: ['deployments'] })
        }
      />
    </>
  )
}
