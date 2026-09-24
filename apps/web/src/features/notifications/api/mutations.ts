import { mutationOptions } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import {
  markNotificationRead,
  batchMarkNotificationsRead,
  markAllNotificationsRead,
  deleteNotification,
  createNotification,
  updateNotification,
  deleteAdminNotification,
  batchDeleteNotifications
} from './service';
import { notificationKeys } from './queries';
import type { CreateNotificationPayload, UpdateNotificationPayload } from './types';

// ============================================================
// User-side mutations
// ============================================================

export const markNotificationReadMutation = mutationOptions({
  mutationFn: (id: number) => markNotificationRead(id),
  onSuccess: () => {
    const qc = getQueryClient();
    qc.invalidateQueries({ queryKey: notificationKeys.all });
  }
});

export const batchMarkReadMutation = mutationOptions({
  mutationFn: (ids: number[]) => batchMarkNotificationsRead(ids),
  onSuccess: () => {
    const qc = getQueryClient();
    qc.invalidateQueries({ queryKey: notificationKeys.all });
  }
});

export const markAllReadMutation = mutationOptions({
  mutationFn: () => markAllNotificationsRead(),
  onSuccess: () => {
    const qc = getQueryClient();
    qc.invalidateQueries({ queryKey: notificationKeys.all });
  }
});

export const deleteNotificationMutation = mutationOptions({
  mutationFn: (id: number) => deleteNotification(id),
  onSuccess: () => {
    const qc = getQueryClient();
    qc.invalidateQueries({ queryKey: notificationKeys.all });
  }
});

// ============================================================
// Admin-side mutations
// ============================================================

export const createAdminNotificationMutation = mutationOptions({
  mutationFn: (data: CreateNotificationPayload) => createNotification(data),
  onSuccess: () => {
    const qc = getQueryClient();
    qc.invalidateQueries({ queryKey: notificationKeys.all });
  }
});

export const updateAdminNotificationMutation = mutationOptions({
  mutationFn: ({ id, data }: { id: number; data: UpdateNotificationPayload }) =>
    updateNotification(id, data),
  onSuccess: () => {
    const qc = getQueryClient();
    qc.invalidateQueries({ queryKey: notificationKeys.all });
  }
});

export const deleteAdminNotificationMutation = mutationOptions({
  mutationFn: (id: number) => deleteAdminNotification(id),
  onSuccess: () => {
    const qc = getQueryClient();
    qc.invalidateQueries({ queryKey: notificationKeys.all });
  }
});

export const batchDeleteAdminNotificationsMutation = mutationOptions({
  mutationFn: (ids: number[]) => batchDeleteNotifications(ids),
  onSuccess: () => {
    const qc = getQueryClient();
    qc.invalidateQueries({ queryKey: notificationKeys.all });
  }
});
