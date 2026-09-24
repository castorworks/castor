'use client';

import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { useAppForm } from '@/components/ui/tanstack-form';
import { Button } from '@/components/ui/button';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle
} from '@/components/ui/sheet';
import { Icons } from '@/components/icons';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { updateJobMutation } from '../api/mutations';
import type { Job } from '../api/types';
import { useJobText } from '../lib/job-name';
import { jobSchema, type JobFormValues } from '../schemas/job';

interface JobFormSheetProps {
  job: Job;
  timeZone: string;
  onClose: () => void;
}

export function JobFormSheet({ job, timeZone, onClose }: JobFormSheetProps) {
  const t = useTranslations('jobs');
  const tc = useTranslations('common');
  const text = useJobText();
  const mutation = useMutation({
    ...mergeMutationOptions(updateJobMutation, {
      onSuccess: () => {
        toast.success(t('messages.saved'));
        onClose();
      },
      onError: (error) => toast.error(error.message || t('messages.saveFailed'))
    })
  });

  const form = useAppForm({
    defaultValues: { cron: job.cron, isEnabled: job.isEnabled } as JobFormValues,
    validators: { onSubmit: jobSchema({ cronInvalid: t('validation.cronInvalid') }) },
    onSubmit: async ({ value }) => {
      await mutation.mutateAsync({
        key: job.key,
        data: { cron: value.cron.trim(), isEnabled: value.isEnabled }
      });
    }
  });

  return (
    <Sheet open onOpenChange={(open) => !open && onClose()}>
      <SheetContent className='flex w-full flex-col sm:max-w-lg'>
        <SheetHeader>
          <SheetTitle>{text.name(job.key)}</SheetTitle>
          <SheetDescription>{t('form.description')}</SheetDescription>
        </SheetHeader>
        <div className='flex-1 overflow-auto px-1'>
          <form.AppForm>
            <form.Form id='job-form' className='space-y-4'>
              <form.TextField
                name='cron'
                label={t('form.cron')}
                description={t('form.cronDescription', { timeZone, defaultCron: job.defaultCron })}
                required
              />
              <Button
                type='button'
                variant='link'
                size='sm'
                className='h-auto px-0'
                onClick={() => form.setFieldValue('cron', job.defaultCron)}
              >
                <Icons.refresh /> {t('form.restoreDefault')}
              </Button>
              <form.SwitchField
                name='isEnabled'
                label={t('form.enabled')}
                description={t('form.enabledDescription')}
              />
            </form.Form>
          </form.AppForm>
        </div>
        <SheetFooter>
          <Button variant='outline' onClick={onClose}>
            {tc('cancel')}
          </Button>
          <Button type='submit' form='job-form' isLoading={mutation.isPending}>
            <Icons.check /> {tc('save')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
