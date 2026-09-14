'use client';

import { useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { toast } from 'sonner';
import { useRouter } from 'next/navigation';
import { Icons } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { NotificationCard } from '@/components/ui/notification-card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { EmptyState } from '@/components/ui/empty-state';
import { AlertModal } from '@/components/modal/alert-modal';
import { userNotificationsQueryOptions, unreadCountQueryOptions } from '../api/queries';
import {
  markNotificationReadMutation,
  markAllReadMutation,
  deleteNotificationMutation
} from '../api/mutations';
import type { Notification } from '../api/types';
import { getSafeNotificationHref } from '../utils/links';
import { getDictLabel, getDictVariant } from '@/lib/dict';
import { useTranslations } from 'next-intl';
import { mergeMutationOptions } from '@/lib/mutation-utils';
import { useAuthStore } from '@/stores/auth-store';
import { shouldEnableNotificationQueries } from '../utils/auth';

const PAGE_SIZE = 10;

function toNotificationCard(n: Notification): {
  id: string;
  title: string;
  body: string;
  status: 'unread' | 'read';
  createdAt: string;
} {
  return {
    id: String(n.id),
    title: n.title,
    body: n.content,
    status: n.isRead ? 'read' : 'unread',
    createdAt: n.createdAt
  };
}

export default function NotificationsPage() {
  const t = useTranslations('notifications');
  const tc = useTranslations('common');
  const router = useRouter();
  const isAuthenticated = useAuthStore((s) => s.user !== null);
  const isAuthLoading = useAuthStore((s) => s.isLoading);
  const queriesEnabled = shouldEnableNotificationQueries(isAuthenticated, isAuthLoading);
  const [page, setPage] = useState(1);
  const [tab, setTab] = useState('all');
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [notificationToDelete, setNotificationToDelete] = useState<Notification | null>(null);

  const { data: notificationsData } = useQuery(
    userNotificationsQueryOptions(
      { page, pageSize: PAGE_SIZE, unreadOnly: tab === 'unread' },
      undefined,
      { enabled: queriesEnabled }
    )
  );
  const { data: unreadData } = useQuery(
    unreadCountQueryOptions(undefined, { enabled: queriesEnabled })
  );

  const markRead = useMutation({
    ...mergeMutationOptions(markNotificationReadMutation, {
      onSuccess: async () => {
        toast.success(t('messages.markedRead'));
      }
    })
  });

  const markAllRead = useMutation({
    ...mergeMutationOptions(markAllReadMutation, {
      onSuccess: async () => {
        toast.success(t('messages.allMarkedRead'));
      }
    })
  });

  const deleteNotif = useMutation({
    ...mergeMutationOptions(deleteNotificationMutation, {
      onSuccess: async () => {
        toast.success(t('messages.deleted'));
        setDeleteOpen(false);
      },
      onError: () => toast.error(t('messages.deleteFailed'))
    })
  });

  const notifications = notificationsData?.list ?? [];
  const unreadCount = unreadData?.count ?? 0;
  const total = notificationsData?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const unreadNotifications = notifications.filter((n) => !n.isRead);
  const readNotifications = notifications.filter((n) => n.isRead);

  const handleMarkAsRead = (id: string) => {
    markRead.mutate(Number(id));
  };

  const handleMarkAllRead = () => {
    if (unreadCount > 0) {
      markAllRead.mutate();
    }
  };

  const handleDelete = (notification: Notification) => {
    setNotificationToDelete(notification);
    setDeleteOpen(true);
  };

  const handleConfirmDelete = () => {
    if (notificationToDelete) {
      deleteNotif.mutate(notificationToDelete.id);
    }
  };

  const handleTabChange = (value: string) => {
    setTab(value);
    setPage(1);
  };

  const renderList = (items: Notification[]) => {
    if (items.length === 0) {
      return <EmptyState icon={<Icons.notification />} title={t('noNotifications')} />;
    }

    return (
      <div className='flex flex-col gap-3'>
        {items.map((notification) => {
          const card = toNotificationCard(notification);
          const safeHref = getSafeNotificationHref(notification.link);
          return (
            <div key={notification.id} className='group'>
              <NotificationCard
                id={card.id}
                title={card.title}
                body={card.body}
                status={card.status}
                createdAt={card.createdAt}
                markAsReadLabel={t('markRead')}
                onMarkAsRead={handleMarkAsRead}
                actions={
                  safeHref
                    ? [
                        {
                          id: 'view-link',
                          label: t('viewDetails'),
                          type: 'redirect' as const,
                          style: 'primary' as const
                        }
                      ]
                    : undefined
                }
                onAction={(notifId, actionId) => {
                  if (actionId === 'view-link' && safeHref) {
                    handleMarkAsRead(notifId);
                    router.push(safeHref);
                  }
                }}
              />
              <div className='flex items-center justify-between px-4 py-1'>
                <div className='flex gap-1'>
                  <Badge
                    variant={getDictVariant('notification_type', notification.type)}
                    className='text-[10px] h-5'
                  >
                    {getDictLabel('notification_type', notification.type)}
                  </Badge>
                  <Badge
                    variant={getDictVariant('notification_level', notification.level)}
                    className='text-[10px] h-5'
                  >
                    {getDictLabel('notification_level', notification.level)}
                  </Badge>
                </div>
                <Button
                  size='sm'
                  variant='ghost'
                  className='h-6 w-6 p-0 text-muted-foreground hover:text-destructive'
                  onClick={() => handleDelete(notification)}
                >
                  <Icons.trash className='h-3 w-3' />
                </Button>
              </div>
            </div>
          );
        })}
      </div>
    );
  };

  return (
    <>
      <AlertModal
        isOpen={deleteOpen}
        onClose={() => setDeleteOpen(false)}
        onConfirm={handleConfirmDelete}
        loading={deleteNotif.isPending}
      />
      <div className='flex flex-col gap-4'>
        <div className='flex items-center justify-end'>
          {unreadCount > 0 && (
            <Button
              variant='outline'
              size='sm'
              onClick={handleMarkAllRead}
              disabled={markAllRead.isPending}
            >
              {t('markAllRead')}
            </Button>
          )}
        </div>
        <Tabs value={tab} onValueChange={handleTabChange}>
          <TabsList>
            <TabsTrigger value='all'>
              {t('all')} ({total})
            </TabsTrigger>
            <TabsTrigger value='unread'>
              {t('unread')} ({unreadCount})
            </TabsTrigger>
            <TabsTrigger value='read'>
              {t('read')} ({readNotifications.length})
            </TabsTrigger>
          </TabsList>
          <TabsContent value='all' className='mt-4'>
            {renderList(notifications)}
          </TabsContent>
          <TabsContent value='unread' className='mt-4'>
            {renderList(unreadNotifications)}
          </TabsContent>
          <TabsContent value='read' className='mt-4'>
            {renderList(readNotifications)}
          </TabsContent>
        </Tabs>
        <div className='flex items-center justify-between gap-3'>
          <p className='text-muted-foreground text-sm'>
            {tc('pageOf', { current: page, total: pageCount })}
          </p>
          <div className='flex items-center gap-2'>
            <Button
              variant='outline'
              size='sm'
              onClick={() => setPage((current) => Math.max(1, current - 1))}
              disabled={page <= 1}
            >
              {tc('previous')}
            </Button>
            <Button
              variant='outline'
              size='sm'
              onClick={() => setPage((current) => Math.min(pageCount, current + 1))}
              disabled={page >= pageCount}
            >
              {tc('next')}
            </Button>
          </div>
        </div>
      </div>
    </>
  );
}
