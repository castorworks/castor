import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/lib/api-client';
import {
  disableTwoFactor,
  enableTwoFactor,
  recoveryCodesFile,
  regenerateRecoveryCodes
} from './two-factor';

vi.mock('@/lib/api-client', () => ({ apiClient: vi.fn() }));
vi.mock('@/features/users/lib/rsa', () => ({
  encryptPassword: vi.fn(async (plain: string) => `encrypted:${plain}`)
}));

const mocked = vi.mocked(apiClient);

describe('two-factor service', () => {
  beforeEach(() => mocked.mockReset());

  it('trims the code when enabling', async () => {
    mocked.mockResolvedValueOnce({ recoveryCodes: [] });
    await enableTwoFactor(' 123456 ');
    expect(mocked).toHaveBeenCalledWith('/v1/account/totp/enable', {
      method: 'POST',
      body: JSON.stringify({ code: '123456' })
    });
  });

  it('encrypts the current password when disabling, and sends none for password-less accounts', async () => {
    mocked.mockResolvedValue(undefined as never);
    await disableTwoFactor('Secret-1', '123456');
    expect(mocked.mock.calls[0][1]).toEqual({
      method: 'POST',
      body: JSON.stringify({ password: 'encrypted:Secret-1', code: '123456' })
    });
    await disableTwoFactor('', 'abcd-efgh-jkmn');
    expect(JSON.parse(String(mocked.mock.calls[1][1]?.body)).password).toBe('');
  });

  it('regenerates recovery codes with a code', async () => {
    mocked.mockResolvedValueOnce({ recoveryCodes: ['a'] });
    await expect(regenerateRecoveryCodes('123456')).resolves.toEqual({ recoveryCodes: ['a'] });
    expect(mocked.mock.calls[0][0]).toBe('/v1/account/totp/recovery-codes');
  });

  it('formats a recovery codes file', () => {
    expect(recoveryCodesFile(['aaaa-bbbb-cccc', 'dddd-eeee-ffff'], 'Castor')).toBe(
      'Castor\n\naaaa-bbbb-cccc\ndddd-eeee-ffff\n'
    );
  });
});
