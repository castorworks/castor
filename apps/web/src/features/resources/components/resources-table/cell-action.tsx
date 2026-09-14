'use client';
import { AlertModal } from '@/components/modal/alert-modal';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu';
import { deleteResourceMutation } from '../../api/mutations';
import type { Resource } from '../../api/types';
import { Icons } from '@/components/icons';
import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { ResourceFormSheet } from '../resource-form-sheet';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { usePermission } from '@/hooks/use-permission';

interface CellActionProps {
  data: Resource;
}

export function CellAction({ data }: CellActionProps) {
  const t = useTranslations('resources');
  const tc = useTranslations('common');
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const canUpdateResource = usePermission('/api/v1/admin/resources/:id:PUT');
  const canDeleteResource = usePermission('/api/v1/admin/resources/:id:DELETE');

  const deleteMutation = useMutation({
    ...mergeMutationOptions(deleteResourceMutation, {
      onSuccess: () => {
        toast.success(t('messages.deleteSuccess'));
        setDeleteOpen(false);
      },
      onError: () => {
        toast.error(t('messages.deleteFailed'));
      }
    })
  });

  if (!canUpdateResource && !canDeleteResource) {
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
      <ResourceFormSheet resource={data} open={editOpen} onOpenChange={setEditOpen} />
      <DropdownMenu modal={false}>
        <DropdownMenuTrigger asChild>
          <Button variant='ghost' className='h-8 w-8 p-0'>
            <span className='sr-only'>{tc('openMenu')}</span>
            <Icons.ellipsis className='h-4 w-4' />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align='end'>
          <DropdownMenuLabel>{tc('actions')}</DropdownMenuLabel>
          {canUpdateResource && (
            <DropdownMenuItem onClick={() => setEditOpen(true)}>
              <Icons.edit className='mr-2 h-4 w-4' /> {tc('update')}
            </DropdownMenuItem>
          )}
          {canUpdateResource && canDeleteResource && <DropdownMenuSeparator />}
          {canDeleteResource &&
            (data.isSystem ? (
              <DropdownMenuItem disabled>
                <Icons.lock className='mr-2 h-4 w-4' /> {t('table.systemCannotDelete')}
              </DropdownMenuItem>
            ) : (
              <DropdownMenuItem onClick={() => setDeleteOpen(true)}>
                <Icons.trash className='mr-2 h-4 w-4' /> {tc('delete')}
              </DropdownMenuItem>
            ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  );
}
