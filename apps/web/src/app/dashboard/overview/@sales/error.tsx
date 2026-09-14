'use client';

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Icons } from '@/components/icons';
import { useTranslations } from 'next-intl';

export default function SalesError({ error }: { error: Error }) {
  const t = useTranslations('dashboard.overview.errors');
  const tc = useTranslations('common');

  return (
    <Alert variant='destructive'>
      <Icons.alertCircle className='h-4 w-4' />
      <AlertTitle>{tc('error')}</AlertTitle>
      <AlertDescription>{t('salesData', { message: error.message })}</AlertDescription>
    </Alert>
  );
}
