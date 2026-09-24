import { describe, expect, it, vi } from 'vitest';
import { getRouteGuardDecision as decide, refreshSession, rewriteCookieHeader } from './proxy';

function getRouteGuardDecision(input: {
  pathname: string;
  search: string;
  hasJwt: boolean;
  jwtValid: boolean;
  permissions: string[];
}) {
  const paths: Record<string, string> = {
    '/api/v1/admin/audit-logs:GET': '/dashboard/audit-logs',
    '/api/v1/admin/dashboard/stats:GET': '/dashboard/overview',
    '/api/v1/admin/users:GET': '/dashboard/users',
    '/api/v1/admin/notifications:GET': '/dashboard/admin-notifications'
  };
  return decide({
    ...input,
    routes: [
      { path: '/dashboard/profile', allowed: true },
      ...Object.entries(paths).map(([permission, path]) => ({
        path,
        allowed: input.permissions.includes(permission)
      }))
    ]
  });
}

describe('getRouteGuardDecision', () => {
  it('redirects protected routes when a jwt cookie exists but validation fails', () => {
    expect(
      getRouteGuardDecision({
        pathname: '/dashboard/overview',
        search: '?tab=stats',
        hasJwt: true,
        jwtValid: false,
        permissions: []
      })
    ).toEqual({
      type: 'redirect',
      location: '/auth/sign-in?redirect=%2Fdashboard%2Foverview%3Ftab%3Dstats',
      clearJwt: true
    });
  });

  it('allows protected routes only when jwt validation succeeds', () => {
    expect(
      getRouteGuardDecision({
        pathname: '/dashboard/overview',
        search: '',
        hasJwt: true,
        jwtValid: true,
        permissions: ['/api/v1/admin/dashboard/stats:GET']
      })
    ).toEqual({
      type: 'next'
    });
  });

  it('redirects users without the route permission', () => {
    expect(
      getRouteGuardDecision({
        pathname: '/dashboard/users',
        search: '',
        hasJwt: true,
        jwtValid: true,
        permissions: []
      })
    ).toEqual({
      type: 'redirect',
      location: '/403'
    });
  });

  it('redirects authenticated non-admin users away from notification management', () => {
    expect(
      getRouteGuardDecision({
        pathname: '/dashboard/admin-notifications',
        search: '',
        hasJwt: true,
        jwtValid: true,
        permissions: []
      })
    ).toEqual({
      type: 'redirect',
      location: '/403'
    });
  });

  it('renders the public home page without touching the session', () => {
    for (const session of [
      { hasJwt: false, jwtValid: false },
      { hasJwt: true, jwtValid: false },
      { hasJwt: true, jwtValid: true }
    ]) {
      expect(
        getRouteGuardDecision({ pathname: '/', search: '', permissions: [], ...session })
      ).toEqual({ type: 'next' });
    }
  });

  it('does not treat every path as public because the home page is', () => {
    expect(
      getRouteGuardDecision({
        pathname: '/dashboard/users',
        search: '',
        hasJwt: false,
        jwtValid: false,
        permissions: []
      })
    ).toEqual({
      type: 'redirect',
      location: '/auth/sign-in?redirect=%2Fdashboard%2Fusers',
      clearJwt: false
    });
  });

  it('opens the overview from the console entry when it is allowed', () => {
    expect(
      getRouteGuardDecision({
        pathname: '/dashboard',
        search: '',
        hasJwt: true,
        jwtValid: true,
        permissions: ['/api/v1/admin/users:GET', '/api/v1/admin/dashboard/stats:GET']
      })
    ).toEqual({ type: 'redirect', location: '/dashboard/overview' });
  });

  it('uses the profile as the landing page when dashboard permission is absent', () => {
    expect(
      getRouteGuardDecision({
        pathname: '/dashboard',
        search: '',
        hasJwt: true,
        jwtValid: true,
        permissions: []
      })
    ).toEqual({
      type: 'redirect',
      location: '/dashboard/profile'
    });
  });

  it('allows users with a read permission to open management routes', () => {
    expect(
      getRouteGuardDecision({
        pathname: '/dashboard/audit-logs',
        search: '',
        hasJwt: true,
        jwtValid: true,
        permissions: ['/api/v1/admin/audit-logs:GET']
      })
    ).toEqual({
      type: 'next'
    });
  });

  it('does not infer access from a role name', () => {
    expect(
      getRouteGuardDecision({
        pathname: '/dashboard/users',
        search: '',
        hasJwt: true,
        jwtValid: true,
        permissions: ['/api/v1/admin/users:GET']
      })
    ).toEqual({
      type: 'next'
    });
  });
});

