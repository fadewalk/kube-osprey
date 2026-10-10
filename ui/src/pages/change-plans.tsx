import { useMemo, useState } from 'react'
import {
  IconArrowBackUp,
  IconCircleCheck,
  IconCircleX,
  IconClock,
  IconFlask,
  IconRefresh,
} from '@tabler/icons-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  approveChangePlan,
  ChangePlan,
  ChangePlanStep,
  getChangePlan,
  listChangePlans,
  rejectChangePlan,
  rollbackChangePlan,
} from '@/lib/api/changeplans'
import { MonacoDiffEditor } from '@/lib/monaco-loader'
import { translateError } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useAppearance } from '@/components/appearance-provider'

const STATUS_FILTERS = [
  'PENDING',
  'APPROVED',
  'APPLIED',
  'FAILED',
  'REJECTED',
  'EXPIRED',
  'CREATED',
] as const

const PLAN_STATUS_STYLE: Record<string, string> = {
  PENDING: 'bg-amber-500/15 text-amber-600 dark:text-amber-400',
  APPROVED: 'bg-blue-500/15 text-blue-600 dark:text-blue-400',
  APPLYING: 'bg-blue-500/15 text-blue-600 dark:text-blue-400',
  APPLIED: 'bg-green-500/15 text-green-600 dark:text-green-400',
  FAILED: 'bg-red-500/15 text-red-600 dark:text-red-400',
  REJECTED: 'bg-zinc-500/15 text-zinc-600 dark:text-zinc-400',
  EXPIRED: 'bg-zinc-500/15 text-zinc-600 dark:text-zinc-400',
  CREATED: 'bg-zinc-500/15 text-zinc-600 dark:text-zinc-400',
}

const STEP_STATUS_STYLE: Record<string, string> = {
  APPLIED: 'bg-green-500/15 text-green-600 dark:text-green-400',
  DRY_RUN_OK: 'bg-green-500/15 text-green-600 dark:text-green-400',
  DRY_RUN_FAILED: 'bg-red-500/15 text-red-600 dark:text-red-400',
  FAILED: 'bg-red-500/15 text-red-600 dark:text-red-400',
  SKIPPED: 'bg-zinc-500/15 text-zinc-600 dark:text-zinc-400',
  DRY_RUN_UNSUPPORTED: 'bg-zinc-500/15 text-zinc-600 dark:text-zinc-400',
  PENDING: 'bg-amber-500/15 text-amber-600 dark:text-amber-400',
}

function DangerBadge({ level }: { level: string }) {
  const { t } = useTranslation()
  const style =
    level === 'HIGH'
      ? 'bg-red-500/15 text-red-600 dark:text-red-400'
      : level === 'MEDIUM'
        ? 'bg-amber-500/15 text-amber-600 dark:text-amber-400'
        : 'bg-zinc-500/15 text-zinc-600 dark:text-zinc-400'
  return (
    <Badge variant="outline" className={style}>
      {t(`changePlans.danger.${level}`, level)}
    </Badge>
  )
}

function StatusBadge({ status }: { status: string }) {
  const style = PLAN_STATUS_STYLE[status] ?? STEP_STATUS_STYLE[status] ?? ''
  return (
    <Badge variant="outline" className={style}>
      {status}
    </Badge>
  )
}

function StepDiff({ step, theme }: { step: ChangePlanStep; theme: string }) {
  const { t } = useTranslation()
  const pretty = (raw: string, fallback: string) => {
    if (!raw) return fallback
    try {
      return JSON.stringify(JSON.parse(raw), null, 2)
    } catch {
      return raw
    }
  }
  const original = pretty(step.beforeJson, '')
  const modified = pretty(step.afterJson, '')

  if (!step.dryRunSupported) {
    return (
      <div className="rounded-md border bg-muted/40 p-3 text-sm text-muted-foreground">
        {t(
          'changePlans.detail.noDryRun',
          'This operation has no server-side dry-run; only the current object snapshot is recorded.'
        )}
        {original && (
          <pre className="mt-2 max-h-48 overflow-auto rounded bg-background p-2 font-mono text-xs">
            {original}
          </pre>
        )}
      </div>
    )
  }
  if (step.dryRunStatus === 'DRY_RUN_FAILED') {
    return (
      <div className="rounded-md border border-red-500/40 bg-red-500/5 p-3 text-sm text-red-600 dark:text-red-400">
        {step.dryRunResponse || step.errorMessage}
      </div>
    )
  }
  if (!original && !modified) {
    return null
  }
  return (
    <MonacoDiffEditor
      key={`step-${step.id}-${theme}`}
      height={260}
      language="json"
      theme={theme}
      original={original || t('changePlans.detail.notExists', '(not present)')}
      modified={modified || t('changePlans.detail.deleted', '(deleted)')}
      options={{
        readOnly: true,
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        renderSideBySide: true,
        fontSize: 12,
      }}
    />
  )
}

