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
import {
  buildContentSecurityPolicy,
  buildStaticSecurityHeaders,
  type SecurityHeaderOptions
} from '@/lib/security-headers';

type RouteGuardDecision =
  | { type: 'next' }
  | { type: 'redirect'; location: string; clearJwt?: boolean };

// Paths that do NOT require authentication
const PUBLIC_PATHS = [
  '/',
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

// `/dashboard` is the console entry (sign-in default, site header): prefer the
// overview, then the profile every account has, then any page the user may open.
function dashboardLandingPath(routes: MenuRoute[]): string {
  const allowed = routes.filter((route) => route.allowed);
  return (
    allowed.find((route) => route.path === '/dashboard/overview')?.path ??
    allowed.find((route) => route.path === '/dashboard/profile')?.path ??
    allowed[0]?.path ??
    '/403'
  );
}

export function getRouteGuardDecision({
  pathname,
  search,
  hasJwt,
  jwtValid,
  refreshFailed = false,
  routes
}: {
  pathname: string;
  search: string;
  hasJwt: boolean;
  jwtValid: boolean;
  /**
   * A refresh of an expired token was attempted and rejected. The jwt cookie is then
   * left alone: a concurrent request (another tab restoring after a browser restart)
   * may have rotated it a moment ago, and clearing it would sign every tab out.
   */
  refreshFailed?: boolean;
  routes: MenuRoute[];
}): RouteGuardDecision {
  const fullPath = `${pathname}${search}`;
  const isPublic = PUBLIC_PATHS.some(
    (p) => pathname === p || (p !== '/' && pathname.startsWith(`${p}/`))
  );

  if (isPublic) {
    return { type: 'next' };
  }

  const isProtected = PROTECTED_PREFIXES.some((p) => pathname.startsWith(p));
  if (!isProtected) {
    return { type: 'next' };
  }

  if (!hasJwt || !jwtValid) {
    return {
      type: 'redirect',
      location: `/auth/sign-in?redirect=${encodeURIComponent(fullPath)}`,
      clearJwt: hasJwt && !refreshFailed
    };
  }

  if (pathname === '/dashboard')
    return { type: 'redirect', location: dashboardLandingPath(routes) };
  const page = routes
    .filter((route) => pathname === route.path || pathname.startsWith(`${route.path}/`))
    .toSorted((a, b) => b.path.length - a.path.length)[0];
  if (!page?.allowed) return { type: 'redirect', location: '/403' };

  return { type: 'next' };
}

type AuthState =
  | { status: 'valid'; routes: MenuRoute[] }
  // The access token expired (401) but may still be refreshable within the refresh window.
  | { status: 'expired' }
  // Revoked, blacklisted or otherwise unusable (403 or malformed).
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

  if (accessRes.status === 401) return { status: 'expired' };
  if (accessRes.status === 403) return { status: 'invalid' };
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

export interface RefreshedSession {
  token: string;
  /** Raw Set-Cookie headers of the refresh response (rotated jwt + csrf_token). */
  setCookies: string[];
}

/**
 * Exchanges an expired access token for a new one on behalf of a full page request.
 * Without this, a user returning after the access token lifetime (2h by default) is
 * sent to sign-in even though the session is still within its refresh window, which
 * also makes "remember me" meaningless. Mirrors the browser refresh: the jwt cookie
 * plus the double-submit CSRF token.
 */
export async function refreshSession(
  token: string,
  csrfToken: string,
  fetchImpl: typeof fetch = fetch
): Promise<RefreshedSession | null> {
  let res: Response;
  try {
    res = await fetchImpl(`${getCastorApiUrl()}/api/v1/auth/refresh-token`, {
      method: 'POST',
      cache: 'no-store',
      headers: {
        Cookie: `jwt=${token}; csrf_token=${csrfToken}`,
        'X-CSRF-Token': csrfToken
      },
      signal: AbortSignal.timeout(NAVIGATION_TIMEOUT_MS)
    });
  } catch (error) {
    console.error('[proxy] token refresh failed', error);
    return null;
  }
  if (!res.ok) return null;
  const setCookies = res.headers.getSetCookie();
  const rotated = setCookies
    .map((cookie) => /^jwt=([^;]*)/.exec(cookie)?.[1])
    .find((value) => Boolean(value));
  return rotated ? { token: rotated, setCookies } : null;
}

/** Rebuilds the Cookie request header with the rotated values, for server components. */
export function rewriteCookieHeader(
  cookies: { name: string; value: string }[],
  setCookies: string[]
): string {
  const values = new Map(cookies.map((cookie) => [cookie.name, cookie.value]));
  for (const header of setCookies) {
    const match = /^([^=;\s]+)=([^;]*)/.exec(header);
    if (match) values.set(match[1], match[2]);
  }
  return Array.from(values, ([name, value]) => `${name}=${value}`).join('; ');
}

// Fresh base64 nonce per request; used to stamp Next's inline scripts and the
// production CSP so 'unsafe-inline' can be dropped for scripts.
function generateNonce(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  let binary = '';
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary);
}

