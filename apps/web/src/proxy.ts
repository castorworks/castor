// ============================================================
// Route Guard Middleware
// ============================================================
// Protects /dashboard/* routes with the live RBAC authorization session.
// Redirects unauthenticated or unauthorized users before rendering a page.
// ============================================================

import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';
import type { MenuRoute, Navigation } from '@/features/menus/api/types';
import { getCastorApiUrl } from '@/lib/server-config';

type RouteGuardDecision =
  | { type: 'next' }
  | { type: 'redirect'; location: string; clearJwt?: boolean };

// Paths that do NOT require authentication
const PUBLIC_PATHS = [
  '/auth/sign-in',
  '/auth/sign-up',
  '/auth/reset-password',
  '/403',
  '/about',
  '/privacy-policy',
  '/terms-of-service',
  '/favicon.ico'
];

// Paths that require authentication
const PROTECTED_PREFIXES = ['/dashboard'];
const NAVIGATION_TIMEOUT_MS = 5000;

export function getRouteGuardDecision({
  pathname,
  search,
  hasJwt,
  jwtValid,
  routes
}: {
  pathname: string;
  search: string;
  hasJwt: boolean;
  jwtValid: boolean;
  routes: MenuRoute[];
}): RouteGuardDecision {
  const fullPath = `${pathname}${search}`;
  const isPublic = PUBLIC_PATHS.some((p) => pathname === p || pathname.startsWith(`${p}/`));

  if (isPublic) {
    return { type: 'next' };
  }

  if (pathname === '/') {
    if (hasJwt && jwtValid) {
      return {
        type: 'redirect',
        location:
          routes.find((route) => route.allowed && route.path === '/dashboard/overview')?.path ??
          routes.find((route) => route.allowed && route.path === '/dashboard/profile')?.path ??
          routes.find((route) => route.allowed)?.path ??
          '/403'
      };
    }

    return {
      type: 'redirect',
      location: `/auth/sign-in?redirect=${encodeURIComponent(fullPath)}`,
      clearJwt: hasJwt
    };
  }

  const isProtected = PROTECTED_PREFIXES.some((p) => pathname.startsWith(p));
  if (!isProtected) {
    return { type: 'next' };
  }

  if (!hasJwt || !jwtValid) {
    return {
      type: 'redirect',
      location: `/auth/sign-in?redirect=${encodeURIComponent(fullPath)}`,
      clearJwt: hasJwt
    };
  }

  if (pathname === '/dashboard')
    return { type: 'redirect', location: routes.find((route) => route.allowed)?.path ?? '/403' };
  const page = routes
    .filter((route) => pathname === route.path || pathname.startsWith(`${route.path}/`))
    .toSorted((a, b) => b.path.length - a.path.length)[0];
  if (!page?.allowed) return { type: 'redirect', location: '/403' };

  return { type: 'next' };
}

type AuthState =
  | { status: 'valid'; routes: MenuRoute[] }
  | { status: 'invalid' }
  | { status: 'unavailable' };

async function validateAuthState(token: string): Promise<AuthState> {
  let accessRes: Response;
  try {
    accessRes = await fetch(`${getCastorApiUrl()}/api/v1/account/navigation`, {
      method: 'GET',
      cache: 'no-store',
      headers: { Authorization: `Bearer ${token}` },
      signal: AbortSignal.timeout(NAVIGATION_TIMEOUT_MS)
    });
  } catch (error) {
    // Timeout, network failure or missing server config: the session may still
    // be valid, so do not log the user out — surface a server error instead.
    console.error('[proxy] navigation lookup failed', error);
    return { status: 'unavailable' };
  }

  if (accessRes.status === 401 || accessRes.status === 403) return { status: 'invalid' };
  if (!accessRes.ok) return { status: 'unavailable' };

  try {
    const accessBody = (await accessRes.json()) as { code?: number; data?: Navigation };
    if (accessBody.code !== 200 || !Array.isArray(accessBody.data?.routes))
      return { status: 'invalid' };
    return { status: 'valid', routes: accessBody.data.routes };
  } catch {
    return { status: 'unavailable' };
  }
}

function applyRouteGuardDecision(req: NextRequest, decision: RouteGuardDecision) {
  if (decision.type === 'next') {
    return NextResponse.next();
  }

  const response = NextResponse.redirect(new URL(decision.location, req.url));
  if (decision.clearJwt) {
    response.cookies.set('jwt', '', {
      maxAge: 0,
      path: '/',
      httpOnly: true,
      sameSite: 'lax'
    });
  }
  return response;
}

export async function proxy(req: NextRequest) {
  const { pathname } = req.nextUrl;

  // Skip middleware for:
  // - Next.js internal routes
  // - Static files
  // - API routes (auth is handled by API handlers)
  // - Health check endpoints
  if (
    pathname.startsWith('/_next') ||
    pathname.startsWith('/__next') ||
    pathname.startsWith('/api/') ||
    pathname.startsWith('/monitoring') ||
    pathname.includes('.')
  ) {
    return NextResponse.next();
  }

  const token = req.cookies.get('jwt')?.value;
  const hasJwt = Boolean(token);
  const shouldValidate =
    hasJwt && (pathname === '/' || PROTECTED_PREFIXES.some((p) => pathname.startsWith(p)));
  const authState: AuthState =
    shouldValidate && token ? await validateAuthState(token) : { status: 'invalid' };
  if (authState.status === 'unavailable') {
    return new NextResponse('Service temporarily unavailable', {
      status: 503,
      headers: { 'Retry-After': '5', 'Cache-Control': 'no-store' }
    });
  }
  const decision = getRouteGuardDecision({
    pathname,
    search: req.nextUrl.search,
    hasJwt,
    jwtValid: authState.status === 'valid',
    routes: authState.status === 'valid' ? authState.routes : []
  });

  return applyRouteGuardDecision(req, decision);
}

export const config = {
  matcher: [
    '/((?!_next|api/|monitoring|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)'
  ]
};
