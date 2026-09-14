import { describe, expect, it } from 'vitest';
import { buildContentSecurityPolicy, buildSecurityHeaders } from './security-headers';

const keys = (headers: { key: string }[]) => headers.map((header) => header.key);

describe('buildSecurityHeaders', () => {
  it('adds HSTS only in production', () => {
    expect(keys(buildSecurityHeaders({ isProduction: true }))).toContain(
      'Strict-Transport-Security'
    );
    expect(keys(buildSecurityHeaders({ isProduction: false }))).not.toContain(
      'Strict-Transport-Security'
    );
  });
});

describe('buildContentSecurityPolicy', () => {
  it('locks down framing, base URI, forms and plugins', () => {
    const csp = buildContentSecurityPolicy({ isProduction: true });
    expect(csp).toContain("frame-ancestors 'none'");
    expect(csp).toContain("base-uri 'self'");
    expect(csp).toContain("form-action 'self'");
    expect(csp).toContain("object-src 'none'");
    expect(csp).not.toContain('unsafe-eval');
    expect(csp).not.toContain('ws:');
  });

  it('allows configured API and Sentry origins', () => {
    const csp = buildContentSecurityPolicy({
      isProduction: true,
      publicApiUrl: 'https://api.example.com/base',
      sentryDsn: 'https://key@o1.ingest.sentry.io/42'
    });
    expect(csp).toContain("connect-src 'self' https://api.example.com https://o1.ingest.sentry.io");
    expect(csp).toContain("img-src 'self' data: blob: https://api.example.com");
  });

  it('ignores invalid origins and relaxes dev-only directives', () => {
    const csp = buildContentSecurityPolicy({
      isProduction: false,
      publicApiUrl: 'javascript:alert(1)',
      sentryDsn: 'not a url'
    });
    expect(csp).not.toContain('javascript:');
    expect(csp).toContain("'unsafe-eval'");
    expect(csp).toContain("connect-src 'self' ws: wss:");
  });
});
