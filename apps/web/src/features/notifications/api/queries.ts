import { queryOptions } from '@tanstack/react-query';
import {
  getUserNotifications,
  getUnreadCount,
  getAdminNotifications,
  getAdminNotificationRecipients
} from './service';
import type {
  NotificationFilters,
  AdminNotificationFilters,
  NotificationRecipientFilters
} from './types';

function normalizeNotificationFilters(filters: NotificationFilters): NotificationFilters {
  return {
    ...(filters.page && { page: filters.page }),
    ...(filters.pageSize && { pageSize: filters.pageSize }),
    ...(filters.unreadOnly && { unreadOnly: true })
  };
}

export const notificationKeys = {
  all: ['notifications'] as const,
  user: (filters: NotificationFilters) =>
    [...notificationKeys.all, 'user', normalizeNotificationFilters(filters)] as const,
  unreadCount: () => [...notificationKeys.all, 'unread-count'] as const,
  admin: (filters: AdminNotificationFilters) =>
    [...notificationKeys.all, 'admin', filters] as const,
  recipients: (id: number, filters: NotificationRecipientFilters) =>
    [...notificationKeys.all, 'admin', id, 'recipients', filters] as const
};

export function userNotificationsQueryOptions(
  filters: NotificationFilters,
  requestInit?: RequestInit,
  options?: { enabled?: boolean }
) {
  const normalizedFilters = normalizeNotificationFilters(filters);
  return queryOptions({
    queryKey: notificationKeys.user(normalizedFilters),
    queryFn: () => getUserNotifications(normalizedFilters, requestInit),
    enabled: options?.enabled
  });
}

export function unreadCountQueryOptions(
  requestInit?: RequestInit,
  options?: { enabled?: boolean }
) {
  return queryOptions({
    queryKey: notificationKeys.unreadCount(),
    queryFn: () => getUnreadCount(requestInit),
    enabled: options?.enabled
  });
}

export function adminNotificationsQueryOptions(
  filters: AdminNotificationFilters,
  requestInit?: RequestInit
) {
  return queryOptions({
    queryKey: notificationKeys.admin(filters),
    queryFn: () => getAdminNotifications(filters, requestInit)
  });
}

export function adminNotificationRecipientsQueryOptions(
  id: number,
  filters: NotificationRecipientFilters,
  requestInit?: RequestInit
) {
  return queryOptions({
    queryKey: notificationKeys.recipients(id, filters),
    queryFn: () => getAdminNotificationRecipients(id, filters, requestInit),
    enabled: id > 0
  });
}
