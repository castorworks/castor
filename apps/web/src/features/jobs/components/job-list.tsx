'use client';

import { useState } from 'react';
import { useMutation, useSuspenseQuery } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle
} from '@/components/ui/card';
import { DictBadge } from '@/components/dict-badge';
import { Icons } from '@/components/icons';
import { usePermission } from '@/hooks/use-permission';
import { formatDateTime } from '@/lib/format';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { runJobMutation } from '../api/mutations';
import { jobsQueryOptions } from '../api/queries';
import type { Job } from '../api/types';
import { useJobText } from '../lib/job-name';
import { JobFormSheet } from './job-form-sheet';

export function JobList() {
  const t = useTranslations('jobs');
  const locale = useLocale();
  const text = useJobText();
  const { data } = useSuspenseQuery(jobsQueryOptions());
  const [editing, setEditing] = useState<Job | null>(null);
  const canUpdate = usePermission('/api/v1/admin/jobs/:key:PUT');
  const canRun = usePermission('/api/v1/admin/jobs/:key/run:POST');
  const runMutation = useMutation({
    ...mergeMutationOptions(runJobMutation, {
      onSuccess: () => toast.success(t('messages.started')),
      onError: (error) => toast.error(error.message || t('messages.startFailed'))
    })
  });

  if (data.jobs.length === 0) {
    return <p className='text-muted-foreground text-sm'>{t('empty')}</p>;
  }

  return (
    <>
      <div className='grid gap-4 lg:grid-cols-2'>
        {data.jobs.map((job) => {
          const running = job.lastRun?.status === 'RUNNING';
          return (
            <Card key={job.key}>
              <CardHeader>
                <CardTitle className='flex flex-wrap items-center gap-2'>
                  {text.name(job.key)}
                  <Badge variant={job.isEnabled ? 'success' : 'secondary'}>
                    {job.isEnabled ? t('enabled') : t('disabled')}
                  </Badge>
                </CardTitle>
                <CardDescription>{text.description(job.key)}</CardDescription>
                <CardAction className='flex gap-1'>
                  {canRun && (
                    <Button
                      variant='outline'
                      size='sm'
                      disabled={running || runMutation.isPending}
                      onClick={() => runMutation.mutate(job.key)}
                    >
                      <Icons.play /> {t('runNow')}
                    </Button>
                  )}
                  {canUpdate && (
                    <Button
                      variant='ghost'
                      size='sm'
                      aria-label={t('edit')}
                      onClick={() => setEditing(job)}
                    >
                      <Icons.edit />
                    </Button>
                  )}
                </CardAction>
              </CardHeader>
              <CardContent>
                <dl className='grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm'>
                  <dt className='text-muted-foreground'>{t('fields.cron')}</dt>
                  <dd>
                    <code className='font-mono'>{job.cron}</code>
                    <span className='text-muted-foreground ml-2 text-xs'>{data.timeZone}</span>
                  </dd>
                  <dt className='text-muted-foreground'>{t('fields.nextRun')}</dt>
                  <dd suppressHydrationWarning>
                    {job.nextRunAt ? formatDateTime(job.nextRunAt, { locale }) : '—'}
                  </dd>
                  <dt className='text-muted-foreground'>{t('fields.lastRun')}</dt>
                  <dd className='flex flex-wrap items-center gap-2'>
                    {job.lastRun ? (
                      <>
                        <DictBadge type='job_run_status' value={job.lastRun.status} />
                        <span suppressHydrationWarning>
                          {formatDateTime(job.lastRun.startedAt, { locale })}
                        </span>
                        {job.lastRun.status === 'SUCCEEDED' && (
                          <span className='text-muted-foreground'>
                            {t('affected', { count: job.lastRun.affected })}
                          </span>
                        )}
                      </>
                    ) : (
                      t('neverRun')
                    )}
                  </dd>
                  {job.lastRun?.status === 'FAILED' && job.lastRun.message && (
                    <>
                      <dt className='text-muted-foreground'>{t('fields.error')}</dt>
                      <dd className='text-destructive font-mono text-xs break-all'>
                        {job.lastRun.message}
                      </dd>
                    </>
                  )}
                </dl>
              </CardContent>
            </Card>
          );
        })}
      </div>
      {editing && (
        <JobFormSheet job={editing} timeZone={data.timeZone} onClose={() => setEditing(null)} />
      )}
    </>
  );
}
