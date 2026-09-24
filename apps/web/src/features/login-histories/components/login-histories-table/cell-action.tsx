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
import { deleteLoginHistoryMutation } from '../../api/mutations';
import type { LoginHistory } from '../../api/types';
import { Icons } from '@/components/icons';
import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { usePermission } from '@/hooks/use-permission';

interface CellActionProps {
  data: LoginHistory;
}

export function CellAction({ data }: CellActionProps) {
  const t = useTranslations('loginHistory');
  const tc = useTranslations('common');
  const [deleteOpen, setDeleteOpen] = useState(false);
  const canDelete = usePermission('/api/v1/admin/login-histories/:id:DELETE');

  const deleteMutation = useMutation({
    ...mergeMutationOptions(deleteLoginHistoryMutation, {
      onSuccess: () => {
        toast.success(t('messages.deleteSuccess'));
        setDeleteOpen(false);
      },
      onError: () => {
        toast.error(t('messages.deleteFailed'));
      }
    })
  });

  if (!canDelete) {
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
      <DropdownMenu modal={false}>
        <DropdownMenuTrigger asChild>
          <Button variant='ghost' className='h-8 w-8 p-0'>
            <span className='sr-only'>{tc('openMenu')}</span>
            <Icons.ellipsis className='h-4 w-4' />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align='end'>
          <DropdownMenuLabel>{tc('actions')}</DropdownMenuLabel>
          <DropdownMenuItem onClick={() => setDeleteOpen(true)}>
            <Icons.trash className='mr-2 h-4 w-4' /> {tc('delete')}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  );
}
