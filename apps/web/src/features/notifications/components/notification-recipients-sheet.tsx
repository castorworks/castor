'use client';

import { DictBadge } from '@/components/dict-badge';

import { useQuery } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { Badge } from '@/components/ui/badge';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle
} from '@/components/ui/sheet';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@/components/ui/table';
import { Icons } from '@/components/icons';
import { formatDateTime } from '@/lib/format';
import { adminNotificationRecipientsQueryOptions } from '../api/queries';
import type { Notification } from '../api/types';

interface NotificationRecipientsSheetProps {
  notification: Notification;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function formatOptionalDate(value: string | null, locale: string, fallback: string) {
  return value ? formatDateTime(value, { locale }) : fallback;
}

export function NotificationRecipientsSheet({
  notification,
  open,
  onOpenChange
}: NotificationRecipientsSheetProps) {
  const t = useTranslations('notifications.admin');
  const locale = useLocale();
  const { data, isLoading } = useQuery(
    adminNotificationRecipientsQueryOptions(notification.id, { page: 1, pageSize: 100 })
  );

  const recipients = data?.list ?? [];

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className='flex flex-col sm:max-w-3xl'>
        <SheetHeader>
          <SheetTitle>{t('recipients.title')}</SheetTitle>
          <SheetDescription>{notification.title}</SheetDescription>
        </SheetHeader>

        <div className='grid grid-cols-2 gap-3 sm:grid-cols-4'>
          <div className='rounded-md border p-3'>
            <div className='text-muted-foreground text-xs'>{t('table.recipients')}</div>
            <div className='mt-1 text-xl font-semibold tabular-nums'>
              {notification.recipientCount}
            </div>
          </div>
          <div className='rounded-md border p-3'>
            <div className='text-muted-foreground text-xs'>{t('table.read')}</div>
            <div className='mt-1 text-xl font-semibold tabular-nums'>{notification.readCount}</div>
          </div>
          <div className='rounded-md border p-3'>
            <div className='text-muted-foreground text-xs'>{t('table.unread')}</div>
            <div className='mt-1 text-xl font-semibold tabular-nums'>
              {notification.unreadCount}
            </div>
          </div>
          <div className='rounded-md border p-3'>
            <div className='text-muted-foreground text-xs'>{t('table.dismissed')}</div>
            <div className='mt-1 text-xl font-semibold tabular-nums'>
              {notification.dismissedCount}
            </div>
          </div>
        </div>

        <div className='min-h-0 flex-1 overflow-auto rounded-md border'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('recipients.userId')}</TableHead>
                <TableHead>{t('recipients.status')}</TableHead>
                <TableHead>{t('recipients.deliveredAt')}</TableHead>
                <TableHead>{t('recipients.readAt')}</TableHead>
                <TableHead>{t('recipients.dismissedAt')}</TableHead>
                {notification.sendEmail && <TableHead>{t('recipients.email')}</TableHead>}
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={notification.sendEmail ? 6 : 5} className='h-24 text-center'>
                    <Icons.spinner className='mx-auto h-4 w-4 animate-spin' />
                  </TableCell>
                </TableRow>
              ) : recipients.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={notification.sendEmail ? 6 : 5} className='h-24 text-center'>
                    {t('recipients.empty')}
                  </TableCell>
                </TableRow>
              ) : (
                recipients.map((recipient) => (
                  <TableRow key={`${notification.id}-${recipient.userId}`}>
                    <TableCell className='font-mono text-xs'>{recipient.userId}</TableCell>
                    <TableCell>
                      <div className='flex flex-wrap gap-1'>
                        <Badge variant={recipient.isRead ? 'default' : 'secondary'}>
                          {recipient.isRead ? t('table.read') : t('table.unread')}
                        </Badge>
                        {recipient.isDismissed && (
                          <Badge variant='outline'>{t('table.dismissed')}</Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell suppressHydrationWarning>
                      {formatDateTime(recipient.deliveredAt, { locale })}
                    </TableCell>
                    <TableCell suppressHydrationWarning>
                      {formatOptionalDate(recipient.readAt, locale, '-')}
                    </TableCell>
                    <TableCell suppressHydrationWarning>
                      {formatOptionalDate(recipient.dismissedAt, locale, '-')}
                    </TableCell>
                    {notification.sendEmail && (
                      <TableCell>
                        {recipient.emailStatus ? (
                          <DictBadge
                            type='notification_email_status'
                            value={recipient.emailStatus}
                          />
                        ) : (
                          <span className='text-muted-foreground text-xs'>
                            {t('recipients.noEmail')}
                          </span>
                        )}
                      </TableCell>
                    )}
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
        {data && data.total > recipients.length && (
          <p className='text-muted-foreground text-xs'>
            {t('recipients.truncated', { shown: recipients.length, total: data.total })}
          </p>
        )}
      </SheetContent>
    </Sheet>
  );
}
