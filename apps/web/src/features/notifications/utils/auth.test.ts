import { describe, expect, it } from 'vitest';
import { shouldEnableNotificationQueries } from './auth';

describe('shouldEnableNotificationQueries', () => {
  it('waits until account bootstrap has completed', () => {
    expect(shouldEnableNotificationQueries(false, true)).toBe(false);
  });

  it('skips notification queries for unauthenticated users', () => {
    expect(shouldEnableNotificationQueries(false, false)).toBe(false);
  });

  it('enables notification queries for authenticated users after bootstrap', () => {
    expect(shouldEnableNotificationQueries(true, false)).toBe(true);
  });
});
