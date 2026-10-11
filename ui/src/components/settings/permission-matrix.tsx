/* eslint-disable react-refresh/only-export-components */
import { useMemo } from 'react'
import { IconBan, IconCheck } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import { resourceCatalog } from '@/lib/resource-catalog'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'

export const RBAC_VERBS = ['get', 'create', 'update', 'delete', 'log', 'exec']

export interface PermissionScope {
  clusters: string[]
  namespaces: string[]
  resources: string[]
  verbs: string[]
}

// Mirrors pkg/rbac match() for plain, "*" and "!negation" entries so the UI
// preview never promises more or less than the backend grants.
function hit(list: string[], value: string): boolean {
  return list.some((p) => p === '*' || p === value)
}

function blocked(list: string[], value: string): boolean {
  return list.some((p) => p.startsWith('!') && p.slice(1) === value)
}

export function cellGranted(
  scope: PermissionScope,
  resource: string,
  verb: string
): boolean {
  if (blocked(scope.resources, resource)) return false
  if (blocked(scope.verbs, verb)) return false
  return hit(scope.resources, resource) && hit(scope.verbs, verb)
}

function resourceLabel(
  type: string,
  t: ReturnType<typeof useTranslation>['t']
) {
  const entry = resourceCatalog.find((r) => r.type === type)
  if (entry) {
    return 'titleKey' in entry
      ? t(entry.titleKey, { defaultValue: entry.pluralLabel })
      : entry.pluralLabel
  }
  return type
}

function scopeRows(scope: PermissionScope): string[] {
  const entries = scope.resources.filter((r) => r !== '*' && !r.startsWith('!'))
  const excluded = scope.resources.filter((r) => r.startsWith('!'))
  const rows: string[] = []
  if (scope.resources.includes('*')) rows.push('*')
  return [...rows, ...entries, ...excluded]
}

function scopeColumns(scope: PermissionScope): string[] {
  const custom = scope.verbs.filter(
    (v) => v !== '*' && !v.startsWith('!') && !RBAC_VERBS.includes(v)
  )
  if (scope.verbs.includes('*')) return [...RBAC_VERBS, ...custom]
  return [...RBAC_VERBS.filter((v) => scope.verbs.includes(v)), ...custom]
}

