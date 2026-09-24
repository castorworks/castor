import * as Sentry from '@sentry/nextjs';
import { getCastorApiUrl } from '@/lib/server-config';

const sentryOptions: Sentry.NodeOptions | Sentry.EdgeOptions = {
  // Sentry DSN
  dsn: process.env.NEXT_PUBLIC_SENTRY_DSN,

  // Enable Spotlight in development
  spotlight: process.env.NODE_ENV === 'development',

  // Keep personally identifiable request data out of telemetry by default.
  sendDefaultPii: false,

  // Keep tracing costs bounded; override with a value between 0 and 1.
  tracesSampleRate: getTracesSampleRate(),

  // Setting this option to true will print useful information to the console while you're setting up Sentry.
  debug: false
};

function getTracesSampleRate(): number {
  const value = Number(process.env.NEXT_PUBLIC_SENTRY_TRACES_SAMPLE_RATE ?? '0.1');
  return Number.isFinite(value) && value >= 0 && value <= 1 ? value : 0.1;
}

export async function register() {
  if (process.env.NEXT_RUNTIME === 'nodejs') {
    // Fail fast on server startup when required production config is missing.
    getCastorApiUrl();
  }

  if (process.env.NEXT_PUBLIC_SENTRY_DSN && process.env.NEXT_PUBLIC_SENTRY_DISABLED !== 'true') {
    Sentry.init(sentryOptions);
  }
}

export const onRequestError = Sentry.captureRequestError;
