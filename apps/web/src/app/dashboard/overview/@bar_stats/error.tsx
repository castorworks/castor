'use client';

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { Icons } from '@/components/icons';
import { useRouter } from 'next/navigation';
import { useEffect, useTransition } from 'react';
import * as Sentry from '@sentry/nextjs';
import { useTranslations } from 'next-intl';

interface StatsErrorProps {
  error: Error;
  reset: () => void; // Add reset function from error boundary
}
export default function StatsError({ error, reset }: StatsErrorProps) {
  const router = useRouter();
  const [isPending, startTransition] = useTransition();
  const t = useTranslations('dashboard.overview.errors');
  const tc = useTranslations('common');

  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  // the reload fn ensures the refresh is deffered  until the next render phase allowing react to handle any pending states before processing
  const reload = () => {
    startTransition(() => {
      router.refresh();
      reset();
    });
  };
  return (
    <Card className='border-destructive'>
      <CardHeader className='flex flex-col items-stretch space-y-0 border-b p-0 sm:flex-row'>
        <div className='flex flex-1 flex-col justify-center gap-1 px-6 py-5 sm:py-6'>
          <Alert variant='destructive' className='border-none'>
            <Icons.alertCircle className='h-4 w-4' />
            <AlertTitle>{tc('error')}</AlertTitle>
            <AlertDescription className='mt-2'>
              {t('statistics', { message: error.message })}
            </AlertDescription>
          </Alert>
        </div>
      </CardHeader>
      <CardContent className='flex h-[316px] items-center justify-center p-6'>
        <div className='text-center'>
          <p className='text-muted-foreground mb-4 text-sm'>{t('unableToDisplayStatistics')}</p>
          <Button
            onClick={() => reload()}
            variant='outline'
            className='min-w-[120px]'
            disabled={isPending}
          >
            {t('tryAgain')}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
