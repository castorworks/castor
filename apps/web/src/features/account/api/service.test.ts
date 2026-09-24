import { beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient } from '@/lib/api-client';
import {
  bindAccountContact,
  changeExpiredPassword,
  resetAccountPassword,
  sendAccountContactCode,
  sendAuthCode
} from './service';

vi.mock('@/lib/api-client', () => ({
  apiClient: vi.fn(),
  apiUpload: vi.fn()
}));

vi.mock('@/lib/rsa', () => ({
  encryptPassword: (publicKey: string, plaintext: string) => `enc(${publicKey}:${plaintext})`
}));

const mockedApiClient = vi.mocked(apiClient);

function lastCall() {
  const call = mockedApiClient.mock.calls.at(-1);
  if (!call) throw new Error('apiClient was not called');
  return { endpoint: call[0], init: call[1] ?? {} };
}

describe('account contact service', () => {
  beforeEach(() => {
    mockedApiClient.mockReset();
    mockedApiClient.mockResolvedValue(undefined as never);
  });

  it('posts the contact code request', async () => {
    await sendAccountContactCode({ contactType: 'EMAIL', contact: 'user@example.com' });

    const { endpoint, init } = lastCall();
    expect(endpoint).toBe('/v1/account/contact/code');
    expect(init.method).toBe('POST');
    expect(JSON.parse(String(init.body))).toEqual({
      contactType: 'EMAIL',
      contact: 'user@example.com'
    });
  });

  it('binds the contact with the verification code', async () => {
    await bindAccountContact({ contactType: 'MOBILE', contact: '13800138000', code: '123456' });

    const { endpoint, init } = lastCall();
    expect(endpoint).toBe('/v1/account/contact');
    expect(init.method).toBe('PUT');
    expect(JSON.parse(String(init.body))).toEqual({
      contactType: 'MOBILE',
      contact: '13800138000',
      code: '123456'
    });
  });
});

describe('sendAuthCode', () => {
  beforeEach(() => {
    mockedApiClient.mockReset();
    mockedApiClient.mockResolvedValue(undefined as never);
  });

  it('defaults the purpose to auth', async () => {
    await sendAuthCode({ codeType: 'EMAIL', username: 'user@example.com' });

    expect(JSON.parse(String(lastCall().init.body))).toEqual({
      codeType: 'EMAIL',
      username: 'user@example.com',
      purpose: 'auth'
    });
  });

  it('keeps an explicit reset purpose', async () => {
    await sendAuthCode({ codeType: 'MOBILE', username: '13800138000', purpose: 'reset' });

    expect(JSON.parse(String(lastCall().init.body))).toEqual({
      codeType: 'MOBILE',
      username: '13800138000',
      purpose: 'reset'
    });
  });
});

describe('resetAccountPassword', () => {
  beforeEach(() => {
    mockedApiClient.mockReset();
  });

  it('encrypts the new password with the fetched public key', async () => {
    mockedApiClient.mockResolvedValueOnce({ publicKey: 'PEM' } as never);
    mockedApiClient.mockResolvedValueOnce(undefined as never);

    await resetAccountPassword({
      username: 'user@example.com',
      confirmCode: '123456',
      newPassword: 'secret123'
    });

    const { endpoint, init } = lastCall();
    expect(endpoint).toBe('/v1/account/password/reset');
    expect(init.method).toBe('PUT');
    expect(JSON.parse(String(init.body))).toEqual({
      username: 'user@example.com',
      confirmCode: '123456',
      newPassword: 'enc(PEM:secret123)'
    });
  });
});

describe('changeExpiredPassword', () => {
  beforeEach(() => {
    mockedApiClient.mockReset();
  });

  it('encrypts both passwords and calls the public expired-password endpoint', async () => {
    mockedApiClient.mockResolvedValueOnce({ publicKey: 'PEM' } as never);
    mockedApiClient.mockResolvedValueOnce(undefined as never);

    await changeExpiredPassword({
      username: 'carol',
      currentPassword: 'Old-password1',
      newPassword: 'New-password1',
      captchaId: 'cid',
      captchaCode: '1234'
    });

    const { endpoint, init } = lastCall();
    expect(endpoint).toBe('/v1/auth/password/expired');
    expect(init.method).toBe('PUT');
    expect(JSON.parse(String(init.body))).toEqual({
      username: 'carol',
      currentPassword: 'enc(PEM:Old-password1)',
      newPassword: 'enc(PEM:New-password1)',
      captchaId: 'cid',
      captchaCode: '1234'
    });
  });
});
