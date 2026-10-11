import { useEffect, useMemo, useState } from 'react'
import { IconPlus, IconSearch, IconTrash, IconUsers } from '@tabler/icons-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Role } from '@/types/api'
import {
  addGroupMembers,
  assignRole,
  createUserGroup,
  deleteUserGroup,
  removeGroupMember,
  unassignRole,
  updateUserGroup,
  useRoleList,
  useUserGroups,
  useUserList,
} from '@/lib/api'
import type { UserGroupItem } from '@/lib/api'
import { formatDate } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
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
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { DeleteConfirmationDialog } from '@/components/delete-confirmation-dialog'

import { Action, ActionTable } from '../action-table'
import { MergedPermissionsTable } from './permission-matrix'

function groupRoles(roles: Role[], groupName: string): Role[] {
  return roles.filter((role) =>
    (role.assignments || []).some(
      (a) => a.subjectType === 'group' && a.subject === groupName
    )
  )
}

export function UserGroupManagement() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { data: groups = [] } = useUserGroups()
  const { data: roles = [] } = useRoleList()

  const [searchQuery, setSearchQuery] = useState('')
  const [creating, setCreating] = useState(false)
  const [editingGroup, setEditingGroup] = useState<UserGroupItem | null>(null)
  const [detailGroup, setDetailGroup] = useState<UserGroupItem | null>(null)
  const [deletingGroup, setDeletingGroup] = useState<UserGroupItem | null>(null)

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['user-groups'] })
    queryClient.invalidateQueries({ queryKey: ['role-list'] })
  }

  const filtered = useMemo(() => {
    const q = searchQuery.trim().toLowerCase()
    if (!q) return groups
    return groups.filter(
      (g) =>
        g.name.toLowerCase().includes(q) ||
        (g.description || '').toLowerCase().includes(q)
    )
  }, [groups, searchQuery])

  const actions = useMemo<Action<UserGroupItem>[]>(
    () => [
      {
        label: t('userGroups.viewDetail', 'View Detail'),
        onClick: (g) => setDetailGroup(g),
      },
      {
        label: t('common.actions.edit', 'Edit'),
        onClick: (g) => setEditingGroup(g),
      },
      {
        label: (
          <div className="inline-flex items-center gap-2 text-destructive">
            <IconTrash className="h-4 w-4" />
            {t('common.actions.delete', 'Delete')}
          </div>
        ),
        onClick: (g) => setDeletingGroup(g),
      },
    ],
    [t]
  )

  const columns = useMemo<ColumnDef<UserGroupItem>[]>(
    () => [
      {
        id: 'name',
        header: t('common.fields.name', 'Name'),
        cell: ({ row: { original: g } }) => (
          <button
            type="button"
            onClick={() => setDetailGroup(g)}
            className="app-link inline-flex items-center gap-2 text-left"
          >
            <IconUsers className="text-muted-foreground h-4 w-4" />
            <span className="font-medium">{g.name}</span>
          </button>
        ),
      },
      {
        id: 'description',
        header: t('common.fields.description', 'Description'),
        cell: ({ row: { original: g } }) => (
          <span
            className="text-muted-foreground line-clamp-1 max-w-72 text-sm"
            title={g.description}
          >
            {g.description || '-'}
          </span>
        ),
      },
      {
        id: 'status',
        header: t('common.fields.status', 'Status'),
        cell: ({ row: { original: g } }) =>
          g.enabled ? (
            <Badge>{t('status.enabled', 'Enabled')}</Badge>
          ) : (
            <Badge variant="secondary">
              {t('status.disabled', 'Disabled')}
            </Badge>
          ),
      },
      {
        id: 'members',
        header: t('userGroups.members', 'Members'),
        cell: ({ row: { original: g } }) => (
          <span className="text-muted-foreground text-sm">
            {g.members?.length ?? 0}
          </span>
        ),
      },
      {
        id: 'roles',
        header: t('common.fields.roles', 'Roles'),
        cell: ({ row: { original: g } }) => (
          <span className="text-muted-foreground text-sm">
            {groupRoles(roles, g.name).length}
          </span>
        ),
      },
      {
        id: 'createdAt',
        header: t('common.fields.createdAt', 'Created At'),
        cell: ({ row: { original: g } }) => (
          <span className="text-muted-foreground text-sm">
            {formatDate(g.createdAt) || '-'}
          </span>
        ),
      },
    ],
    [roles, t]
  )

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle>{t('userGroups.title', 'User Groups')}</CardTitle>
          <p className="text-muted-foreground mt-1 text-sm">
            {t(
              'userGroups.description',
              'Grant roles to groups so members inherit permissions; memberships can carry an expiry date.'
            )}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <div className="relative">
            <IconSearch className="text-muted-foreground absolute top-2.5 left-2 h-4 w-4" />
            <Input
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder={t(
                'userGroups.searchPlaceholder',
                'Search groups...'
              )}
              className="w-56 pl-8"
            />
          </div>
          <Button onClick={() => setCreating(true)}>
            <IconPlus className="h-4 w-4" />
            {t('userGroups.addGroup', 'Add User Group')}
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <ActionTable data={filtered} actions={actions} columns={columns} />
      </CardContent>

      <GroupFormSheet
        open={creating || !!editingGroup}
        group={editingGroup}
        onOpenChange={(open) => {
          if (!open) {
            setCreating(false)
            setEditingGroup(null)
          }
        }}
        onCreated={(group) => {
          invalidate()
          setCreating(false)
          // Kuboard-style: jump straight into the group detail after create.
          setDetailGroup(group)
        }}
      />

      {detailGroup && (
        <GroupDetailSheet
          group={groups.find((g) => g.id === detailGroup.id) || detailGroup}
          roles={roles}
          onOpenChange={(open) => {
            if (!open) {
              setDetailGroup(null)
              invalidate()
            }
          }}
        />
      )}

      <DeleteConfirmationDialog
        open={!!deletingGroup}
        onOpenChange={(open) => {
          if (!open) setDeletingGroup(null)
        }}
        resourceName={deletingGroup?.name || ''}
        resourceType={t('userGroups.title', 'User group')}
        additionalNote={t(
          'userGroups.deleteDescription',
          'Members lose permissions inherited from this group, and its role bindings are removed. This cannot be undone.'
        )}
        onConfirm={async () => {
          if (!deletingGroup) return
          await deleteUserGroup(deletingGroup.id)
          toast.success(t('userGroups.deleted', 'User group deleted'))
          invalidate()
        }}
      />
    </Card>
  )
}

