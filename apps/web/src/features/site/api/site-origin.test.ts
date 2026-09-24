import { describe, expect, it } from 'vitest';
import { originFromHeaders } from './site-origin';

describe('originFromHeaders', () => {
  it('prefers the host and protocol forwarded by the ingress', () => {
    expect(
      originFromHeaders(
        new Headers({
          host: 'web:3000',
          'x-forwarded-host': 'admin.example.com, proxy.internal',
          'x-forwarded-proto': 'https'
        })
      )
    ).toBe('https://admin.example.com');
  });

  it('falls back to the Host header and keeps plain http only when forwarded as such', () => {
    expect(
      originFromHeaders(new Headers({ host: '127.0.0.1:3000', 'x-forwarded-proto': 'http' }))
    ).toBe('http://127.0.0.1:3000');
    expect(originFromHeaders(new Headers({ host: 'admin.example.com' }))).toBe(
      'https://admin.example.com'
    );
  });

  it('rejects hosts that are not a plain hostname', () => {
    expect(originFromHeaders(new Headers())).toBeNull();
    expect(originFromHeaders(new Headers({ host: 'evil.example/path' }))).toBeNull();
    expect(originFromHeaders(new Headers({ host: 'user@evil.example' }))).toBeNull();
  });
});
