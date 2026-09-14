// ============================================================
// Notification types — aligned with castor backend DTOs
// ============================================================

export type NotificationType = 'SYSTEM' | 'ANNOUNCE' | 'MESSAGE' | 'ALERT' | 'TASK';
export type NotificationLevel = 'INFO' | 'SUCCESS' | 'WARNING' | 'ERROR';

export interface Notification {
  id: number;
  title: string;
  content: string;
  templateKey: string;
  templateData: string;
  type: NotificationType;
  level: NotificationLevel;
  link: string | null;
  extra: string | null;
  senderId: number;
  isGlobal: boolean;
  expireAt: string | null;
  isRead: boolean;
  readAt: string | null;
  isDismissed: boolean;
  dismissedAt: string | null;
  recipientCount: number;
  readCount: number;
  unreadCount: number;
  dismissedCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface NotificationRecipient {
  userId: number;
  isRead: boolean;
  readAt: string | null;
  isDismissed: boolean;
  dismissedAt: string | null;
  deliveredAt: string;
}

export interface NotificationFilters {
  page?: number;
  pageSize?: number;
  unreadOnly?: boolean;
}

export interface NotificationsPageResult {
  list: Notification[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface NotificationRecipientsPageResult {
  list: NotificationRecipient[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface UnreadCountResult {
  count: number;
}

export interface CreateNotificationPayload {
  title: string;
  content?: string;
  type?: NotificationType;
  level?: NotificationLevel;
  link?: string;
  extra?: string;
  isGlobal?: boolean;
  userIds?: number[];
  expireAt?: string;
}

export interface UpdateNotificationPayload {
  title?: string;
  content?: string;
  type?: NotificationType;
  level?: NotificationLevel;
  link?: string;
  extra?: string;
  isGlobal?: boolean;
  userIds?: number[];
  expireAt?: string;
}

export interface AdminNotificationFilters {
  page?: number;
  pageSize?: number;
  title?: string;
  type?: string;
  level?: string;
  sort?: string;
}

export interface NotificationRecipientFilters {
  page?: number;
  pageSize?: number;
}