describe('dynamic menu route guard', () => {
  it('rejects disabled child pages even when the parent route is allowed', () => {
    expect(
      decide({
        pathname: '/dashboard/reports/private',
        search: '',
        hasJwt: true,
        jwtValid: true,
        routes: [
          { path: '/dashboard/reports', allowed: true },
          { path: '/dashboard/reports/private', allowed: false }
        ]
      })
    ).toEqual({ type: 'redirect', location: '/403' });
  });
  it('rejects unregistered pages', () => {
    expect(
      decide({
        pathname: '/dashboard/unknown',
        search: '',
        hasJwt: true,
        jwtValid: true,
        routes: []
      })
    ).toEqual({ type: 'redirect', location: '/403' });
  });
});

describe('expired session refresh', () => {
  it('keeps the jwt cookie when a refresh attempt was rejected', () => {
    // Another tab may have rotated the token a moment ago; clearing it would sign every tab out.
    expect(
      decide({
        pathname: '/dashboard/users',
        search: '',
        hasJwt: true,
        jwtValid: false,
        refreshFailed: true,
        routes: []
      })
    ).toEqual({
      type: 'redirect',
      location: '/auth/sign-in?redirect=%2Fdashboard%2Fusers',
      clearJwt: false
    });
  });

  it('exchanges the expired token with the double-submit CSRF token', async () => {
    const fetchImpl = vi.fn<typeof fetch>().mockResolvedValue(
      new Response(JSON.stringify({ code: 200, data: {} }), {
        status: 200,
        headers: [
          ['Set-Cookie', 'jwt=new-token; Path=/; Max-Age=604800; HttpOnly; SameSite=Lax'],
          ['Set-Cookie', 'csrf_token=new-csrf; Path=/; Max-Age=604800; SameSite=Lax']
        ]
      })
    );
    const refreshed = await refreshSession('old-token', 'old-csrf', fetchImpl);

    expect(refreshed).toEqual({
      token: 'new-token',
      setCookies: [
        'jwt=new-token; Path=/; Max-Age=604800; HttpOnly; SameSite=Lax',
        'csrf_token=new-csrf; Path=/; Max-Age=604800; SameSite=Lax'
      ]
    });
    const [url, init] = fetchImpl.mock.calls[0];
    expect(String(url)).toMatch(/\/api\/v1\/auth\/refresh-token$/);
    expect(init?.method).toBe('POST');
    expect(init?.headers).toEqual({
      Cookie: 'jwt=old-token; csrf_token=old-csrf',
      'X-CSRF-Token': 'old-csrf'
    });
  });

  it('reports a rejected refresh as no session', async () => {
    const fetchImpl = vi.fn<typeof fetch>().mockResolvedValue(new Response('{}', { status: 401 }));
    expect(await refreshSession('old-token', 'old-csrf', fetchImpl)).toBeNull();
  });

  it('passes the rotated cookies on to server components', () => {
    expect(
      rewriteCookieHeader(
        [
          { name: 'NEXT_LOCALE', value: 'en' },
          { name: 'jwt', value: 'old-token' },
          { name: 'csrf_token', value: 'old-csrf' }
        ],
        ['jwt=new-token; Path=/; HttpOnly', 'csrf_token=new-csrf; Path=/']
      )
    ).toBe('NEXT_LOCALE=en; jwt=new-token; csrf_token=new-csrf');
  });
});
