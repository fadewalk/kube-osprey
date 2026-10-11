import { useCallback, useEffect, useMemo, useState } from 'react'
import { IconAlertCircle, IconEye } from '@tabler/icons-react'
import {
  ColumnDef,
  getCoreRowModel,
  PaginationState,
  useReactTable,
} from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'

import { ResourceHistory } from '@/types/api'
import {
  useAuditLogs,
  useClusterList,
  useOperationLogs,
  useUserList,
  type OperationLogEntry,
} from '@/lib/api'
import { formatDate } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ResourceTableView } from '@/components/resource-table-view'
import { YamlDiffViewer } from '@/components/yaml-diff-viewer'

function ResourceChangesAudit() {
  const { t } = useTranslation()
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [operatorId, setOperatorId] = useState<number | undefined>(undefined)
  const [searchQuery, setSearchQuery] = useState('')
  const [operationFilter, setOperationFilter] = useState('')
  const [clusterFilter, setClusterFilter] = useState('')
  const [selectedHistory, setSelectedHistory] =
    useState<ResourceHistory | null>(null)
  const [isDiffOpen, setIsDiffOpen] = useState(false)
  const [isErrorDialogOpen, setIsErrorDialogOpen] = useState(false)

  const { data: usersData } = useUserList(1, 200)
  const { data: clusters = [] } = useClusterList()
  const showCluster = clusters.length > 1
  const {
    data: auditData,
    isLoading,
    error,
  } = useAuditLogs(
    pagination.pageIndex + 1,
    pagination.pageSize,
    operatorId,
    searchQuery,
    operationFilter || undefined,
    showCluster ? clusterFilter || undefined : undefined
  )

  useEffect(() => {
    if (!showCluster && clusterFilter) {
      setClusterFilter('')
    }
  }, [clusterFilter, showCluster])

  const handleUserFilterChange = useCallback((value: string) => {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
    if (value === 'all') {
      setOperatorId(undefined)
      return
    }
    const parsed = Number(value)
    setOperatorId(Number.isNaN(parsed) ? undefined : parsed)
  }, [])

  const handleSearchChange = useCallback((value: string) => {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
    setSearchQuery(value)
  }, [])

  const handleOperationChange = useCallback((value: string) => {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
    setOperationFilter(value === 'all' ? '' : value)
  }, [])

  const handleClusterChange = useCallback((value: string) => {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
    setClusterFilter(value === 'all' ? '' : value)
  }, [])

  const getOperationTypeColor = (operationType: string) => {
    switch (operationType.toLowerCase()) {
      case 'create':
        return 'default'
      case 'update':
        return 'secondary'
      case 'delete':
        return 'destructive'
      case 'apply':
        return 'outline'
      default:
        return 'secondary'
    }
  }

  const getOperationTypeLabel = useCallback(
    (operationType: string) => {
      switch (operationType.toLowerCase()) {
        case 'create':
          return t('common.actions.create')
        case 'update':
          return t('common.actions.update')
        case 'delete':
          return t('common.actions.delete')
        case 'apply':
          return t('common.actions.apply')
        default:
          return operationType
      }
    },
    [t]
  )

  const columns = useMemo<ColumnDef<ResourceHistory>[]>(
    () => [
      {
        id: 'time',
        header: t('common.fields.time', 'Time'),
        cell: ({ row }) => (
          <span className="text-muted-foreground text-sm">
            {formatDate(row.original.createdAt)}
          </span>
        ),
      },
      {
        id: 'operator',
        header: t('common.fields.operator', 'Operator'),
        cell: ({ row }) => (
          <div className="font-medium">
            {row.original.operator?.username || '-'}
            {row.original.operator?.provider === 'api_key' && (
              <span className="ml-2 text-xs text-muted-foreground italic">
                apikey
              </span>
            )}
            {row.original.operationSource === 'ai' && (
              <span className="ml-2 text-xs text-muted-foreground italic">
                AI
              </span>
            )}
          </div>
        ),
      },
      {
        id: 'operationType',
        header: t('common.fields.operation', 'Operation'),
        cell: ({ row }) => (
          <Badge variant={getOperationTypeColor(row.original.operationType)}>
            {getOperationTypeLabel(row.original.operationType)}
          </Badge>
        ),
      },
      {
        id: 'resource',
        header: t('common.fields.resource', 'Resource'),
        cell: ({ row }) => {
          const resource = row.original
          const name = resource.namespace
            ? `${resource.namespace}/${resource.resourceName}`
            : resource.resourceName
          return (
            <div className="text-sm">
              <div className="font-medium">{name || '-'}</div>
              <div className="text-muted-foreground text-xs">
                {resource.resourceType || '-'}
              </div>
            </div>
          )
        },
      },
      ...(showCluster
        ? [
            {
              id: 'cluster',
              header: t('common.fields.cluster', 'Cluster'),
              cell: ({ row }: { row: { original: ResourceHistory } }) => (
                <span className="text-sm text-muted-foreground">
                  {row.original.clusterName || '-'}
                </span>
              ),
            },
          ]
        : []),
      {
        id: 'status',
        header: t('common.fields.status', 'Status'),
        cell: ({ row }) => (
          <Badge variant={row.original.success ? 'default' : 'destructive'}>
            {row.original.success
              ? t('status.success', 'Success')
              : t('status.failed', 'Failed')}
          </Badge>
        ),
      },
      {
        id: 'actions',
        header: t('common.fields.actions', 'Actions'),
        cell: ({ row }) => {
          const item = row.original
          if (!item.success) {
            return (
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  setSelectedHistory(item)
                  setIsErrorDialogOpen(true)
                }}
                disabled={!item.errorMessage}
              >
                <IconAlertCircle className="w-4 h-4 mr-1" />
                {t('common.actions.viewError', 'View Error')}
              </Button>
            )
          }
          return (
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setSelectedHistory(item)
                setIsDiffOpen(true)
              }}
              disabled={!item.resourceYaml && !item.previousYaml}
            >
              <IconEye className="w-4 h-4 mr-1" />
              {t('common.actions.viewDiff', 'View Diff')}
            </Button>
          )
        },
      },
    ],
    [getOperationTypeLabel, showCluster, t]
  )

  const table = useReactTable({
    data: auditData?.data ?? [],
    columns,
    getCoreRowModel: getCoreRowModel(),
    state: { pagination },
    onPaginationChange: setPagination,
    manualPagination: true,
    pageCount: Math.ceil((auditData?.total ?? 0) / pagination.pageSize) || 0,
  })

  const emptyState = (() => {
    if (isLoading) {
      return (
        <div className="py-10 text-center text-muted-foreground">
          {t('common.messages.loadingResource', {
            resource: t('common.fields.auditLogs', 'audit logs'),
            defaultValue: 'Loading audit logs...',
          })}
        </div>
      )
    }
    if (error) {
      return (
        <div className="py-10 text-center text-destructive">
          {t('common.messages.failedToLoad', {
            resource: t('common.fields.auditLogs', 'audit logs'),
            defaultValue: 'Failed to load audit logs',
          })}
        </div>
      )
    }
    if ((auditData?.data.length ?? 0) === 0) {
      return (
        <div className="py-10 text-center text-muted-foreground">
          {t('common.messages.noItemsFound', {
            resource: t('common.fields.auditLogs', 'audit logs'),
            defaultValue: 'No audit logs found',
          })}
        </div>
      )
    }
    return null
  })()

  const totalRowCount = auditData?.total ?? 0
  const filteredRowCount = auditData?.data.length ?? 0

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>{t('common.fields.auditLogs', 'Audit Logs')}</CardTitle>
            <p className="text-muted-foreground text-sm">
              {t(
                'common.messages.auditLogsDescription',
                'Track who changed resources and review YAML diffs'
              )}
            </p>
          </div>
          <div className="flex items-center gap-3">
            <Input
              placeholder={t(
                'common.placeholders.searchResourceName',
                'Search resource name...'
              )}
              value={searchQuery}
              onChange={(event) => handleSearchChange(event.target.value)}
              className="w-64"
            />
            <Select
              value={operationFilter || 'all'}
              onValueChange={handleOperationChange}
            >
              <SelectTrigger className="w-44">
                <SelectValue
                  placeholder={t(
                    'common.values.allOperations',
                    'All operations'
                  )}
                />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">
                  {t('common.values.allOperations', 'All operations')}
                </SelectItem>
                <SelectItem value="create">
                  {t('common.actions.create')}
                </SelectItem>
                <SelectItem value="update">
                  {t('common.actions.update')}
                </SelectItem>
                <SelectItem value="delete">
                  {t('common.actions.delete')}
                </SelectItem>
                <SelectItem value="apply">
                  {t('common.actions.apply')}
                </SelectItem>
                <SelectItem value="patch">
                  {t('common.actions.patch')}
                </SelectItem>
              </SelectContent>
            </Select>
            {showCluster && (
              <Select
                value={clusterFilter || 'all'}
                onValueChange={handleClusterChange}
              >
                <SelectTrigger className="w-56">
                  <SelectValue
                    placeholder={t('common.values.allClusters', 'All clusters')}
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">
                    {t('common.values.allClusters', 'All clusters')}
                  </SelectItem>
                  {clusters.map((cluster) => (
                    <SelectItem key={cluster.name} value={cluster.name}>
                      {cluster.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
            <Select
              value={operatorId ? String(operatorId) : 'all'}
              onValueChange={handleUserFilterChange}
            >
              <SelectTrigger className="w-56">
                <SelectValue
                  placeholder={t('common.values.allUsers', 'All users')}
                />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">
                  {t('common.values.allUsers', 'All users')}
                </SelectItem>
                {(usersData?.users ?? []).map((user) => (
                  <SelectItem key={user.id} value={String(user.id)}>
                    {user.username}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <ResourceTableView
          table={table}
          columnCount={columns.length}
          isLoading={isLoading}
          data={auditData?.data}
          allPageSize={totalRowCount}
          emptyState={emptyState}
          hasActiveFilters={
            Boolean(operatorId) ||
            Boolean(searchQuery) ||
            Boolean(operationFilter) ||
            (showCluster && Boolean(clusterFilter))
          }
          filteredRowCount={filteredRowCount}
          totalRowCount={totalRowCount}
          searchQuery={searchQuery}
          pagination={pagination}
          setPagination={setPagination}
          maxBodyHeightClassName="max-h-[600px]"
        />
      </CardContent>

      {selectedHistory && (
        <YamlDiffViewer
          open={isDiffOpen}
          onOpenChange={(open) => {
            setIsDiffOpen(open)
            if (!open) {
              setSelectedHistory(null)
            }
          }}
          original={selectedHistory.previousYaml || ''}
          modified={selectedHistory.resourceYaml || ''}
          title={t('common.fields.yamlDiff', 'YAML Diff')}
          height={560}
        />
      )}

      <Dialog
        open={isErrorDialogOpen}
        onOpenChange={(open) => {
          setIsErrorDialogOpen(open)
          if (!open) {
            setSelectedHistory(null)
          }
        }}
      >
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>
              {t('common.fields.errorDetails', 'Error Details')}
            </DialogTitle>
          </DialogHeader>
          <div className="text-sm text-muted-foreground whitespace-pre-wrap">
            {selectedHistory?.errorMessage ||
              t('common.messages.noErrorMessage', 'No error message')}
          </div>
        </DialogContent>
      </Dialog>
    </Card>
  )
}

function OperationLogsAudit() {
  const { t } = useTranslation()
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [usernameFilter, setUsernameFilter] = useState('')
  const [moduleFilter, setModuleFilter] = useState('')
  const { data, isLoading, error } = useOperationLogs(
    pagination.pageIndex + 1,
    pagination.pageSize,
    usernameFilter || undefined,
    moduleFilter || undefined
  )
  const [selectedEntry, setSelectedEntry] = useState<OperationLogEntry | null>(
    null
  )

  const columns = useMemo<ColumnDef<OperationLogEntry>[]>(
    () => [
      {
        accessorKey: 'createdAt',
        header: t('common.fields.created'),
        cell: ({ getValue }) => (
          <span className="whitespace-nowrap text-xs">
            {formatDate(String(getValue() || ''))}
          </span>
        ),
      },
      {
        accessorKey: 'username',
        header: t('audit.operationLogs.operator'),
        cell: ({ getValue }) => (
          <span className="font-medium">{String(getValue() || '-')}</span>
        ),
      },
      {
        accessorKey: 'module',
        header: t('audit.operationLogs.module'),
        cell: ({ row }) => (
          <div className="flex items-center gap-2">
            <Badge
              variant={
                row.original.statusCode >= 400 ? 'destructive' : 'secondary'
              }
            >
              {row.original.method}
            </Badge>
            <span className="text-sm">
              {t(`audit.operationLogs.modules.${row.original.module}`, {
                defaultValue: row.original.module,
              })}
            </span>
          </div>
        ),
      },
      {
        accessorKey: 'action',
        header: t('audit.operationLogs.operation'),
        cell: ({ row }) => (
          <div className="text-sm">
            <span>{row.original.action}</span>
            {row.original.resource ? (
              <span className="ml-1 text-xs text-muted-foreground">
                {t(`audit.operationLogs.resources.${row.original.resource}`, {
                  defaultValue: row.original.resource,
                })}
                {row.original.resourceName
                  ? ` / ${row.original.resourceName}`
                  : ''}
              </span>
            ) : null}
          </div>
        ),
      },
      {
        accessorKey: 'cluster',
        header: t('common.fields.cluster'),
        cell: ({ getValue }) => (
          <span className="font-mono text-xs">{String(getValue() || '-')}</span>
        ),
      },
      {
        accessorKey: 'statusCode',
        header: t('audit.operationLogs.status'),
        cell: ({ getValue }) => (
          <span
            className={`font-mono text-xs ${
              Number(getValue()) >= 400 ? 'text-destructive' : ''
            }`}
          >
            {Number(getValue())}
          </span>
        ),
      },
      {
        accessorKey: 'durationMs',
        header: t('audit.operationLogs.duration'),
        cell: ({ getValue }) => (
          <span className="whitespace-nowrap text-xs text-muted-foreground">
            {Number(getValue())} ms
          </span>
        ),
      },
      {
        id: 'body',
        header: t('audit.operationLogs.body'),
        cell: ({ row }) =>
          row.original.body ? (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setSelectedEntry(row.original)}
            >
              <IconEye className="h-4 w-4" />
            </Button>
          ) : null,
      },
    ],
    [t]
  )

  const table = useReactTable({
    data: data?.data ?? [],
    columns,
    getCoreRowModel: getCoreRowModel(),
    state: { pagination },
    onPaginationChange: setPagination,
    manualPagination: true,
    pageCount: Math.ceil((data?.total ?? 0) / pagination.pageSize) || 0,
  })

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <CardTitle>{t('audit.operationLogs.title')}</CardTitle>
            <p className="text-muted-foreground text-sm">
              {t('audit.operationLogs.description')}
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Input
              placeholder={t('audit.operationLogs.usernameFilter')}
              value={usernameFilter}
              onChange={(event) => {
                setPagination((prev) => ({ ...prev, pageIndex: 0 }))
                setUsernameFilter(event.target.value)
              }}
              className="w-40"
            />
            <Select
              value={moduleFilter || 'all'}
              onValueChange={(value) => {
                setPagination((prev) => ({ ...prev, pageIndex: 0 }))
                setModuleFilter(value === 'all' ? '' : value)
              }}
            >
              <SelectTrigger className="w-44">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">
                  {t('audit.operationLogs.allModules')}
                </SelectItem>
                {[
                  'clusters',
                  'roles',
                  'users',
                  'settings',
                  'resources',
                  'workloads',
                  'changeplans',
                  'alertmanager',
                  'plugins',
                  'account',
                ].map((moduleName) => (
                  <SelectItem key={moduleName} value={moduleName}>
                    {t(`audit.operationLogs.modules.${moduleName}`, {
                      defaultValue: moduleName,
                    })}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <ResourceTableView
          table={table}
          columnCount={columns.length}
          isLoading={isLoading}
          data={data?.data}
          allPageSize={data?.total ?? 0}
          emptyState={
            error ? (
              <div className="py-10 text-center text-destructive">
                {error.message}
              </div>
            ) : (
              <div className="py-10 text-center text-muted-foreground">
                {t('audit.operationLogs.empty')}
              </div>
            )
          }
          hasActiveFilters={Boolean(usernameFilter) || Boolean(moduleFilter)}
          filteredRowCount={data?.data.length ?? 0}
          totalRowCount={data?.total ?? 0}
          searchQuery={usernameFilter}
          pagination={pagination}
          setPagination={setPagination}
          maxBodyHeightClassName="max-h-[600px]"
        />
      </CardContent>
      <Dialog
        open={!!selectedEntry}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedEntry(null)
          }
        }}
      >
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>{t('audit.operationLogs.bodyTitle')}</DialogTitle>
          </DialogHeader>
          <div className="max-h-96 overflow-auto whitespace-pre-wrap break-all rounded-md bg-muted p-3 font-mono text-xs">
            {selectedEntry?.body || t('audit.operationLogs.noBody')}
          </div>
        </DialogContent>
      </Dialog>
    </Card>
  )
}

export function AuditLog() {
  const { t } = useTranslation()
  const [view, setView] = useState<'resource-changes' | 'operation-logs'>(
    'resource-changes'
  )

  return (
    <div className="flex flex-col gap-4">
      <Tabs
        value={view}
        onValueChange={(value) => setView(value as typeof view)}
      >
        <TabsList>
          <TabsTrigger value="resource-changes">
            {t('audit.operationLogs.resourceChangesTab')}
          </TabsTrigger>
          <TabsTrigger value="operation-logs">
            {t('audit.operationLogs.tab')}
          </TabsTrigger>
        </TabsList>
      </Tabs>
      {view === 'resource-changes' ? (
        <ResourceChangesAudit />
      ) : (
        <OperationLogsAudit />
      )}
    </div>
  )
}