// Drawer form mirroring Kuboard's slide-in create/edit panel.
function GroupFormSheet({
  open,
  group,
  onOpenChange,
  onCreated,
}: {
  open: boolean
  group: UserGroupItem | null
  onOpenChange: (open: boolean) => void
  onCreated: (group: UserGroupItem) => void
}) {
  const { t } = useTranslation()
  const isEdit = !!group
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [enabled, setEnabled] = useState(true)

  useEffect(() => {
    if (!open) return
    setName(group?.name || '')
    setDescription(group?.description || '')
    setEnabled(group?.enabled ?? true)
  }, [group, open])

  const mutation = useMutation({
    mutationFn: async () => {
      if (isEdit && group) {
        return updateUserGroup(group.id, { description, enabled })
      }
      return createUserGroup({ name, description })
    },
    onSuccess: (result) => {
      toast.success(
        isEdit
          ? t('common.messages.updated', {
              resource: t('userGroups.title', 'User group'),
              defaultValue: 'Updated',
            })
          : t('userGroups.created', 'User group created')
      )
      onOpenChange(false)
      if (!isEdit) {
        // Backend returns the created entity; open its detail right away.
        const created = result as UserGroupItem
        if (created?.id) onCreated(created)
      }
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : String(error))
    },
  })

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="sm:!max-w-md" side="right">
        <SheetHeader>
          <SheetTitle>
            {isEdit
              ? t('userGroups.editTitle', 'Edit User Group')
              : t('userGroups.createTitle', 'Add User Group')}
          </SheetTitle>
          <SheetDescription>
            {t(
              'userGroups.formDescription',
              'The group name cannot be changed after creation; role assignments reference it.'
            )}
          </SheetDescription>
        </SheetHeader>
        <form
          className="flex flex-1 flex-col gap-4 overflow-y-auto px-4 pb-4"
          onSubmit={(e) => {
            e.preventDefault()
            mutation.mutate()
          }}
        >
          <div className="space-y-2">
            <Label htmlFor="group-name">
              {t('common.fields.name', 'Name')} *
            </Label>
            <Input
              id="group-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              disabled={isEdit}
              minLength={2}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="group-desc">
              {t('common.fields.description', 'Description')}
            </Label>
            <Input
              id="group-desc"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          {isEdit && (
            <div className="flex items-center justify-between rounded-md border p-3">
              <div>
                <Label className="text-sm font-medium">
                  {t('userGroups.groupEnabled', 'Enabled')}
                </Label>
                <p className="text-muted-foreground text-xs">
                  {t(
                    'userGroups.groupEnabledHint',
                    'Disabled groups suspend permissions for all members.'
                  )}
                </p>
              </div>
              <Switch checked={enabled} onCheckedChange={setEnabled} />
            </div>
          )}
          <div className="mt-auto flex justify-end gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              {t('common.actions.cancel', 'Cancel')}
            </Button>
            <Button type="submit" disabled={mutation.isPending}>
              {mutation.isPending
                ? t('common.actions.saving', 'Saving...')
                : isEdit
                  ? t('common.actions.save', 'Save')
                  : t('common.actions.create', 'Create')}
            </Button>
          </div>
        </form>
      </SheetContent>
    </Sheet>
  )
}

