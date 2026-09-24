// ============================================================
// HTTP security headers
// ============================================================
// The CSP is dynamic (per-request nonce) and is owned by the proxy middleware
// (`src/proxy.ts`); it must NOT also be emitted from next.config to avoid a
// conflicting duplicate. The remaining headers are static and are emitted from
// next.config `headers()` for every route (including static assets the
// middleware matcher skips), and re-applied by the middleware on the responses
// it terminates itself (redirects / 503), which next.config `headers()` does
// not cover. Browser-facing origins come from build-time env (NEXT_PUBLIC_*).
// ============================================================

export interface SecurityHeaderOptions {
  isProduction: boolean;
  /** Browser-side API origin when the API is not proxied through /api. */
  publicApiUrl?: string;
  /** Sentry DSN; its ingest origin is allowed as a fallback to the /monitoring tunnel. */
  sentryDsn?: string;
  /**
   * Per-request nonce (base64), supplied by the proxy middleware. When present
   * in production the CSP drops `'unsafe-inline'` for scripts in favour of
   * `'nonce-<value>' 'strict-dynamic'`. The build-time next.config never sets a
   * nonce, so it never emits a script CSP of its own.
   */
  nonce?: string;
}

function toOrigin(value: string | undefined): string | null {
  if (!value) return null;
  try {
    const { protocol, origin } = new URL(value);
    return protocol === 'http:' || protocol === 'https:' ? origin : null;
  } catch {
    return null;
  }
}

function extra(...values: (string | null | false)[]): string[] {
  return values.filter((value): value is string => Boolean(value));
}

export function buildContentSecurityPolicy({
  isProduction,
  publicApiUrl,
  sentryDsn,
  nonce
}: SecurityHeaderOptions): string {
  const apiOrigin = toOrigin(publicApiUrl);
  const sentryOrigin = toOrigin(sentryDsn);

  // Next.js App Router emits inline bootstrap/flight scripts. In production the
  // proxy middleware supplies a per-request nonce: Next reads it from the
  // request CSP header and stamps it on every script it emits, and
  // 'strict-dynamic' lets those trusted scripts load the chunk graph, so
  // 'unsafe-inline' can be dropped entirely. Without a nonce (dev, or a prod
  // fallback) inline scripts still need 'unsafe-inline'; dev additionally needs
  // eval + HMR sockets. Styles keep 'unsafe-inline' (Tailwind / inline styles).
  const scriptSrc =
    isProduction && nonce
      ? ["'self'", `'nonce-${nonce}'`, "'strict-dynamic'"]
      : ["'self'", "'unsafe-inline'", ...extra(!isProduction && "'unsafe-eval'")];

  const directives: Record<string, string[]> = {
    'default-src': ["'self'"],
    'script-src': scriptSrc,
    'style-src': ["'self'", "'unsafe-inline'"],
    'img-src': ["'self'", 'data:', 'blob:', ...extra(apiOrigin)],
    'font-src': ["'self'", 'data:'],
    'connect-src': [
      "'self'",
      ...extra(apiOrigin, sentryOrigin, !isProduction && 'ws:', !isProduction && 'wss:')
    ],
    'media-src': ["'self'", 'blob:', ...extra(apiOrigin)],
    'worker-src': ["'self'", 'blob:'],
    'frame-src': ["'none'"],
    'frame-ancestors': ["'none'"],
    'base-uri': ["'self'"],
    'form-action': ["'self'"],
    'object-src': ["'none'"]
  };

  return Object.entries(directives)
    .map(([name, values]) => `${name} ${[...new Set(values)].join(' ')}`)
    .join('; ');
}

/**
 * Static (non-CSP) security headers. Emitted by next.config for all routes and
 * re-applied by the middleware on redirect / 503 responses.
 */
export function buildStaticSecurityHeaders(options: SecurityHeaderOptions) {
  const headers = [
    { key: 'X-Frame-Options', value: 'DENY' },
    { key: 'X-Content-Type-Options', value: 'nosniff' },
    { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
    { key: 'Cross-Origin-Opener-Policy', value: 'same-origin' },
    { key: 'Cross-Origin-Resource-Policy', value: 'same-origin' },
    { key: 'X-Permitted-Cross-Domain-Policies', value: 'none' },
    {
      key: 'Permissions-Policy',
      value: 'camera=(), microphone=(), geolocation=(), payment=(), usb=(), browsing-topics=()'
    }
  ];

  if (options.isProduction) {
    headers.push({
      key: 'Strict-Transport-Security',
      value: 'max-age=63072000; includeSubDomains'
    });
  }

  return headers;
}

/** Full header set (dynamic CSP + static headers). Used by the proxy middleware. */
export function buildSecurityHeaders(options: SecurityHeaderOptions) {
  return [
    { key: 'Content-Security-Policy', value: buildContentSecurityPolicy(options) },
    ...buildStaticSecurityHeaders(options)
  ];
}
