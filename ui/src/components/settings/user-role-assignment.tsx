import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Role } from '@/types/api'
import { assignRole, unassignRole, useRoleList } from '@/lib/api'
import { formatDate } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  subject?: {
    type: 'user'
    name: string
  }
}

export function UserRoleAssignment({ open, onOpenChange, subject }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const { data: roles = [] } = useRoleList()
  const [selected, setSelected] = useState<Record<number, boolean>>({})
  const [expiresAt, setExpiresAt] = useState('')

  const subjectType = subject?.type || 'user'
  const subjectName = subject?.name

  useEffect(() => {
    if (!subjectName || !roles) return
    const mapping: Record<number, boolean> = {}
    roles.forEach((r) => {
      const has = r.assignments?.some(
        (a) => a.subjectType === subjectType && a.subject === subjectName
      )
      mapping[r.id] = !!has
    })
    setSelected(mapping)
    setExpiresAt('')
  }, [open, subjectName, subjectType, roles])

  const buildExpiry = (): string | undefined => {
    if (!expiresAt) return undefined
    return new Date(`${expiresAt}T23:59:59`).toISOString()
  }

  const toggle = async (roleId: number) => {
    if (!subjectName) return
    try {
      if (selected[roleId]) {
        await unassignRole(roleId, subjectType, subjectName)
        toast.success(t('common.messages.unassigned', 'Unassigned'))
      } else {
        await assignRole(roleId, {
          subjectType: subjectType,
          subject: subjectName,
          expiresAt: buildExpiry(),
        })
        toast.success(t('common.messages.assigned', 'Assigned'))
      }
      queryClient.invalidateQueries({ queryKey: ['user-list'] })
      queryClient.invalidateQueries({ queryKey: ['apikey-list'] })
      queryClient.invalidateQueries({ queryKey: ['role-list'] })
      setSelected((s) => ({ ...s, [roleId]: !s[roleId] }))
    } catch (err: unknown) {
      toast.error(
        (err as Error).message ||
          t('common.messages.failedToAssign', 'Failed to assign')
      )
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {t('common.actions.assignRoles', 'Assign Roles')}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-2">
          <Label htmlFor="role-assignment-expiry">
            {t('roleExpiry.label')}
          </Label>
          <Input
            id="role-assignment-expiry"
            type="date"
            value={expiresAt}
            onChange={(e) => setExpiresAt(e.target.value)}
          />
          <p className="text-muted-foreground text-xs">
            {t('roleExpiry.hint')}
          </p>
        </div>
        <div className="space-y-2 py-2">
          {roles.map((r: Role) => {
            const assignment = r.assignments?.find(
              (a) => a.subjectType === subjectType && a.subject === subjectName
            )
            const expired =
              !!assignment?.expiresAt &&
              new Date(assignment.expiresAt) <= new Date()
            return (
              <div key={r.id} className="flex items-center justify-between">
                <div>
                  <div className="font-medium">{r.name}</div>
                  {r.description && (
                    <div className="text-muted-foreground text-sm">
                      {r.description}
                    </div>
                  )}
                  {assignment?.expiresAt && (
                    <div className="text-muted-foreground mt-1 flex items-center gap-2 text-xs">
                      {expired && (
                        <Badge
                          variant="outline"
                          className="text-destructive px-1.5"
                        >
                          {t('roleExpiry.expired')}
                        </Badge>
                      )}
                      {t('roleExpiry.expiresAt', {
                        date: formatDate(assignment.expiresAt),
                      })}
                    </div>
                  )}
                </div>
                <div>
                  <Checkbox
                    checked={!!selected[r.id]}
                    onCheckedChange={() => toggle(r.id)}
                  />
                </div>
              </div>
            )
          })}
        </div>
        <DialogFooter>
          <Button onClick={() => onOpenChange(false)}>
            {t('common.actions.close', 'Close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export default UserRoleAssignment
