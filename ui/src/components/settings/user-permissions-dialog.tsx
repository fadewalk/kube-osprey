import { useMemo } from 'react'
import { IconShieldCheck } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import { UserItem } from '@/types/api'
import { resourceCatalog } from '@/lib/resource-catalog'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

import {
  cellGranted,
  PermissionMatrix,
  RBAC_VERBS,
  type PermissionScope,
} from './permission-matrix'

interface UserPermissionsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: UserItem | null
}

type Scope = Pick<
  PermissionScope,
  'clusters' | 'namespaces' | 'resources' | 'verbs'
>

export function UserPermissionsDialog({
  open,
  onOpenChange,
  user,
}: UserPermissionsDialogProps) {
  const { t } = useTranslation()
  const roles = useMemo(() => user?.roles ?? [], [user])

  // Union of every (resource, verb) cell granted by any of the user's roles;
  // each cell records which roles granted it for troubleshooting.
  const merged = useMemo(() => {
    const map = new Map<
      string,
      { label: string; verbs: Map<string, Set<string>> }
    >()
    const ensure = (type: string) => {
      const entry = resourceCatalog.find((r) => r.type === type)
      if (!map.has(type)) {
        map.set(type, {
          label: entry
            ? 'titleKey' in entry
              ? t(entry.titleKey, { defaultValue: entry.pluralLabel })
              : entry.pluralLabel
            : type,
          verbs: new Map(),
        })
      }
      return map.get(type)!
    }
    for (const role of roles) {
      const scope: Scope = {
        clusters: role.clusters || [],
        namespaces: role.namespaces || [],
        resources: role.resources || [],
        verbs: role.verbs || [],
      }
      const resourceTypes = new Set(
        scope.resources.filter((r) => r !== '*' && !r.startsWith('!'))
      )
      if (scope.resources.includes('*')) {
        for (const r of resourceCatalog) {
          if (r.type !== 'crs') resourceTypes.add(r.type)
        }
      }
      const verbSet = new Set(
        scope.verbs.includes('*')
          ? RBAC_VERBS
          : scope.verbs.filter((v) => !v.startsWith('!'))
      )
      for (const r of resourceTypes) {
        const row = ensure(r)
        for (const v of verbSet) {
          if (cellGranted(scope, r, v)) {
            if (!row.verbs.has(v)) row.verbs.set(v, new Set())
            row.verbs.get(v)!.add(role.name)
          }
        }
      }
    }
    return Array.from(map.entries())
  }, [roles, t])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="!max-w-4xl max-h-[90vh] overflow-y-auto sm:!max-w-4xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <IconShieldCheck className="h-5 w-5" />
            {t('rbac.userPermissions.title', '{{name}} 的权限', {
              name: user?.username || '',
            })}
          </DialogTitle>
          <DialogDescription>
            {t(
              'rbac.userPermissions.description',
              'Effective permissions resolved from every role assigned to this user. Hover a cell to see which role granted it.'
            )}
          </DialogDescription>
        </DialogHeader>

        {roles.length === 0 ? (
          <p className="text-muted-foreground py-6 text-center text-sm">
            {t(
              'rbac.userPermissions.noRoles',
              'No roles assigned to this user.'
            )}
          </p>
        ) : (
          <div className="space-y-6">
            <div>
              <h4 className="mb-2 text-sm font-medium">
                {t('rbac.userPermissions.mergedTitle', 'Merged permissions')}
              </h4>
              <div className="overflow-x-auto rounded-md border">
                <table className="w-full min-w-max text-sm">
                  <thead>
                    <tr className="bg-muted/40 border-b">
                      <th className="px-3 py-2 text-left font-medium">
                        {t('common.fields.resources', 'Resources')}
                      </th>
                      <th className="px-3 py-2 text-left font-medium">
                        {t('common.fields.verbs', 'Verbs')}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {merged.map(([type, row]) => (
                      <tr key={type} className="border-b last:border-b-0">
                        <td className="px-3 py-1.5">{row.label}</td>
                        <td className="px-3 py-1.5">
                          <div className="flex flex-wrap gap-1">
                            {Array.from(row.verbs.entries()).map(
                              ([verb, sources]) => (
                                <Badge
                                  key={verb}
                                  variant="secondary"
                                  className="font-mono text-xs"
                                  title={t(
                                    'rbac.userPermissions.fromRoles',
                                    'Granted by: {{roles}}',
                                    { roles: Array.from(sources).join(', ') }
                                  )}
                                >
                                  {verb}
                                </Badge>
                              )
                            )}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            <div>
              <h4 className="mb-2 text-sm font-medium">
                {t('rbac.userPermissions.byRoleTitle', 'Per role')}
              </h4>
              <div className="space-y-4">
                {roles.map((role) => (
                  <div key={role.id} className="space-y-2">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium">{role.name}</span>
                      {role.isSystem && (
                        <Badge variant="secondary">System</Badge>
                      )}
                      {role.description && (
                        <span className="text-muted-foreground truncate text-xs">
                          {role.description}
                        </span>
                      )}
                      <span className="text-muted-foreground text-xs">
                        {t(
                          'rbac.userPermissions.scopeIn',
                          'clusters {{clusters}} · namespaces {{namespaces}}',
                          {
                            clusters: role.clusters?.join(', ') || '-',
                            namespaces: role.namespaces?.join(', ') || '-',
                          }
                        )}
                      </span>
                    </div>
                    <PermissionMatrix
                      scope={{
                        clusters: role.clusters || [],
                        namespaces: role.namespaces || [],
                        resources: role.resources || [],
                        verbs: role.verbs || [],
                      }}
                    />
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}

export default UserPermissionsDialog
