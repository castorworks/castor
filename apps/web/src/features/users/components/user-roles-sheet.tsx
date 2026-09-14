'use client';

import { useMemo, useState } from 'react';
import { useTranslations } from 'next-intl';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle
} from '@/components/ui/sheet';
import { Icons } from '@/components/icons';
import { toast } from 'sonner';
import type { User } from '../api/types';
import { addUserRoleMutation, removeUserRoleMutation } from '../api/mutations';
import {
  allRolesQueryOptions,
  userPermissionsQueryOptions,
  userRolesQueryOptions
} from '../api/queries';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { usePermission } from '@/hooks/use-permission';

interface UserRolesSheetProps {
  user: User;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function UserRolesSheet({ user, open, onOpenChange }: UserRolesSheetProps) {
  const t = useTranslations('users.rolesSheet');
  const tm = useTranslations('users.messages');
  const tc = useTranslations('common');
  const [selectedRole, setSelectedRole] = useState('');
  const canViewRoles = usePermission('/api/v1/admin/authorization/users/:username/roles:GET');
  const canViewPermissions = usePermission(
    '/api/v1/admin/authorization/users/:username/permissions:GET'
  );
  const canListRoles = usePermission('/api/v1/admin/roles:GET');
  const canAssignRole = usePermission('/api/v1/admin/authorization/users/:username/roles:POST');
  const canRemoveRole = usePermission(
    '/api/v1/admin/authorization/users/:username/roles/:role:DELETE'
  );

  const { data: currentRoles = [], isLoading: rolesLoading } = useQuery({
    ...userRolesQueryOptions(user.username),
    enabled: open && canViewRoles
  });

  const { data: allRoles = [], isLoading: allRolesLoading } = useQuery({
    ...allRolesQueryOptions(),
    enabled: open && canListRoles && canAssignRole
  });

  const { data: permissionsData, isLoading: permissionsLoading } = useQuery({
    ...userPermissionsQueryOptions(user.username),
    enabled: open && canViewPermissions
  });

  const availableRoles = useMemo(
    () => allRoles.filter((role) => !currentRoles.includes(role.code)),
    [allRoles, currentRoles]
  );

  const addMutation = useMutation({
    ...mergeMutationOptions(addUserRoleMutation, {
      onSuccess: async () => {
        toast.success(tm('roleAssigned'));
        setSelectedRole('');
      },
      onError: () => toast.error(tm('roleAssignFailed'))
    })
  });

  const removeMutation = useMutation({
    ...mergeMutationOptions(removeUserRoleMutation, {
      onSuccess: async () => {
        toast.success(tm('roleRemoved'));
      },
      onError: () => toast.error(tm('roleRemoveFailed'))
    })
  });

  const isBusy = rolesLoading || allRolesLoading;
  const canShowAssignRole = canViewRoles && canListRoles && canAssignRole;
  const permissions = permissionsData?.permissions ?? [];
  const authorizedRoles = permissionsData?.authorizedRoles ?? [];

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className='w-full sm:max-w-xl'>
        <SheetHeader>
          <SheetTitle>{t('title')}</SheetTitle>
          <SheetDescription>
            {user.username}
            {user.name ? ` (${user.name})` : ''}
          </SheetDescription>
        </SheetHeader>

        <div className='flex-1 overflow-auto pr-1'>
          <div className='space-y-6'>
            <section className='space-y-3'>
              <div className='flex items-center justify-between'>
                <h3 className='text-sm font-medium'>{t('assignedRoles')}</h3>
                {rolesLoading && <Icons.spinner className='h-4 w-4 animate-spin' />}
              </div>
              <div className='flex min-h-10 flex-wrap gap-2'>
                {currentRoles.length > 0 ? (
                  currentRoles.map((role) => (
                    <Badge key={role} variant='secondary' className='gap-1.5 px-2 py-1'>
                      {role}
                      {canRemoveRole && (
                        <button
                          type='button'
                          className='rounded-xs opacity-70 hover:opacity-100 disabled:pointer-events-none'
                          disabled={removeMutation.isPending}
                          onClick={() => removeMutation.mutate({ username: user.username, role })}
                          aria-label={`${tc('remove')} ${role}`}
                        >
                          <Icons.close className='h-3 w-3' />
                        </button>
                      )}
                    </Badge>
                  ))
                ) : (
                  <p className='text-sm text-muted-foreground'>
                    {rolesLoading ? t('loadingRoles') : t('noRoles')}
                  </p>
                )}
              </div>
            </section>

            {canShowAssignRole && (
              <>
                <section className='space-y-3'>
                  <h3 className='text-sm font-medium'>{t('assignRole')}</h3>
                  <div className='flex gap-2'>
                    <Select value={selectedRole} onValueChange={setSelectedRole} disabled={isBusy}>
                      <SelectTrigger className='flex-1'>
                        <SelectValue placeholder={t('selectRole')} />
                      </SelectTrigger>
                      <SelectContent>
                        {availableRoles.map((role) => (
                          <SelectItem key={role.code} value={role.code}>
                            {role.name} ({role.code})
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <Button
                      type='button'
                      disabled={!selectedRole}
                      isLoading={addMutation.isPending}
                      onClick={() =>
                        addMutation.mutate({ username: user.username, role: selectedRole })
                      }
                    >
                      <Icons.add /> {tc('add')}
                    </Button>
                  </div>
                  {!availableRoles.length && !isBusy && (
                    <p className='text-xs text-muted-foreground'>{t('allAssigned')}</p>
                  )}
                </section>

                <Separator />
              </>
            )}

            {canViewPermissions && (
              <section className='space-y-3'>
                <div className='space-y-2'>
                  <h3 className='text-sm font-medium'>{t('authorizedRoles')}</h3>
                  <div className='flex flex-wrap gap-2'>
                    {authorizedRoles.map((role) => (
                      <Badge key={role.code} variant='outline'>
                        {role.name} ({role.code})
                      </Badge>
                    ))}
                  </div>
                </div>
                <Separator />
                <div className='flex items-center justify-between'>
                  <h3 className='text-sm font-medium'>{t('effectivePermissions')}</h3>
                  {permissionsLoading && <Icons.spinner className='h-4 w-4 animate-spin' />}
                </div>
                <div className='space-y-2'>
                  {permissions.length > 0 ? (
                    permissions.map((permission, index) => (
                      <div
                        key={`${permission.roleCode}-${permission.resourcePath}-${permission.action}-${index}`}
                        className='flex items-center justify-between gap-3 rounded-md border px-3 py-2 text-sm'
                      >
                        <div className='min-w-0'>
                          <div className='truncate font-medium'>{permission.resourcePath}</div>
                          <div className='text-xs text-muted-foreground'>{permission.roleCode}</div>
                        </div>
                        <Badge variant='outline'>{permission.action}</Badge>
                      </div>
                    ))
                  ) : (
                    <p className='text-sm text-muted-foreground'>
                      {permissionsLoading ? t('loadingPermissions') : t('noPermissions')}
                    </p>
                  )}
                </div>
              </section>
            )}
          </div>
        </div>
      </SheetContent>
    </Sheet>
  );
}
