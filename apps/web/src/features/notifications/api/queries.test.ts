import { describe, expect, it } from 'vitest';
import {
  notificationKeys,
  unreadCountQueryOptions,
  userNotificationsQueryOptions
} from './queries';

describe('notificationKeys', () => {
  it('uses the same user notification key when unreadOnly is omitted or false', () => {
    expect(notificationKeys.user({ page: 1, pageSize: 10 })).toEqual(
      notificationKeys.user({ page: 1, pageSize: 10, unreadOnly: false })
    );
  });

  it('keeps unread-only user notification queries distinct', () => {
    expect(notificationKeys.user({ page: 1, pageSize: 10, unreadOnly: true })).not.toEqual(
      notificationKeys.user({ page: 1, pageSize: 10 })
    );
  });

  it('passes enabled flags through to user notification query options', () => {
    expect(
      userNotificationsQueryOptions({ page: 1, pageSize: 10 }, undefined, {
        enabled: false
      }).enabled
    ).toBe(false);
    expect(unreadCountQueryOptions(undefined, { enabled: false }).enabled).toBe(false);
  });
});