function PlanDetail({
  planId,
  canApprove,
  onActionDone,
}: {
  planId: number
  canApprove: boolean
  onActionDone: () => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { actualTheme } = useAppearance()
  const theme = actualTheme === 'dark' ? 'vs-dark' : 'vs'
  const [openStep, setOpenStep] = useState<number | null>(null)

  const { data: plan, isLoading } = useQuery({
    queryKey: ['change-plans', planId],
    queryFn: () => getChangePlan(planId),
    enabled: planId > 0,
  })

  const act = useMutation({
    mutationFn: async (action: 'approve' | 'reject' | 'rollback') => {
      if (action === 'approve') {
        await approveChangePlan(planId)
      } else if (action === 'reject') {
        await rejectChangePlan(planId)
      } else {
        await rollbackChangePlan(planId)
      }
    },
    onSuccess: (_, action) => {
      const key =
        action === 'approve'
          ? 'approved'
          : action === 'reject'
            ? 'rejected'
            : 'rolledBack'
      toast.success(t(`changePlans.messages.${key}`))
      queryClient.invalidateQueries({ queryKey: ['change-plans'] })
      onActionDone()
    },
    onError: (error) => {
      toast.error(translateError(error, t))
    },
  })

  if (isLoading || !plan) {
    return <div className="p-4 text-sm text-muted-foreground">Loading...</div>
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2 text-sm">
        <StatusBadge status={plan.status} />
        <DangerBadge level={plan.dangerLevel} />
        <span className="text-muted-foreground">
          {t('changePlans.detail.creator', { name: plan.username })}
        </span>
        {plan.approverName && (
          <span className="text-muted-foreground">
            {t('changePlans.detail.approver', { name: plan.approverName })}
          </span>
        )}
        {plan.expiresAt && plan.status === 'PENDING' && (
          <span className="text-muted-foreground">
            {t('changePlans.detail.expires', {
              time: new Date(plan.expiresAt).toLocaleString(),
            })}
          </span>
        )}
      </div>

      {plan.description && (
        <p className="rounded-md border bg-muted/40 p-3 text-sm">
          {plan.description}
        </p>
      )}

      {plan.status === 'PENDING' && canApprove && (
        <div className="flex gap-2">
          <Button
            onClick={() => act.mutate('approve')}
            disabled={act.isPending}
            className="gap-1"
          >
            <IconCircleCheck className="size-4" />
            {t('changePlans.actions.approve')}
          </Button>
          <Button
            variant="destructive"
            onClick={() => act.mutate('reject')}
            disabled={act.isPending}
            className="gap-1"
          >
            <IconCircleX className="size-4" />
            {t('changePlans.actions.reject')}
          </Button>
        </div>
      )}

      {plan.status === 'APPLIED' && canApprove && (
        <div className="flex gap-2">
          <Button
            variant="outline"
            onClick={() => act.mutate('rollback')}
            disabled={act.isPending}
            className="gap-1"
          >
            <IconArrowBackUp className="size-4" />
            {t('changePlans.actions.rollback')}
          </Button>
        </div>
      )}

      <div className="space-y-2">
        {plan.steps.map((step) => (
          <Collapsible
            key={step.id}
            open={openStep === step.id}
            onOpenChange={(open) => setOpenStep(open ? step.id : null)}
          >
            <div className="rounded-lg border">
              <CollapsibleTrigger asChild>
                <button className="flex w-full items-center gap-2 p-3 text-left text-sm hover:bg-muted/40">
                  <span className="font-mono text-xs text-muted-foreground">
                    #{step.seq}
                  </span>
                  <span className="font-medium">{step.toolName}</span>
                  <span className="text-muted-foreground">
                    {step.cluster}
                    {step.namespace ? `/${step.namespace}` : ''}
                    {step.resource ? ` · ${step.resource}` : ''}
                  </span>
                  <span className="ml-auto flex items-center gap-2">
                    <DangerBadge level={step.dangerLevel} />
                    <StatusBadge status={step.status} />
                  </span>
                </button>
              </CollapsibleTrigger>
              {step.description && (
                <p className="px-3 pb-2 text-xs text-muted-foreground">
                  {step.description}
                </p>
              )}
              <CollapsibleContent>
                <div className="border-t p-3">
                  {step.errorMessage && (
                    <p className="mb-2 text-sm text-red-600 dark:text-red-400">
                      {step.errorMessage}
                    </p>
                  )}
                  {step.applyResponse && (
                    <p className="mb-2 font-mono text-xs text-muted-foreground">
                      {step.applyResponse}
                    </p>
                  )}
                  <StepDiff step={step} theme={theme} />
                </div>
              </CollapsibleContent>
            </div>
          </Collapsible>
        ))}
      </div>
    </div>
  )
}

export function ChangePlansPage() {
  const { t } = useTranslation()
  const [statusFilter, setStatusFilter] = useState<string>('PENDING')
  const [selectedId, setSelectedId] = useState<number | null>(null)

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['change-plans', 'list', statusFilter],
    queryFn: () =>
      listChangePlans(statusFilter === 'ALL' ? undefined : statusFilter),
    refetchInterval: 15000,
  })

  const plans = useMemo(() => data?.plans ?? [], [data])
  const canApprove = data?.canApprove ?? false

  const pendingCount = plans.filter((p) => p.status === 'PENDING').length

  return (
    <div className="space-y-4 p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <h1 className="text-lg font-bold">
            {t('changePlans.title', 'Change Plans')}
          </h1>
          {statusFilter === 'PENDING' && pendingCount > 0 && (
            <Badge variant="outline" className="bg-amber-500/15 text-amber-600">
              {pendingCount}
            </Badge>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Select value={statusFilter} onValueChange={setStatusFilter}>
            <SelectTrigger className="w-44">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ALL">{t('changePlans.filter.all')}</SelectItem>
              {STATUS_FILTERS.map((status) => (
                <SelectItem key={status} value={status}>
                  {status}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button variant="outline" size="icon" onClick={() => refetch()}>
            <IconRefresh className="size-4" />
          </Button>
        </div>
      </div>

      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('changePlans.table.plan')}</TableHead>
              <TableHead>{t('changePlans.table.creator')}</TableHead>
              <TableHead>{t('changePlans.table.danger')}</TableHead>
              <TableHead>{t('changePlans.table.steps')}</TableHead>
              <TableHead>{t('changePlans.table.status')}</TableHead>
              <TableHead>{t('changePlans.table.updated')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading && (
              <TableRow>
                <TableCell
                  colSpan={6}
                  className="text-center text-muted-foreground"
                >
                  Loading...
                </TableCell>
              </TableRow>
            )}
            {!isLoading && plans.length === 0 && (
              <TableRow>
                <TableCell
                  colSpan={6}
                  className="text-center text-muted-foreground"
                >
                  <div className="flex flex-col items-center gap-2 py-6">
                    <IconClock className="size-6 text-muted-foreground" />
                    {t('changePlans.table.empty', 'No change plans')}
                  </div>
                </TableCell>
              </TableRow>
            )}
            {plans.map((plan: ChangePlan) => (
              <TableRow
                key={plan.id}
                className="cursor-pointer"
                onClick={() => setSelectedId(plan.id)}
              >
                <TableCell className="max-w-72">
                  <div className="truncate font-medium">
                    {plan.description || `#${plan.id}`}
                  </div>
                  {plan.clientName && (
                    <div className="text-xs text-muted-foreground">
                      {plan.clientName}
                    </div>
                  )}
                </TableCell>
                <TableCell>{plan.username}</TableCell>
                <TableCell>
                  <DangerBadge level={plan.dangerLevel} />
                </TableCell>
                <TableCell>{plan.steps?.length ?? 0}</TableCell>
                <TableCell>
                  <StatusBadge status={plan.status} />
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {new Date(plan.updatedAt).toLocaleString()}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <Sheet
        open={selectedId !== null}
        onOpenChange={(open) => !open && setSelectedId(null)}
      >
        <SheetContent className="sm:max-w-3xl sm:!max-w-3xl overflow-y-auto">
          <SheetHeader>
            <SheetTitle className="flex items-center gap-2">
              <IconFlask className="size-4" />
              {t('changePlans.detail.title', { id: selectedId })}
            </SheetTitle>
          </SheetHeader>
          {selectedId !== null && (
            <div className="mt-4">
              <PlanDetail
                planId={selectedId}
                canApprove={canApprove}
                onActionDone={() => setSelectedId(null)}
              />
            </div>
          )}
        </SheetContent>
      </Sheet>
    </div>
  )
}
