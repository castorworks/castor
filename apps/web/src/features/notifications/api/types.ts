// ============================================================
// Notification types — aligned with castor backend DTOs
// ============================================================

import type { AssetAttachment } from '@/features/assets/api/types';

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
  /** 私有附件：只有能看到这条通知的用户可经 `notificationAttachmentUrl` 下载。 */
  attachments: AssetAttachment[];
  /** Also emailed to recipients with a verified email who did not mute notification emails. */
  sendEmail: boolean;
  /** Admin views only, when `sendEmail` is on. */
  emailStats?: NotificationEmailStats;
  createdAt: string;
  updatedAt: string;
}

/** 提交给后端的附件：对象键 + 收件人看到的文件名。 */
export interface NotificationAttachmentPayload {
  objectKey: string;
  name: string;
}

/** 单条通知最多携带的附件数，与后端 `dto.NotificationAttachmentsMax` 一致。 */
export const NOTIFICATION_ATTACHMENTS_MAX = 10;

/** 通知模块自己的附件上传路由：能发通知的人不必拥有资产库权限。 */
export const NOTIFICATION_ATTACHMENT_UPLOAD_PATH = '/v1/admin/notifications/attachments';
export const NOTIFICATION_ATTACHMENT_UPLOAD_PERMISSION =
  '/api/v1/admin/notifications/attachments:POST';

export const NOTIFICATION_ATTACHMENT_ADMIN_DOWNLOAD_PERMISSION =
  '/api/v1/admin/notifications/:id/attachments/:objectKey:GET';

export interface NotificationRecipient {
  userId: number;
  isRead: boolean;
  readAt: string | null;
  isDismissed: boolean;
  dismissedAt: string | null;
  deliveredAt: string;
  /** Status of the email to this recipient (`notification_email_status`); absent when none was queued. */
  emailStatus?: string;
}

/** Email delivery counts of a notification with `sendEmail`. */
export interface NotificationEmailStats {
  pending: number;
  sent: number;
  failed: number;
  skipped: number;
}

/** Delivery channels available besides the in-app inbox. */
export interface NotificationChannels {
  /** SMTP is configured, so notifications can also be emailed. */
  email: boolean;
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
  attachments?: NotificationAttachmentPayload[];
  sendEmail?: boolean;
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
  attachments?: NotificationAttachmentPayload[];
  /** Turning it on (or changing recipients) emails the recipients not emailed yet. */
  sendEmail?: boolean;
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