function GroupDetailSheet({
  group,
  roles,
  onOpenChange,
}: {
  group: UserGroupItem
  roles: Role[]
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [tab, setTab] = useState('members')
  const [addingMembers, setAddingMembers] = useState(false)
  const [bindingRole, setBindingRole] = useState(false)

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['user-groups'] })
    queryClient.invalidateQueries({ queryKey: ['role-list'] })
  }

  const boundRoles = groupRoles(roles, group.name)
  const removeMember = useMutation({
    mutationFn: (memberId: number) => removeGroupMember(group.id, memberId),
    onSuccess: () => {
      toast.success(t('userGroups.memberRemoved', 'Member removed'))
      invalidate()
    },
    onError: (error) => toast.error(String(error)),
  })
  const unassign = useMutation({
    mutationFn: (roleId: number) => unassignRole(roleId, 'group', group.name),
    onSuccess: () => {
      toast.success(t('userGroups.roleUnbound', 'Role unbound'))
      invalidate()
    },
    onError: (error) => toast.error(String(error)),
  })
  const toggleEnabled = useMutation({
    mutationFn: (enabled: boolean) => updateUserGroup(group.id, { enabled }),
    onSuccess: () => invalidate(),
    onError: (error) => toast.error(String(error)),
  })

  return (
    <Sheet open onOpenChange={onOpenChange}>
      <SheetContent className="flex flex-col gap-0 sm:!max-w-2xl" side="right">
        <SheetHeader className="pb-2">
          <SheetTitle className="flex items-center gap-3">
            {group.name}
            {group.enabled ? (
              <Badge>{t('status.enabled', 'Enabled')}</Badge>
            ) : (
              <Badge variant="secondary">
                {t('status.disabled', 'Disabled')}
              </Badge>
            )}
            <span className="text-muted-foreground flex items-center gap-1 text-xs font-normal">
              <Switch
                checked={group.enabled}
                onCheckedChange={(checked) => toggleEnabled.mutate(checked)}
              />
              {t('userGroups.groupEnabled', 'Enabled')}
            </span>
          </SheetTitle>
          {group.description && (
            <SheetDescription>{group.description}</SheetDescription>
          )}
        </SheetHeader>
        <Tabs
          value={tab}
          onValueChange={setTab}
          className="flex min-h-0 flex-1 flex-col gap-3 px-4 pb-4"
        >
          <TabsList>
            <TabsTrigger value="members">
              {t('userGroups.membersTab', 'Members')} (
              {group.members?.length ?? 0})
            </TabsTrigger>
            <TabsTrigger value="roles">
              {t('userGroups.rolesTab', 'Roles')} ({boundRoles.length})
            </TabsTrigger>
            <TabsTrigger value="permissions">
              {t('userGroups.permissionsTab', 'Permissions')}
            </TabsTrigger>
          </TabsList>

          <TabsContent
            value="members"
            className="min-h-0 flex-1 space-y-3 overflow-y-auto"
          >
            <div className="flex justify-end">
              <Button size="sm" onClick={() => setAddingMembers(true)}>
                <IconPlus className="h-4 w-4" />
                {t('userGroups.addMembers', 'Add Members')}
              </Button>
            </div>
            <div className="rounded-md border">
              {(group.members || []).length === 0 ? (
                <p className="text-muted-foreground p-6 text-center text-sm">
                  {t('userGroups.noMembers', 'No members in this group yet.')}
                </p>
              ) : (
                (group.members || []).map((m) => {
                  const expired =
                    m.expiresAt && new Date(m.expiresAt) <= new Date()
                  return (
                    <div
                      key={m.id}
                      className="flex items-center justify-between gap-3 border-b px-3 py-2 last:border-b-0"
                    >
                      <div className="flex min-w-0 flex-col">
                        <span className="truncate text-sm font-medium">
                          {m.user.username}
                          {m.user.name ? (
                            <span className="text-muted-foreground ml-2 font-normal">
                              {m.user.name}
                            </span>
                          ) : null}
                        </span>
                        <span className="text-muted-foreground text-xs">
                          {m.user.provider}
                          {m.expiresAt
                            ? ` · ${t('userGroups.expiresAt', 'expires {{date}}', { date: formatDate(m.expiresAt) || m.expiresAt })}`
                            : ''}
                        </span>
                      </div>
                      <div className="flex items-center gap-2">
                        {expired && (
                          <Badge variant="destructive">
                            {t('userGroups.expired', 'Expired')}
                          </Badge>
                        )}
                        <Button
                          size="icon"
                          variant="ghost"
                          aria-label={t('common.actions.delete', 'Delete')}
                          onClick={() => removeMember.mutate(m.id)}
                        >
                          <IconTrash className="text-muted-foreground h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  )
                })
              )}
            </div>
          </TabsContent>

          <TabsContent
            value="roles"
            className="min-h-0 flex-1 space-y-3 overflow-y-auto"
          >
            <div className="flex justify-end">
              <Button size="sm" onClick={() => setBindingRole(true)}>
                <IconPlus className="h-4 w-4" />
                {t('userGroups.bindRole', 'Bind Role')}
              </Button>
            </div>
            <div className="rounded-md border">
              {boundRoles.length === 0 ? (
                <p className="text-muted-foreground p-6 text-center text-sm">
                  {t(
                    'userGroups.noRoles',
                    'No roles bound. Bind a role to grant permissions to every member.'
                  )}
                </p>
              ) : (
                boundRoles.map((role) => (
                  <div
                    key={role.id}
                    className="flex items-center justify-between gap-3 border-b px-3 py-2 last:border-b-0"
                  >
                    <div className="flex min-w-0 flex-col">
                      <span className="flex items-center gap-2 text-sm font-medium">
                        {role.name}
                        {role.isSystem && (
                          <Badge variant="secondary">System</Badge>
                        )}
                      </span>
                      {role.description && (
                        <span className="text-muted-foreground truncate text-xs">
                          {role.description}
                        </span>
                      )}
                    </div>
                    <Button
                      size="icon"
                      variant="ghost"
                      aria-label={t('common.actions.delete', 'Delete')}
                      onClick={() => unassign.mutate(role.id)}
                    >
                      <IconTrash className="text-muted-foreground h-4 w-4" />
                    </Button>
                  </div>
                ))
              )}
            </div>
          </TabsContent>

          <TabsContent
            value="permissions"
            className="min-h-0 flex-1 overflow-y-auto"
          >
            <MergedPermissionsTable roles={boundRoles} />
          </TabsContent>
        </Tabs>

        <AddMembersDialog
          open={addingMembers}
          group={group}
          onOpenChange={setAddingMembers}
          onAdded={() => {
            setAddingMembers(false)
            invalidate()
          }}
        />
        <BindRoleDialog
          open={bindingRole}
          group={group}
          roles={roles}
          boundRoles={boundRoles}
          onOpenChange={setBindingRole}
          onBound={() => {
            setBindingRole(false)
            invalidate()
          }}
        />
      </SheetContent>
    </Sheet>
  )
}

