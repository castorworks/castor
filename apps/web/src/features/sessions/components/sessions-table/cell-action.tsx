'use client';
import { Modal } from '@/components/ui/modal';
import { Button } from '@/components/ui/button';
import { Icons } from '@/components/icons';
import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { usePermission } from '@/hooks/use-permission';
import { revokeSessionMutation } from '../../api/mutations';
import type { Session } from '../../api/types';

interface CellActionProps {
  data: Session;
}

export function CellAction({ data }: CellActionProps) {
  const t = useTranslations('sessions');
  const tc = useTranslations('common');
  const [confirmOpen, setConfirmOpen] = useState(false);
  const canRevoke = usePermission('/api/v1/admin/sessions/:id:DELETE');

  const revokeMutation = useMutation({
    ...mergeMutationOptions(revokeSessionMutation, {
      onSuccess: () => {
        toast.success(t('messages.revokeSuccess'));
        setConfirmOpen(false);
      },
      onError: (error) => toast.error(error.message || t('messages.revokeFailed'))
    })
  });

  // The caller's own session ends with "Sign out", not from this table.
  if (!canRevoke || data.current) {
    return null;
  }

  return (
    <>
      <Modal
        title={t('revokeTitle')}
        description={t('revokeDescription', { username: data.username })}
        isOpen={confirmOpen}
        onClose={() => setConfirmOpen(false)}
      >
        <div className='flex w-full items-center justify-end gap-2 pt-6'>
          <Button
            disabled={revokeMutation.isPending}
            variant='outline'
            onClick={() => setConfirmOpen(false)}
          >
            {tc('cancel')}
          </Button>
          <Button
            isLoading={revokeMutation.isPending}
            disabled={revokeMutation.isPending}
            variant='destructive'
            onClick={() => revokeMutation.mutate(data.id)}
          >
            {t('revoke')}
          </Button>
        </div>
      </Modal>
      <Button variant='ghost' size='sm' onClick={() => setConfirmOpen(true)}>
        <Icons.logout className='mr-1 h-4 w-4' /> {t('revoke')}
      </Button>
    </>
  );
}
