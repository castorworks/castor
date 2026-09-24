import { describe, expect, it } from 'vitest';
import {
  buildContentSecurityPolicy,
  buildSecurityHeaders,
  buildStaticSecurityHeaders
} from './security-headers';

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

  it('includes the CSP alongside the static headers', () => {
    expect(keys(buildSecurityHeaders({ isProduction: true }))).toContain('Content-Security-Policy');
  });
});

describe('buildStaticSecurityHeaders', () => {
  it('sets isolation and legacy hardening headers without a CSP', () => {
    const headers = buildStaticSecurityHeaders({ isProduction: true });
    const map = Object.fromEntries(headers.map((header) => [header.key, header.value]));
    expect(map['Cross-Origin-Opener-Policy']).toBe('same-origin');
    expect(map['Cross-Origin-Resource-Policy']).toBe('same-origin');
    expect(map['X-Permitted-Cross-Domain-Policies']).toBe('none');
    expect(map['X-Frame-Options']).toBe('DENY');
    expect(map['X-Content-Type-Options']).toBe('nosniff');
    // COEP is intentionally omitted (would break third-party subresources).
    expect(keys(headers)).not.toContain('Cross-Origin-Embedder-Policy');
    expect(keys(headers)).not.toContain('Content-Security-Policy');
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

  it('uses a nonce and strict-dynamic for scripts in production, dropping unsafe-inline', () => {
    const csp = buildContentSecurityPolicy({ isProduction: true, nonce: 'abc123==' });
    expect(csp).toContain("script-src 'self' 'nonce-abc123==' 'strict-dynamic'");
    expect(csp).not.toContain("script-src 'self' 'unsafe-inline'");
    // Styles keep unsafe-inline (Tailwind / inline styles).
    expect(csp).toContain("style-src 'self' 'unsafe-inline'");
  });

  it('falls back to unsafe-inline scripts in production when no nonce is supplied', () => {
    const csp = buildContentSecurityPolicy({ isProduction: true });
    expect(csp).toContain("script-src 'self' 'unsafe-inline'");
    expect(csp).not.toContain('strict-dynamic');
  });

  it('keeps unsafe-inline for scripts in development even with a nonce', () => {
    const csp = buildContentSecurityPolicy({ isProduction: false, nonce: 'abc123==' });
    expect(csp).toContain("'unsafe-inline'");
    expect(csp).not.toContain('nonce-');
  });
});
