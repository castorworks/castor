import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  CastorApiError,
  apiClient,
  buildApiUrl,
  buildDefaultHeaders,
  getAuthRecoveryPolicy,
  readCastorResponse
} from './api-client';

describe('api-client auth recovery', () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it('does not refresh or redirect for login credential failures', () => {
    expect(getAuthRecoveryPolicy('/v1/auth/login?method=password')).toEqual({
      refresh: false,
      redirectOnFailure: false
    });
  });

  it('does not refresh or redirect for expired-password changes', () => {
    expect(getAuthRecoveryPolicy('/v1/auth/password/expired')).toEqual({
      refresh: false,
      redirectOnFailure: false
    });
  });

  it('refreshes protected API requests and redirects when recovery fails', () => {
    expect(getAuthRecoveryPolicy('/v1/account/info')).toEqual({
      refresh: true,
      redirectOnFailure: true
    });
  });

  it('keeps protected client requests pending after redirecting to sign-in', async () => {
    const location = {
      pathname: '/dashboard/notifications',
      search: '',
      href: 'http://localhost:3000/dashboard/notifications'
    };

    vi.stubGlobal('window', { location });
    vi.stubGlobal('document', { cookie: '' });
    vi.stubGlobal('navigator', { language: 'zh-CN' });
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce(
          new Response(JSON.stringify({ code: 401, data: null, message: '没有认证' }), {
            status: 401,
            statusText: 'Unauthorized'
          })
        )
        .mockResolvedValueOnce(
          new Response(JSON.stringify({ code: 401, data: null, message: '没有认证' }), {
            status: 401,
            statusText: 'Unauthorized'
          })
        )
    );

    const request = apiClient('/v1/account/notifications');
    await Promise.resolve();
    await Promise.resolve();

    const result = await Promise.race([
      request.then(
        () => 'resolved',
        () => 'rejected'
      ),
      new Promise<'pending'>((resolve) => setTimeout(() => resolve('pending'), 0))
    ]);

    expect(result).toBe('pending');
    expect(location.href).toBe('/auth/sign-in?redirect=%2Fdashboard%2Fnotifications');
  });

  it('keeps protected client requests pending when the browser is already on sign-in', async () => {
    vi.stubGlobal('window', {
      location: {
        pathname: '/auth/sign-in',
        search: '?redirect=%2Fdashboard%2Fnotifications',
        href: 'http://localhost:3000/auth/sign-in?redirect=%2Fdashboard%2Fnotifications'
      }
    });
    vi.stubGlobal('document', { cookie: '' });
    vi.stubGlobal('navigator', { language: 'zh-CN' });
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce(
          new Response(JSON.stringify({ code: 401, data: null, message: '没有认证' }), {
            status: 401,
            statusText: 'Unauthorized'
          })
        )
        .mockResolvedValueOnce(
          new Response(JSON.stringify({ code: 401, data: null, message: '没有认证' }), {
            status: 401,
            statusText: 'Unauthorized'
          })
        )
    );

    const request = apiClient('/v1/account/notifications');
    await Promise.resolve();
    await Promise.resolve();

    const result = await Promise.race([
      request.then(
        () => 'resolved',
        () => 'rejected'
      ),
      new Promise<'pending'>((resolve) => setTimeout(() => resolve('pending'), 0))
    ]);

    expect(result).toBe('pending');
  });

  it('recovers protected requests when castor returns an auth body with a non-401 HTTP status', async () => {
    const location = {
      pathname: '/dashboard/notifications',
      search: '',
      href: 'http://localhost:3000/dashboard/notifications'
    };

    vi.stubGlobal('window', { location });
    vi.stubGlobal('document', { cookie: '' });
    vi.stubGlobal('navigator', { language: 'zh-CN' });
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValueOnce(
          new Response(
            JSON.stringify({ code: 500, errorCode: 2000, data: null, message: '没有认证' }),
            {
              status: 500,
              statusText: 'Internal Server Error'
            }
          )
        )
        .mockResolvedValueOnce(
          new Response(JSON.stringify({ code: 401, data: null, message: '没有认证' }), {
            status: 401,
            statusText: 'Unauthorized'
          })
        )
    );

    const request = apiClient('/v1/account/notifications');
    await Promise.resolve();
    await Promise.resolve();

    const result = await Promise.race([
      request.then(
        () => 'resolved',
        () => 'rejected'
      ),
      new Promise<'pending'>((resolve) => setTimeout(() => resolve('pending'), 0))
    ]);

    expect(result).toBe('pending');
    expect(location.href).toBe('/auth/sign-in?redirect=%2Fdashboard%2Fnotifications');
  });
});

describe('readCastorResponse', () => {
  it('exposes the business error code of failed responses', async () => {
    const response = new Response(
      JSON.stringify({ code: 400, errorCode: 2002, data: null, message: 'captcha required' }),
      { status: 400 }
    );
    const error = await readCastorResponse(response).catch((e: unknown) => e);
    expect(error).toBeInstanceOf(CastorApiError);
    expect(error).toMatchObject({ code: 400, httpStatus: 400, errorCode: 2002 });
  });

  it('preserves backend error messages for non-2xx castor responses', async () => {
    const response = new Response(
      JSON.stringify({
        code: 401,
        data: null,
        message: 'invalid token'
      }),
      { status: 401, statusText: 'Unauthorized' }
    );

    await expect(readCastorResponse(response)).rejects.toMatchObject({
      name: 'CastorApiError',
      message: 'invalid token',
      code: 401,
      httpStatus: 401
    } satisfies Partial<CastorApiError>);
  });
});

describe('buildApiUrl', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('uses the castor backend base URL for server-side relative endpoints', () => {
    vi.stubEnv('CASTOR_API_URL', 'http://api.internal:1234');
    expect(buildApiUrl('/v1/admin/users?page=1')).toBe(
      'http://api.internal:1234/api/v1/admin/users?page=1'
    );
  });

  it('falls back to the local backend in development', () => {
    vi.stubEnv('CASTOR_API_URL', '');
    expect(buildApiUrl('/v1/admin/users?page=1')).toBe(
      'http://localhost:1234/api/v1/admin/users?page=1'
    );
  });
});

describe('buildDefaultHeaders', () => {
  it('includes Accept-Language so backend i18n can pick the client locale', () => {
    const headers = buildDefaultHeaders('zh-CN');

    expect(headers['Accept-Language']).toBe('zh-CN');
  });
});
