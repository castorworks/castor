'use client';

import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { toast } from 'sonner';
import { useAuthStore } from '@/stores/auth-store';
import { notificationKeys } from '../api/queries';

/** Stream endpoint (same origin; the Next.js rewrite forwards it to the API). */
export const NOTIFICATION_STREAM_URL = '/api/v1/account/notifications/stream';

/** Summary pushed with a `notification` event. */
export interface StreamedNotification {
  id: number;
  title: string;
  type: string;
  level: string;
  link: string;
}

/** Reconnect delay after the n-th consecutive failure: 2s, 4s, 8s … capped at 60s. */
export function reconnectDelay(failures: number): number {
  return Math.min(60_000, 2_000 * 2 ** Math.max(0, failures - 1));
}

/** Parse an SSE `data:` payload; malformed events are ignored. */
export function parseEvent<T>(data: string): T | null {
  try {
    return JSON.parse(data) as T;
  } catch {
    return null;
  }
}

const toastByLevel: Record<
  string,
  (message: string, options?: Parameters<typeof toast>[1]) => unknown
> = {
  SUCCESS: toast.success,
  WARNING: toast.warning,
  ERROR: toast.error
};

/**
 * Keeps the unread count live and announces new notifications while signed in.
 * The server ends the stream when the access token expires; on any drop we refresh
 * the token (the stream itself cannot) and reconnect with backoff.
 */
export function useNotificationStream(enabled: boolean, labels: { open: string }) {
  const queryClient = useQueryClient();
  const router = useRouter();
  const refreshToken = useAuthStore((s) => s.refreshToken);

  useEffect(() => {
    if (!enabled || typeof window === 'undefined' || typeof EventSource === 'undefined') return;
    let source: EventSource | null = null;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let failures = 0;
    let stopped = false;

    const connect = () => {
      source = new EventSource(NOTIFICATION_STREAM_URL, { withCredentials: true });
      source.addEventListener('open', () => {
        failures = 0;
      });
      source.addEventListener('unread', (event) => {
        const data = parseEvent<{ count: number }>((event as MessageEvent).data);
        if (data) queryClient.setQueryData(notificationKeys.unreadCount(), data);
      });
      source.addEventListener('notification', (event) => {
        const data = parseEvent<StreamedNotification>((event as MessageEvent).data);
        if (!data) return;
        void queryClient.invalidateQueries({ queryKey: [...notificationKeys.all, 'user'] });
        const show = toastByLevel[data.level] ?? toast.info;
        show(data.title, {
          action: data.link
            ? {
                label: labels.open,
                onClick: () =>
                  router.push(data.link.startsWith('/') ? data.link : '/dashboard/notifications')
              }
            : undefined
        });
      });
      source.addEventListener('error', () => {
        source?.close();
        source = null;
        if (stopped) return;
        failures += 1;
        // The usual cause is an expired access token: refresh it before reconnecting.
        timer = setTimeout(async () => {
          if (stopped) return;
          await refreshToken();
          if (!stopped) connect();
        }, reconnectDelay(failures));
      });
    };

    connect();
    return () => {
      stopped = true;
      clearTimeout(timer);
      source?.close();
    };
  }, [enabled, queryClient, router, refreshToken, labels.open]);
}
