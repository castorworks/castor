import { describe, expect, it } from 'vitest';
import { getRouteGuardDecision as decide } from './proxy';

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

  it('uses the profile as the landing page when dashboard permission is absent', () => {
    expect(
      getRouteGuardDecision({
        pathname: '/',
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