export function PermissionMatrix({ scope }: { scope: PermissionScope }) {
  const { t } = useTranslation()
  const rows = useMemo(() => scopeRows(scope), [scope])
  const columns = useMemo(() => scopeColumns(scope), [scope])

  if (rows.length === 0 || columns.length === 0) {
    return (
      <p className="text-muted-foreground text-sm">
        {t('rbac.matrix.noGrant', 'No resource permissions granted.')}
      </p>
    )
  }

  return (
    <div className="overflow-x-auto rounded-md border">
      <table className="w-full min-w-max text-sm">
        <thead>
          <tr className="bg-muted/40 border-b">
            <th className="px-3 py-2 text-left font-medium">
              {t('common.fields.resources', 'Resources')}
            </th>
            {columns.map((v) => (
              <th key={v} className="px-3 py-2 text-center font-medium">
                {v}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => {
            const excluded = r.startsWith('!')
            const label = excluded
              ? t('rbac.matrix.except', 'except {{name}}', {
                  name: resourceLabel(r.slice(1), t),
                })
              : r === '*'
                ? t('rbac.matrix.allResources', 'All resources')
                : resourceLabel(r, t)
            return (
              <tr key={r} className="border-b last:border-b-0">
                <td className="px-3 py-1.5">
                  <span
                    className={
                      excluded
                        ? 'text-destructive'
                        : r === '*'
                          ? 'font-medium'
                          : ''
                    }
                  >
                    {label}
                  </span>
                </td>
                {columns.map((v) => {
                  if (excluded) {
                    return (
                      <td key={v} className="px-3 py-1.5 text-center">
                        <IconBan className="text-destructive/60 mx-auto h-4 w-4" />
                      </td>
                    )
                  }
                  const denied = scope.verbs.some(
                    (p) => p.startsWith('!') && p.slice(1) === v
                  )
                  const granted = cellGranted(scope, r, v)
                  return (
                    <td key={v} className="px-3 py-1.5 text-center">
                      {denied ? (
                        <IconBan className="text-destructive/60 mx-auto h-4 w-4" />
                      ) : granted ? (
                        <IconCheck className="mx-auto h-4 w-4 text-green-600 dark:text-green-400" />
                      ) : (
                        <span className="text-muted-foreground/40">–</span>
                      )}
                    </td>
                  )
                })}
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

export function ResourceVerbPicker({
  resources,
  verbs,
  onResourcesChange,
  onVerbsChange,
}: {
  resources: string[]
  verbs: string[]
  onResourcesChange: (items: string[]) => void
  onVerbsChange: (items: string[]) => void
}) {
  const { t } = useTranslation()

  const allResources = resources.includes('*')
  const allVerbs = verbs.includes('*')
  const customResources = resources.filter(
    (r) => r !== '*' && !resourceCatalog.some((c) => c.type === r)
  )

  const toggle = (list: string[], value: string, checked: boolean) => {
    if (checked) return Array.from(new Set([...list, value]))
    return list.filter((i) => i !== value)
  }

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <Checkbox
            id="rbac-all-resources"
            checked={allResources}
            onCheckedChange={(checked) =>
              onResourcesChange(checked ? ['*'] : [])
            }
          />
          <Label htmlFor="rbac-all-resources" className="text-sm font-medium">
            {t('rbac.matrix.allResources', 'All resources')}
          </Label>
        </div>
        {!allResources && (
          <>
            <div className="grid grid-cols-2 gap-x-4 gap-y-1.5 sm:grid-cols-3">
              {resourceCatalog
                .filter((r) => r.type !== 'crs')
                .map((r) => (
                  <div key={r.type} className="flex items-center gap-2">
                    <Checkbox
                      id={`rbac-res-${r.type}`}
                      checked={resources.includes(r.type)}
                      onCheckedChange={(checked) =>
                        onResourcesChange(
                          toggle(resources, r.type, checked === true)
                        )
                      }
                    />
                    <Label
                      htmlFor={`rbac-res-${r.type}`}
                      className="text-sm font-normal"
                    >
                      {'titleKey' in r
                        ? t(r.titleKey, { defaultValue: r.pluralLabel })
                        : r.pluralLabel}
                    </Label>
                  </div>
                ))}
            </div>
            {customResources.length > 0 && (
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-muted-foreground text-xs">
                  {t('rbac.matrix.customResources', 'Custom resources')}
                </span>
                {customResources.map((r) => (
                  <span
                    key={r}
                    className="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs"
                  >
                    {r}
                    <button
                      type="button"
                      aria-label={`remove ${r}`}
                      className="text-muted-foreground hover:text-foreground"
                      onClick={() =>
                        onResourcesChange(resources.filter((i) => i !== r))
                      }
                    >
                      ×
                    </button>
                  </span>
                ))}
              </div>
            )}
          </>
        )}
      </div>

      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <Checkbox
            id="rbac-all-verbs"
            checked={allVerbs}
            onCheckedChange={(checked) => onVerbsChange(checked ? ['*'] : [])}
          />
          <Label htmlFor="rbac-all-verbs" className="text-sm font-medium">
            {t('rbac.matrix.allVerbs', 'All operations')}
          </Label>
        </div>
        {!allVerbs && (
          <div className="flex flex-wrap gap-x-4 gap-y-1.5">
            {RBAC_VERBS.map((v) => (
              <div key={v} className="flex items-center gap-2">
                <Checkbox
                  id={`rbac-verb-${v}`}
                  checked={verbs.includes(v)}
                  onCheckedChange={(checked) =>
                    onVerbsChange(toggle(verbs, v, checked === true))
                  }
                />
                <Label
                  htmlFor={`rbac-verb-${v}`}
                  className="font-mono text-sm font-normal"
                >
                  {v}
                </Label>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
