import { IconShieldCheck } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import { UserItem } from '@/types/api'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

import { MergedPermissionsTable, PermissionMatrix } from './permission-matrix'

interface UserPermissionsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: UserItem | null
}

export function UserPermissionsDialog({
  open,
  onOpenChange,
  user,
}: UserPermissionsDialogProps) {
  const { t } = useTranslation()
  const roles = user?.roles ?? []

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="!max-w-4xl max-h-[90vh] overflow-y-auto sm:!max-w-4xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <IconShieldCheck className="h-5 w-5" />
            {t('rbac.userPermissions.title', "{{name}}'s permissions", {
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
              <MergedPermissionsTable roles={roles} />
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
