// ============================================================
// Notification Service — Data Access Layer
// ============================================================
// User-side: /api/v1/account/notifications
// Admin-side: /api/v1/admin/notifications
// ============================================================

import { apiClient, type CastorListResponse } from '@/lib/api-client';
import { buildCastorOrder } from '@/lib/castor-query';
import type {
  NotificationChannels,
  Notification,
  NotificationFilters,
  NotificationsPageResult,
  NotificationRecipient,
  NotificationRecipientFilters,
  NotificationRecipientsPageResult,
  UnreadCountResult,
  CreateNotificationPayload,
  UpdateNotificationPayload,
  AdminNotificationFilters
} from './types';

const ADMIN_NOTIFICATION_ORDER_FIELDS: Record<string, string> = {
  title: 'title',
  type: 'type',
  level: 'level',
  createdAt: 'created_at',
  expireAt: 'expire_at'
};

function toNotificationsPageResult(
  res: CastorListResponse<Notification>,
  filters: NotificationFilters | AdminNotificationFilters
): NotificationsPageResult {
  const page = res.page ?? filters.page ?? 1;
  const pageSize = res.pageSize ?? filters.pageSize ?? 10;
  return {
    list: (res.list ?? []).map((item) => ({ ...item, attachments: item.attachments ?? [] })),
    total: res.total ?? 0,
    page,
    pageSize,
    totalPages: res.totalPages ?? Math.ceil((res.total ?? 0) / pageSize)
  };
}

/**
 * Download URL of a notification attachment for the current user. Attachments are
 * private assets: the backend serves one only if the notification is visible to the
 * caller and the file really belongs to it.
 */
export function notificationAttachmentUrl(notificationId: number, objectKey: string): string {
  return `/api/v1/account/notifications/${notificationId}/attachments/${encodeURIComponent(objectKey)}`;
}

/** 管理端下载已保存到通知上的附件（编辑表单用），同样把 objectKey 限定在路径的一段里。 */
export function adminNotificationAttachmentUrl(notificationId: number, objectKey: string): string {
  return `/api/v1/admin/notifications/${notificationId}/attachments/${encodeURIComponent(objectKey)}`;
}

// ============================================================
// User-side notifications
// ============================================================

/** Fetch paginated user notifications */
export async function getUserNotifications(
  filters: NotificationFilters,
  options?: RequestInit
): Promise<NotificationsPageResult> {
  const params = new URLSearchParams();
  if (filters.page) params.set('page', String(filters.page));
  if (filters.pageSize) params.set('pageSize', String(filters.pageSize));
  if (filters.unreadOnly) params.set('unreadOnly', String(filters.unreadOnly));

  const query = params.toString();
  const res = await apiClient<CastorListResponse<Notification>>(
    `/v1/account/notifications${query ? `?${query}` : ''}`,
    options
  );
  return toNotificationsPageResult(res, filters);
}

/** Fetch unread notification count */
export async function getUnreadCount(options?: RequestInit): Promise<UnreadCountResult> {
  return apiClient<UnreadCountResult>('/v1/account/notifications/unread-count', options);
}

/** Mark a single notification as read */
export async function markNotificationRead(id: number): Promise<void> {
  await apiClient<void>(`/v1/account/notifications/${id}/read`, { method: 'PUT' });
}

/** Mark multiple notifications as read */
export async function batchMarkNotificationsRead(ids: number[]): Promise<void> {
  await apiClient<void>('/v1/account/notifications/batch-read', {
    method: 'PUT',
    body: JSON.stringify({ ids })
  });
}

/** Mark all notifications as read */
export async function markAllNotificationsRead(): Promise<void> {
  await apiClient<void>('/v1/account/notifications/read-all', { method: 'PUT' });
}

/** Delete a notification */
export async function deleteNotification(id: number): Promise<void> {
  await apiClient<void>(`/v1/account/notifications/${id}`, { method: 'DELETE' });
}

// ============================================================
// Admin-side notifications
// ============================================================

/** Fetch paginated admin notification list */
export async function getAdminNotifications(
  filters: AdminNotificationFilters,
  options?: RequestInit
): Promise<NotificationsPageResult> {
  const params = new URLSearchParams();
  if (filters.page) params.set('page', String(filters.page));
  if (filters.pageSize) params.set('pageSize', String(filters.pageSize));
  if (filters.title) {
    params.set('searchText', filters.title);
    params.set('searchFields', 'title,content');
  }
  if (filters.type) {
    if (filters.type.includes(',')) {
      params.set('type-in', filters.type);
    } else {
      params.set('type-eq', filters.type);
    }
  }
  if (filters.level) {
    if (filters.level.includes(',')) {
      params.set('level-in', filters.level);
    } else {
      params.set('level-eq', filters.level);
    }
  }
  const order = buildCastorOrder(filters.sort, ADMIN_NOTIFICATION_ORDER_FIELDS);
  if (order) params.set('order', order);

  const query = params.toString();
  const res = await apiClient<CastorListResponse<Notification>>(
    `/v1/admin/notifications${query ? `?${query}` : ''}`,
    options
  );
  return toNotificationsPageResult(res, filters);
}

/** Get a single notification detail (admin) */
export async function getAdminNotification(
  id: number,
  options?: RequestInit
): Promise<Notification> {
  return apiClient<Notification>(`/v1/admin/notifications/${id}`, options);
}

/** Fetch recipients for a notification (admin) */
export async function getAdminNotificationRecipients(
  id: number,
  filters: NotificationRecipientFilters,
  options?: RequestInit
): Promise<NotificationRecipientsPageResult> {
  const params = new URLSearchParams();
  if (filters.page) params.set('page', String(filters.page));
  if (filters.pageSize) params.set('pageSize', String(filters.pageSize));

  const query = params.toString();
  const res = await apiClient<CastorListResponse<NotificationRecipient>>(
    `/v1/admin/notifications/${id}/recipients${query ? `?${query}` : ''}`,
    options
  );
  return {
    list: res.list ?? [],
    total: res.total ?? 0,
    page: res.page ?? filters.page ?? 1,
    pageSize: res.pageSize ?? filters.pageSize ?? 10,
    totalPages:
      res.totalPages ?? Math.ceil((res.total ?? 0) / (res.pageSize ?? filters.pageSize ?? 10))
  };
}

/** Create a notification (admin) */
export async function createNotification(data: CreateNotificationPayload): Promise<Notification> {
  return apiClient<Notification>('/v1/admin/notifications', {
    method: 'POST',
    body: JSON.stringify(data)
  });
}

/** Update a notification (admin) */
export async function updateNotification(
  id: number,
  data: UpdateNotificationPayload
): Promise<Notification> {
  return apiClient<Notification>(`/v1/admin/notifications/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data)
  });
}

/** Delete a notification (admin) */
export async function deleteAdminNotification(id: number): Promise<void> {
  await apiClient<void>(`/v1/admin/notifications/${id}`, { method: 'DELETE' });
}

/** Batch delete notifications (admin) */
export async function batchDeleteNotifications(ids: number[]): Promise<void> {
  await apiClient<void>('/v1/admin/notifications/batch/delete', {
    method: 'POST',
    body: JSON.stringify({ ids })
  });
}

/** Delivery channels available for new notifications */
export async function getNotificationChannels(): Promise<NotificationChannels> {
  return apiClient<NotificationChannels>('/v1/admin/notifications/channels');
}
