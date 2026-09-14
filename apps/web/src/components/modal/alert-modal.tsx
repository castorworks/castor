'use client';
import { useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import { Button } from '@/components/ui/button';
import { Modal } from '@/components/ui/modal';

interface AlertModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  loading: boolean;
}

export function AlertModal({ isOpen, onClose, onConfirm, loading }: AlertModalProps) {
  const [isMounted, setIsMounted] = useState(false);
  const t = useTranslations('confirm');
  const tc = useTranslations('common');

  useEffect(() => {
    setIsMounted(true);
  }, []);

  if (!isMounted) {
    return null;
  }

  return (
    <Modal
      title={t('default.title')}
      description={t('default.description')}
      isOpen={isOpen}
      onClose={onClose}
    >
      <div className='flex w-full items-center justify-end space-x-2 pt-6'>
        <Button disabled={loading} variant='outline' onClick={onClose}>
          {tc('cancel')}
        </Button>
        <Button disabled={loading} variant='destructive' onClick={onConfirm}>
          {tc('continue')}
        </Button>
      </div>
    </Modal>
  );
}
