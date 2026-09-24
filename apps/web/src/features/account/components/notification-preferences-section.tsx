'use client';

import { useMutation } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { updateNotificationPreferencesMutation } from '../api/mutations';
import type { AccountInfo } from '../api/types';

/** Whether notifications that admins also send by email reach this user's inbox. */
export function NotificationPreferencesSection({ accountInfo }: { accountInfo: AccountInfo }) {
  const t = useTranslations('profile.notificationPreferences');
  const mutation = useMutation(
    mergeMutationOptions(updateNotificationPreferencesMutation, {
      onSuccess: () => toast.success(t('saved')),
      onError: (error) => toast.error(error.message)
    })
  );
  const receive = !accountInfo.muteNotificationEmails;
  const hasEmail = Boolean(accountInfo.email && accountInfo.emailVerified);

  return (
    <Card className='gap-4'>
      <CardHeader className='pb-0'>
        <CardTitle>{t('title')}</CardTitle>
        <CardDescription>{t('description')}</CardDescription>
      </CardHeader>
      <CardContent className='space-y-2'>
        <div className='flex items-center justify-between gap-4'>
          <Label htmlFor='notification-emails' className='flex flex-col items-start gap-1'>
            <span>{t('emails')}</span>
            <span className='text-muted-foreground text-xs font-normal'>
              {hasEmail
                ? t('emailsDescription', { email: accountInfo.email })
                : t('noVerifiedEmail')}
            </span>
          </Label>
          <Switch
            id='notification-emails'
            checked={receive}
            disabled={mutation.isPending}
            onCheckedChange={(checked) => mutation.mutate(!checked)}
          />
        </div>
      </CardContent>
    </Card>
  );
}
