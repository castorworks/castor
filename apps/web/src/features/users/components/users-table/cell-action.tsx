'use client';
import { AlertModal } from '@/components/modal/alert-modal';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu';
import { deleteUserMutation } from '../../api/mutations';
import type { User } from '../../api/types';
import { Icons } from '@/components/icons';
import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { UserFormSheet } from '../user-form-sheet';
import { UserRolesSheet } from '../user-roles-sheet';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { usePermission } from '@/hooks/use-permission';

interface CellActionProps {
  data: User;
}

export function CellAction({ data }: CellActionProps) {
  const t = useTranslations('users');
  const tc = useTranslations('common');
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [rolesOpen, setRolesOpen] = useState(false);
  const canEditUser = usePermission('/api/v1/admin/users/:id:PUT');
  const canDeleteUser = usePermission('/api/v1/admin/users/:id:DELETE');
  const canViewUserRoles = usePermission('/api/v1/admin/authorization/users/:username/roles:GET');
  const canViewUserPermissions = usePermission(
    '/api/v1/admin/authorization/users/:username/permissions:GET'
  );
  const canAssignUserRole = usePermission('/api/v1/admin/authorization/users/:username/roles:POST');
  const canRemoveUserRole = usePermission(
    '/api/v1/admin/authorization/users/:username/roles/:role:DELETE'
  );

  const deleteMutation = useMutation({
    ...mergeMutationOptions(deleteUserMutation, {
      onSuccess: () => {
        toast.success(t('messages.deleteSuccess'));
        setDeleteOpen(false);
      },
      onError: () => {
        toast.error(t('messages.deleteFailed'));
      }
    })
  });

  const canOpenRolesSheet =
    canViewUserRoles || canViewUserPermissions || canAssignUserRole || canRemoveUserRole;

  if (!canEditUser && !canDeleteUser && !canOpenRolesSheet) {
    return null;
  }

  return (
    <>
      <AlertModal
        isOpen={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={() => deleteMutation.mutate(data.id)}
        loading={deleteMutation.isPending}
      />
      <UserFormSheet user={data} open={editOpen} onOpenChange={setEditOpen} />
      <UserRolesSheet user={data} open={rolesOpen} onOpenChange={setRolesOpen} />
      <DropdownMenu modal={false}>
        <DropdownMenuTrigger asChild>
          <Button variant='ghost' className='h-8 w-8 p-0'>
            <span className='sr-only'>{tc('openMenu')}</span>
            <Icons.ellipsis className='h-4 w-4' />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align='end'>
          <DropdownMenuLabel>{tc('actions')}</DropdownMenuLabel>
          {canEditUser && (
            <DropdownMenuItem onClick={() => setEditOpen(true)}>
              <Icons.edit className='mr-2 h-4 w-4' /> {tc('update')}
            </DropdownMenuItem>
          )}
          {canOpenRolesSheet && (
            <DropdownMenuItem onClick={() => setRolesOpen(true)}>
              <Icons.shield className='mr-2 h-4 w-4' /> {t('table.rolesPermissions')}
            </DropdownMenuItem>
          )}
          {canDeleteUser && (
            <DropdownMenuItem onClick={() => setDeleteOpen(true)}>
              <Icons.trash className='mr-2 h-4 w-4' /> {tc('delete')}
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  );
}
