'use client';

import { useQuery, useMutation } from '@tanstack/react-query';
import { Icons } from '@/components/icons';
import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { NotificationCard } from '@/components/ui/notification-card';
import { EmptyState } from '@/components/ui/empty-state';
import { userNotificationsQueryOptions, unreadCountQueryOptions } from '../api/queries';
import { markAllReadMutation, markNotificationReadMutation } from '../api/mutations';
import { toast } from 'sonner';
import type { Notification } from '../api/types';
import { useTranslations } from 'next-intl';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { useAuthStore } from '@/stores/auth-store';
import { shouldEnableNotificationQueries } from '../utils/auth';

const MAX_VISIBLE = 5;

function toCardProps(n: Notification) {
  return {
    id: String(n.id),
    title: n.title,
    body: n.content,
    status: n.isRead ? ('read' as const) : ('unread' as const),
    createdAt: n.createdAt
  };
}

export function NotificationCenter() {
  const t = useTranslations('notifications');
  const isAuthenticated = useAuthStore((s) => s.user !== null);
  const isAuthLoading = useAuthStore((s) => s.isLoading);
  const enabled = shouldEnableNotificationQueries(isAuthenticated, isAuthLoading);
  const { data: notificationsData } = useQuery(
    userNotificationsQueryOptions({ page: 1, pageSize: MAX_VISIBLE }, undefined, { enabled })
  );
  const { data: unreadData } = useQuery(unreadCountQueryOptions(undefined, { enabled }));

  const markAllRead = useMutation({
    ...mergeMutationOptions(markAllReadMutation, {
      onSuccess: async () => {
        toast.success(t('messages.allMarkedRead'));
      }
    })
  });

  const markRead = useMutation({
    ...mergeMutationOptions(markNotificationReadMutation, {
      onSuccess: async () => {
        toast.success(t('messages.markedRead'));
      }
    })
  });

  const notifications = notificationsData?.list ?? [];
  const count = unreadData?.count ?? 0;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant='ghost' size='icon' className='relative h-8 w-8'>
          <Icons.notification className='h-4 w-4' />
          {count > 0 && (
            <span className='bg-destructive text-destructive-foreground absolute -top-0.5 -right-0.5 flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-[10px] font-medium'>
              {count > 9 ? '9+' : count}
            </span>
          )}
          <span className='sr-only'>{t('title')}</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent align='end' className='w-[calc(100vw-2rem)] p-0 sm:w-[380px]' sideOffset={8}>
        <div className='flex items-center justify-between px-4 py-3'>
          <Link href='/dashboard/notifications' className='group flex items-center gap-1'>
            <h4 className='text-sm font-semibold group-hover:underline'>{t('title')}</h4>
            <Icons.chevronRight className='text-muted-foreground h-3.5 w-3.5 transition-transform group-hover:translate-x-0.5' />
          </Link>
          <div className='flex items-center gap-2'>
            {count > 0 && (
              <span className='bg-muted text-muted-foreground rounded-full px-2 py-0.5 text-xs'>
                {t('newCount', { count })}
              </span>
            )}
            {count > 0 && (
              <Button
                variant='ghost'
                size='sm'
                className='text-muted-foreground h-auto px-2 py-1 text-xs'
                onClick={() => markAllRead.mutate()}
                disabled={markAllRead.isPending}
              >
                {t('markAllRead')}
              </Button>
            )}
          </div>
        </div>
        <Separator />
        <ScrollArea className='h-[400px]'>
          {notifications.length === 0 ? (
            <EmptyState icon={<Icons.notification />} title={t('emptyYet')} className='py-12' />
          ) : (
            <div className='flex flex-col gap-1 p-2'>
              {notifications.map((notification) => {
                const cardProps = toCardProps(notification);
                return (
                  <NotificationCard
                    key={notification.id}
                    {...cardProps}
                    markAsReadLabel={t('markRead')}
                    onMarkAsRead={enabled ? (id) => markRead.mutate(Number(id)) : undefined}
                  />
                );
              })}
            </div>
          )}
        </ScrollArea>
      </PopoverContent>
    </Popover>
  );
}
