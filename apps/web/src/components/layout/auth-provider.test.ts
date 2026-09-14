import { describe, expect, it } from 'vitest';
import { shouldFetchUserInfo } from './auth-provider';

describe('shouldFetchUserInfo', () => {
  it('skips public pages that do not need an account bootstrap request', () => {
    expect(shouldFetchUserInfo('/auth/sign-in')).toBe(false);
    expect(shouldFetchUserInfo('/about')).toBe(false);
  });

  it('allows protected dashboard pages to bootstrap account state', () => {
    expect(shouldFetchUserInfo('/dashboard/notifications')).toBe(true);
  });
});
