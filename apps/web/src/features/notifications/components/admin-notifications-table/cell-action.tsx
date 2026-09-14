'use client';

import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { AlertModal } from '@/components/modal/alert-modal';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu';
import { Icons } from '@/components/icons';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { deleteAdminNotificationMutation } from '../../api/mutations';
import type { Notification } from '../../api/types';
import { NotificationFormSheet } from '../notification-form-sheet';
import { NotificationRecipientsSheet } from '../notification-recipients-sheet';
import { usePermission } from '@/hooks/use-permission';

interface CellActionProps {
  data: Notification;
}

export function CellAction({ data }: CellActionProps) {
  const t = useTranslations('notifications.admin');
  const tc = useTranslations('common');
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [recipientsOpen, setRecipientsOpen] = useState(false);
  const canViewRecipients = usePermission('/api/v1/admin/notifications/:id/recipients:GET');
  const canUpdateNotification = usePermission('/api/v1/admin/notifications/:id:PUT');
  const canDeleteNotification = usePermission('/api/v1/admin/notifications/:id:DELETE');

  const deleteMutation = useMutation({
    ...mergeMutationOptions(deleteAdminNotificationMutation, {
      onSuccess: () => {
        toast.success(t('messages.deleteSuccess'));
        setDeleteOpen(false);
      },
      onError: () => toast.error(t('messages.deleteFailed'))
    })
  });

  if (!canViewRecipients && !canUpdateNotification && !canDeleteNotification) {
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
      <NotificationFormSheet notification={data} open={editOpen} onOpenChange={setEditOpen} />
      <NotificationRecipientsSheet
        notification={data}
        open={recipientsOpen}
        onOpenChange={setRecipientsOpen}
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
          {canUpdateNotification && (
            <DropdownMenuItem onClick={() => setEditOpen(true)}>
              <Icons.edit className='mr-2 h-4 w-4' /> {tc('update')}
            </DropdownMenuItem>
          )}
          {canViewRecipients && (
            <DropdownMenuItem onClick={() => setRecipientsOpen(true)}>
              <Icons.teams className='mr-2 h-4 w-4' /> {t('actions.viewRecipients')}
            </DropdownMenuItem>
          )}
          {canDeleteNotification && (
            <DropdownMenuItem onClick={() => setDeleteOpen(true)}>
              <Icons.trash className='mr-2 h-4 w-4' /> {tc('delete')}
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  );
}