function securityHeaderOptions(nonce: string): SecurityHeaderOptions {
  return {
    isProduction: process.env.NODE_ENV === 'production',
    publicApiUrl: process.env.NEXT_PUBLIC_API_URL,
    sentryDsn:
      process.env.NEXT_PUBLIC_SENTRY_DISABLED === 'true'
        ? undefined
        : process.env.NEXT_PUBLIC_SENTRY_DSN,
    nonce
  };
}

// next.config `headers()` does not run for responses the middleware terminates
// itself (redirects, 503), so re-apply the static headers there.
function applyStaticSecurityHeaders(response: NextResponse, options: SecurityHeaderOptions) {
  for (const header of buildStaticSecurityHeaders(options)) {
    response.headers.set(header.key, header.value);
  }
}

function applyRouteGuardDecision(
  req: NextRequest,
  decision: RouteGuardDecision,
  options: SecurityHeaderOptions,
  csp: string,
  refreshed: RefreshedSession | null
) {
  if (decision.type === 'next') {
    // Forward the CSP (with nonce) as a request header so Next stamps the nonce
    // on its own inline bootstrap/flight scripts, and echo it on the response.
    // The static headers for this passthrough come from next.config `headers()`.
    // `x-nonce` lets the root layout stamp its own inline theme script.
    const requestHeaders = new Headers(req.headers);
    requestHeaders.set('Content-Security-Policy', csp);
    requestHeaders.set('x-nonce', options.nonce ?? '');
    // Server components of this request must already see the rotated token: the
    // previous one is blacklisted by the refresh.
    if (refreshed) {
      requestHeaders.set('cookie', rewriteCookieHeader(req.cookies.getAll(), refreshed.setCookies));
    }
    const response = NextResponse.next({ request: { headers: requestHeaders } });
    response.headers.set('Content-Security-Policy', csp);
    appendSetCookies(response, refreshed);
    return response;
  }

  const response = NextResponse.redirect(new URL(decision.location, req.url));
  appendSetCookies(response, refreshed);
  if (decision.clearJwt) {
    response.cookies.set('jwt', '', {
      maxAge: 0,
      path: '/',
      httpOnly: true,
      sameSite: 'lax'
    });
  }
  response.headers.set('Content-Security-Policy', csp);
  applyStaticSecurityHeaders(response, options);
  return response;
}

function appendSetCookies(response: NextResponse, refreshed: RefreshedSession | null) {
  for (const cookie of refreshed?.setCookies ?? []) {
    response.headers.append('Set-Cookie', cookie);
  }
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

  const nonce = generateNonce();
  const options = securityHeaderOptions(nonce);
  const csp = buildContentSecurityPolicy(options);

  const token = req.cookies.get('jwt')?.value;
  const hasJwt = Boolean(token);
  const shouldValidate = hasJwt && PROTECTED_PREFIXES.some((p) => pathname.startsWith(p));
  let authState: AuthState =
    shouldValidate && token ? await validateAuthState(token) : { status: 'invalid' };
  let refreshed: RefreshedSession | null = null;
  let refreshFailed = false;
  const csrfToken = req.cookies.get('csrf_token')?.value;
  if (authState.status === 'expired' && token && csrfToken) {
    refreshed = await refreshSession(token, csrfToken);
    refreshFailed = refreshed === null;
    authState = refreshed ? await validateAuthState(refreshed.token) : { status: 'invalid' };
  }
  if (authState.status === 'unavailable') {
    const response = new NextResponse('Service temporarily unavailable', {
      status: 503,
      headers: { 'Retry-After': '5', 'Cache-Control': 'no-store' }
    });
    response.headers.set('Content-Security-Policy', csp);
    applyStaticSecurityHeaders(response, options);
    // The refresh already retired the old token; hand over the new one regardless.
    appendSetCookies(response, refreshed);
    return response;
  }
  const decision = getRouteGuardDecision({
    pathname,
    search: req.nextUrl.search,
    hasJwt,
    jwtValid: authState.status === 'valid',
    refreshFailed,
    routes: authState.status === 'valid' ? authState.routes : []
  });

  return applyRouteGuardDecision(req, decision, options, csp, refreshed);
}

export const config = {
  matcher: [
    '/((?!_next|api/|monitoring|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)'
  ]
};
