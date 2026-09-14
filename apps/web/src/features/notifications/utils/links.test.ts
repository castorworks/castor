import { describe, expect, it } from 'vitest';
import { getSafeNotificationHref } from './links';

describe('getSafeNotificationHref', () => {
  it('allows dashboard-relative notification links', () => {
    expect(getSafeNotificationHref('/dashboard/users?tab=active')).toBe(
      '/dashboard/users?tab=active'
    );
  });

  it('rejects external, protocol-relative, script, and non-dashboard links', () => {
    expect(getSafeNotificationHref('https://example.com/dashboard')).toBeNull();
    expect(getSafeNotificationHref('//example.com/dashboard')).toBeNull();
    expect(getSafeNotificationHref('javascript:alert(1)')).toBeNull();
    expect(getSafeNotificationHref('/announcements/1')).toBeNull();
  });
});
