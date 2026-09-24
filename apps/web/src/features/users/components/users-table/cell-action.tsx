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
import { deleteUserMutation, resetUserTwoFactorMutation } from '../../api/mutations';
import { Modal } from '@/components/ui/modal';
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
  const [resetTwoFactorOpen, setResetTwoFactorOpen] = useState(false);
  const canResetTwoFactor =
    usePermission('/api/v1/admin/users/:id/totp:DELETE') && data.totpEnabled === true;
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

  const resetTwoFactor = useMutation({
    ...mergeMutationOptions(resetUserTwoFactorMutation, {
      onSuccess: () => {
        toast.success(t('twoFactor.resetSuccess'));
        setResetTwoFactorOpen(false);
      },
      onError: (error) => toast.error(error.message || t('twoFactor.resetFailed'))
    })
  });

  const canOpenRolesSheet =
    canViewUserRoles || canViewUserPermissions || canAssignUserRole || canRemoveUserRole;

  if (!canEditUser && !canDeleteUser && !canOpenRolesSheet && !canResetTwoFactor) {
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
      <Modal
        title={t('twoFactor.resetTitle')}
        description={t('twoFactor.resetDescription', { username: data.username })}
        isOpen={resetTwoFactorOpen}
        onClose={() => setResetTwoFactorOpen(false)}
      >
        <div className='flex w-full items-center justify-end gap-2 pt-6'>
          <Button
            variant='outline'
            disabled={resetTwoFactor.isPending}
            onClick={() => setResetTwoFactorOpen(false)}
          >
            {tc('cancel')}
          </Button>
          <Button
            variant='destructive'
            isLoading={resetTwoFactor.isPending}
            disabled={resetTwoFactor.isPending}
            onClick={() => resetTwoFactor.mutate(data.id)}
          >
            {t('twoFactor.reset')}
          </Button>
        </div>
      </Modal>
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
          {canResetTwoFactor && (
            <DropdownMenuItem onClick={() => setResetTwoFactorOpen(true)}>
              <Icons.lock className='mr-2 h-4 w-4' /> {t('twoFactor.reset')}
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
