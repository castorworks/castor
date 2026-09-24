// This file configures the initialization of Sentry on the client.
// The added config here will be used whenever a users loads a page in their browser.
// https://docs.sentry.io/platforms/javascript/guides/nextjs/
import * as Sentry from '@sentry/nextjs';

if (process.env.NEXT_PUBLIC_SENTRY_DSN && process.env.NEXT_PUBLIC_SENTRY_DISABLED !== 'true') {
  Sentry.init({
    dsn: process.env.NEXT_PUBLIC_SENTRY_DSN,

    // Keep personally identifiable request data out of telemetry by default.
    sendDefaultPii: false,

    // Keep tracing costs bounded; override with a value between 0 and 1.
    tracesSampleRate: getTracesSampleRate(),

    // Setting this option to true will print useful information to the console while you're setting up Sentry.
    debug: false
  });
}

function getTracesSampleRate(): number {
  const value = Number(process.env.NEXT_PUBLIC_SENTRY_TRACES_SAMPLE_RATE ?? '0.1');
  return Number.isFinite(value) && value >= 0 && value <= 1 ? value : 0.1;
}

// Required by Next.js to instrument router transitions for Sentry tracing.
// eslint-disable-next-line @typescript-eslint/no-explicit-any -- Sentry SDK v10 typing mismatch
export const onRouterTransitionStart = (Sentry as any).captureRouterTransitionStart;
