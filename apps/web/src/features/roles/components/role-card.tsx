'use client';

import { useState } from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Icons } from '@/components/icons';
import type { Role } from '../api/types';
import { RolePermissionsSheet } from './role-permissions-sheet';
import { RoleUsersSheet } from './role-users-sheet';
import { RoleFormSheet } from './role-form-sheet';
import { RoleHierarchySheet } from './role-hierarchy-sheet';
import { AlertModal } from '@/components/modal/alert-modal';
import { useMutation } from '@tanstack/react-query';
import { deleteRoleMutation, toggleRoleEnabledMutation } from '../api/mutations';
import { useTranslations } from 'next-intl';
import { useSeedTranslation } from '@/lib/use-seed-translation';
import { toast } from 'sonner';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { useAllPermissions, usePermission } from '@/hooks/use-permission';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu';

interface RoleCardProps {
  role: Role;
}

export function RoleCard({ role }: RoleCardProps) {
  const t = useTranslations('roles');
  const tc = useTranslations('common');
  const { ts } = useSeedTranslation();
  const [permissionsOpen, setPermissionsOpen] = useState(false);
  const [usersOpen, setUsersOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [hierarchyOpen, setHierarchyOpen] = useState(false);
  const canViewUsers = usePermission('/api/v1/admin/roles/:role/users:GET');
  const canViewPermissions = usePermission('/api/v1/admin/roles/:role/permissions:GET');
  const canViewHierarchy = usePermission('/api/v1/admin/roles/:role/hierarchy:GET');
  const canToggle = usePermission('/api/v1/admin/roles/:role:PUT');
  const canEdit = useAllPermissions([
    '/api/v1/admin/roles/:role:PUT',
    '/api/v1/admin/roles/:role/permissions:PUT'
  ]);
  const canDelete = usePermission('/api/v1/admin/roles/:role:DELETE');
  const hasMoreActions =
    canViewUsers || canViewHierarchy || (!role.isSystem && (canEdit || canDelete));

  const deleteMutation = useMutation({
    ...mergeMutationOptions(deleteRoleMutation, {
      onSuccess: () => {
        toast.success(t('messages.deleteSuccess'));
        setDeleteOpen(false);
      },
      onError: () => {
        toast.error(t('messages.deleteFailed'));
      }
    })
  });

  const toggleMutation = useMutation({
    ...mergeMutationOptions(toggleRoleEnabledMutation, {
      onError: () => {
        toast.error(t('messages.statusUpdateFailed'));
      }
    })
  });

  const handleToggle = (checked: boolean) => {
    toggleMutation.mutate({ code: role.code, isEnabled: checked });
  };

  return (
    <>
      <AlertModal
        isOpen={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={() => deleteMutation.mutate(role.code)}
        loading={deleteMutation.isPending}
      />
      <Card className={!role.isEnabled ? 'opacity-60' : ''}>
        <CardHeader className='pb-3'>
          <div className='flex items-start justify-between gap-3'>
            <div className='flex flex-col'>
              <CardTitle className='text-lg'>{ts(role.name)}</CardTitle>
              <CardDescription className='font-mono text-xs'>{role.code}</CardDescription>
            </div>
            <div className='flex items-center gap-2'>
              <div className='flex flex-wrap gap-1'>
                {role.isSystem && (
                  <Badge variant='secondary' className='text-xs'>
                    <Icons.shield className='mr-1 h-3 w-3' /> {t('card.system')}
                  </Badge>
                )}
                <Badge variant={role.isEnabled ? 'default' : 'destructive'} className='text-xs'>
                  {role.isEnabled ? tc('enabled') : tc('disabled')}
                </Badge>
              </div>
              <Switch
                checked={role.isEnabled}
                onCheckedChange={handleToggle}
                disabled={!canToggle || toggleMutation.isPending}
                aria-label={t('card.toggle', {
                  name: role.name,
                  state: role.isEnabled ? t('card.off') : t('card.on')
                })}
              />
            </div>
          </div>
          {role.description && (
            <p className='mt-1 text-sm text-muted-foreground'>{ts(role.description)}</p>
          )}
        </CardHeader>
        <CardContent>
          <div className='flex flex-wrap items-center justify-between gap-3'>
            <div className='flex items-center gap-2 text-sm text-muted-foreground'>
              <Icons.user className='h-4 w-4' />
              <span>
                {tc(role.userCount === 1 ? 'userCount' : 'userCountPlural', {
                  count: role.userCount ?? 0
                })}
              </span>
            </div>
            <div className='flex flex-wrap justify-end gap-2'>
              {canViewPermissions && (
                <Button onClick={() => setPermissionsOpen(true)}>
                  <Icons.shield className='mr-1 h-4 w-4' /> {t('card.authorize')}
                </Button>
              )}
              {hasMoreActions && (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant='outline' size='icon' aria-label={t('card.moreActions')}>
                      <Icons.ellipsis className='h-4 w-4' />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align='end'>
                    {canViewUsers && (
                      <DropdownMenuItem onSelect={() => setUsersOpen(true)}>
                        <Icons.user /> {t('card.users')}
                      </DropdownMenuItem>
                    )}
                    {canViewHierarchy && (
                      <DropdownMenuItem onSelect={() => setHierarchyOpen(true)}>
                        <Icons.chevronRight /> {t('card.hierarchy')}
                      </DropdownMenuItem>
                    )}
                    {!role.isSystem && canEdit && (
                      <DropdownMenuItem onSelect={() => setEditOpen(true)}>
                        <Icons.edit /> {tc('edit')}
                      </DropdownMenuItem>
                    )}
                    {!role.isSystem && canDelete && (
                      <>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          variant='destructive'
                          onSelect={() => setDeleteOpen(true)}
                        >
                          <Icons.trash /> {tc('delete')}
                        </DropdownMenuItem>
                      </>
                    )}
                  </DropdownMenuContent>
                </DropdownMenu>
              )}
              {!canViewPermissions && !hasMoreActions && (
                <span className='text-xs text-muted-foreground'>{tc('view')}</span>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      <RoleFormSheet role={role} open={editOpen} onOpenChange={setEditOpen} />
      <RolePermissionsSheet role={role} open={permissionsOpen} onOpenChange={setPermissionsOpen} />
      <RoleUsersSheet role={role} open={usersOpen} onOpenChange={setUsersOpen} />
      <RoleHierarchySheet role={role} open={hierarchyOpen} onOpenChange={setHierarchyOpen} />
    </>
  );
}
