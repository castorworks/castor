// ============================================================
// HTTP security headers
// ============================================================
// Consumed by next.config.ts `headers()`. Values are resolved when the Next.js
// config is loaded (at `next build` for production), so browser-facing origins
// here must come from build-time env (NEXT_PUBLIC_*).
// ============================================================

export interface SecurityHeaderOptions {
  isProduction: boolean;
  /** Browser-side API origin when the API is not proxied through /api. */
  publicApiUrl?: string;
  /** Sentry DSN; its ingest origin is allowed as a fallback to the /monitoring tunnel. */
  sentryDsn?: string;
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
  sentryDsn
}: SecurityHeaderOptions): string {
  const apiOrigin = toOrigin(publicApiUrl);
  const sentryOrigin = toOrigin(sentryDsn);

  // Next.js App Router emits inline bootstrap scripts; without a per-request
  // nonce 'unsafe-inline' is required. Dev additionally needs eval + HMR sockets.
  const directives: Record<string, string[]> = {
    'default-src': ["'self'"],
    'script-src': ["'self'", "'unsafe-inline'", ...extra(!isProduction && "'unsafe-eval'")],
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

export function buildSecurityHeaders(options: SecurityHeaderOptions) {
  const headers = [
    { key: 'Content-Security-Policy', value: buildContentSecurityPolicy(options) },
    { key: 'X-Frame-Options', value: 'DENY' },
    { key: 'X-Content-Type-Options', value: 'nosniff' },
    { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
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
