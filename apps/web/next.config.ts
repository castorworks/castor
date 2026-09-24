import type { NextConfig } from 'next';
import { PHASE_PRODUCTION_BUILD } from 'next/constants';
import { withSentryConfig } from '@sentry/nextjs';
import createNextIntlPlugin from 'next-intl/plugin';
import { getCastorApiUrl } from './src/lib/server-config';
import { buildStaticSecurityHeaders } from './src/lib/security-headers';

const withNextIntl = createNextIntlPlugin('./src/i18n/request.ts');

const isProduction = process.env.NODE_ENV === 'production';
const sentryEnabled =
  Boolean(process.env.NEXT_PUBLIC_SENTRY_DSN) && process.env.NEXT_PUBLIC_SENTRY_DISABLED !== 'true';

function castorApiUrlForPhase(phase: string): string {
  const env = { ...process.env, NEXT_PHASE: phase };
  if (phase === PHASE_PRODUCTION_BUILD && !process.env.CASTOR_API_URL?.trim()) {
    console.warn(
      `[castor] CASTOR_API_URL is not set; the /api/v1 rewrite will target ${getCastorApiUrl(env)}.`
    );
  }
  return getCastorApiUrl(env);
}

const createBaseConfig = (phase: string): NextConfig => ({
  output: process.env.BUILD_STANDALONE === 'true' ? 'standalone' : undefined,
  poweredByHeader: false,

  turbopack: {
    root: process.cwd()
  },

  // Proxy /api/v1/* requests to the castor backend. Rewrites are resolved when
  // the config loads (baked in at `next build`), so CASTOR_API_URL must be set
  // for production builds; SSR and the route guard read it again at runtime.
  async rewrites() {
    return [
      {
        source: '/api/v1/:path*',
        destination: `${castorApiUrlForPhase(phase)}/api/v1/:path*`
      }
    ];
  },

  // Static security headers for every route (including the static assets the
  // proxy middleware matcher skips). The dynamic Content-Security-Policy is
  // owned by the middleware (`src/proxy.ts`) so it can carry a per-request
  // nonce; it is intentionally NOT set here to avoid a conflicting duplicate.
  async headers() {
    return [
      {
        source: '/:path*',
        headers: buildStaticSecurityHeaders({ isProduction })
      }
    ];
  },

  transpilePackages: ['geist'],
  compiler: {
    removeConsole: isProduction ? { exclude: ['error', 'warn'] } : false
  }
});

export default function nextConfig(phase: string): NextConfig {
  // Apply next-intl plugin first, then Sentry
  const configWithPlugins = withNextIntl(createBaseConfig(phase));
  if (!sentryEnabled) return configWithPlugins;

  return withSentryConfig(configWithPlugins, {
    org: process.env.NEXT_PUBLIC_SENTRY_ORG,
    project: process.env.NEXT_PUBLIC_SENTRY_PROJECT,
    // Only print logs for uploading source maps in CI
    silent: !process.env.CI,

    // Upload a larger set of source maps for prettier stack traces (increases build time)
    widenClientFileUpload: true,

    // Route browser requests to Sentry through a Next.js rewrite to circumvent ad-blockers.
    tunnelRoute: '/monitoring',

    // Disable Sentry telemetry
    telemetry: false,

    webpack: {
      reactComponentAnnotation: {
        enabled: true
      },
      treeshake: {
        removeDebugLogging: true
      }
    },

    // Disable source map upload when org/project are not configured
    sourcemaps: {
      disable: !process.env.NEXT_PUBLIC_SENTRY_ORG || !process.env.NEXT_PUBLIC_SENTRY_PROJECT
    }
  });
}