function AddMembersDialog({
  open,
  group,
  onOpenChange,
  onAdded,
}: {
  open: boolean
  group: UserGroupItem
  onOpenChange: (open: boolean) => void
  onAdded: () => void
}) {
  const { t } = useTranslation()
  const { data } = useUserList(1, 100, '')
  const users = data?.users || []
  const existing = new Set((group.members || []).map((m) => m.userId))
  const candidates = users.filter((u) => !existing.has(u.id))

  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [expiresAt, setExpiresAt] = useState('')

  useEffect(() => {
    if (open) {
      setSelected(new Set())
      setExpiresAt('')
    }
  }, [open])

  const mutation = useMutation({
    mutationFn: () =>
      addGroupMembers(group.id, {
        userIds: Array.from(selected),
        expiresAt: expiresAt
          ? new Date(`${expiresAt}T23:59:59`).toISOString()
          : null,
      }),
    onSuccess: (result) => {
      toast.success(
        t('userGroups.membersAdded', '{{count}} member(s) added', {
          count: result.added,
        })
      )
      onAdded()
    },
    onError: (error) => toast.error(String(error)),
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('userGroups.addMembers', 'Add Members')}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <div className="max-h-64 space-y-1 overflow-y-auto rounded-md border p-2">
            {candidates.length === 0 && (
              <p className="text-muted-foreground p-4 text-center text-sm">
                {t(
                  'userGroups.allUsersAdded',
                  'All users are already members.'
                )}
              </p>
            )}
            {candidates.map((u) => (
              <button
                key={u.id}
                type="button"
                onClick={() =>
                  setSelected((prev) => {
                    const next = new Set(prev)
                    if (next.has(u.id)) next.delete(u.id)
                    else next.add(u.id)
                    return next
                  })
                }
                className="hover:bg-accent/50 flex w-full items-center gap-2 rounded-sm px-2 py-1 text-left"
              >
                <Checkbox checked={selected.has(u.id)} />
                <span className="flex-1 text-sm">
                  {u.username}
                  {u.name ? (
                    <span className="text-muted-foreground ml-2">{u.name}</span>
                  ) : null}
                  <span className="text-muted-foreground ml-2 text-xs">
                    ({u.provider})
                  </span>
                </span>
              </button>
            ))}
          </div>
          <div className="space-y-2">
            <Label htmlFor="member-expiry">
              {t('userGroups.membershipExpiry', 'Membership expires at')}
            </Label>
            <Input
              id="member-expiry"
              type="date"
              value={expiresAt}
              onChange={(e) => setExpiresAt(e.target.value)}
            />
            <p className="text-muted-foreground text-xs">
              {t(
                'userGroups.membershipExpiryHint',
                'Optional. Expired members automatically lose group permissions.'
              )}
            </p>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.actions.cancel', 'Cancel')}
          </Button>
          <Button
            onClick={() => mutation.mutate()}
            disabled={selected.size === 0 || mutation.isPending}
          >
            {t('userGroups.addMembers', 'Add Members')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function BindRoleDialog({
  open,
  group,
  roles,
  boundRoles,
  onOpenChange,
  onBound,
}: {
  open: boolean
  group: UserGroupItem
  roles: Role[]
  boundRoles: Role[]
  onOpenChange: (open: boolean) => void
  onBound: () => void
}) {
  const { t } = useTranslation()
  const boundIds = new Set(boundRoles.map((r) => r.id))
  const candidates = roles.filter((r) => !boundIds.has(r.id))
  const [roleId, setRoleId] = useState<number | null>(null)

  useEffect(() => {
    if (open) setRoleId(null)
  }, [open])

  const mutation = useMutation({
    mutationFn: () => {
      if (roleId == null) throw new Error('no role selected')
      return assignRole(roleId, { subjectType: 'group', subject: group.name })
    },
    onSuccess: () => {
      toast.success(t('userGroups.roleBound', 'Role bound'))
      onBound()
    },
    onError: (error) => toast.error(String(error)),
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('userGroups.bindRole', 'Bind Role')}</DialogTitle>
        </DialogHeader>
        <div className="space-y-2">
          <Label htmlFor="bind-role">{t('common.fields.role', 'Role')} *</Label>
          <select
            id="bind-role"
            value={roleId ?? ''}
            onChange={(e) =>
              setRoleId(e.target.value ? Number(e.target.value) : null)
            }
            className="border-input bg-transparent flex h-9 w-full rounded-md border px-3 py-1 text-sm shadow-xs focus-visible:ring-2 focus-visible:ring-ring/50 focus-visible:outline-none"
          >
            <option value="" disabled>
              {t('userGroups.selectRole', 'Select a role...')}
            </option>
            {candidates.map((r) => (
              <option key={r.id} value={r.id}>
                {r.name}
                {r.description ? ` — ${r.description}` : ''}
              </option>
            ))}
          </select>
          {candidates.length === 0 && (
            <p className="text-muted-foreground text-xs">
              {t('userGroups.allRolesBound', 'Every role is already bound.')}
            </p>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.actions.cancel', 'Cancel')}
          </Button>
          <Button
            onClick={() => mutation.mutate()}
            disabled={roleId == null || mutation.isPending}
          >
            {t('common.actions.confirm', 'Confirm')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
