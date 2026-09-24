import { describe, expect, it } from 'vitest';
import { DEV_CASTOR_API_URL, getCastorApiUrl, MissingServerConfigError } from './server-config';

const env = (values: Record<string, string | undefined>) => values as NodeJS.ProcessEnv;

describe('getCastorApiUrl', () => {
  it('uses the configured origin without trailing slashes', () => {
    expect(
      getCastorApiUrl(env({ NODE_ENV: 'production', CASTOR_API_URL: ' http://api:1234/ ' }))
    ).toBe('http://api:1234');
  });

  it('falls back to the local backend outside production', () => {
    expect(getCastorApiUrl(env({ NODE_ENV: 'development' }))).toBe(DEV_CASTOR_API_URL);
    expect(getCastorApiUrl(env({ NODE_ENV: 'test', CASTOR_API_URL: '' }))).toBe(DEV_CASTOR_API_URL);
  });

  it('does not fail during next build', () => {
    expect(
      getCastorApiUrl(env({ NODE_ENV: 'production', NEXT_PHASE: 'phase-production-build' }))
    ).toBe(DEV_CASTOR_API_URL);
  });

  it('throws at production runtime when unset', () => {
    expect(() => getCastorApiUrl(env({ NODE_ENV: 'production' }))).toThrow(
      MissingServerConfigError
    );
  });
});
