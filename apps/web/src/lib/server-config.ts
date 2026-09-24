// ============================================================
// Server-side runtime configuration
// ============================================================
// Single source of truth for the backend origin used by SSR, the route guard
// (proxy.ts) and the /api/v1 rewrite in next.config.ts. Never read from
// client code paths: CASTOR_API_URL is not exposed to the browser.
// ============================================================

export const DEV_CASTOR_API_URL = 'http://localhost:1234';

const PRODUCTION_BUILD_PHASE = 'phase-production-build';

export class MissingServerConfigError extends Error {
  constructor(name: string) {
    super(`${name} must be set when running Castor web in production.`);
    this.name = 'MissingServerConfigError';
  }
}

function normalizeOrigin(value: string): string {
  return value.trim().replace(/\/+$/, '');
}

/**
 * Resolve the Castor backend origin.
 *
 * - Set: always used.
 * - Unset in development/test, or during `next build`: falls back to the local
 *   dev backend so builds (e.g. Docker without build args) never fail.
 * - Unset at production runtime: throws, so misconfiguration fails loudly at
 *   request time instead of silently calling localhost.
 */
export function getCastorApiUrl(env: NodeJS.ProcessEnv = process.env): string {
  const value = env.CASTOR_API_URL?.trim();
  if (value) return normalizeOrigin(value);

  if (env.NODE_ENV === 'production' && env.NEXT_PHASE !== PRODUCTION_BUILD_PHASE) {
    throw new MissingServerConfigError('CASTOR_API_URL');
  }

  return DEV_CASTOR_API_URL;
}
