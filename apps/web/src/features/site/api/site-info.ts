// ============================================================
// Site Info — brand for the public site
// ============================================================
// The public pages (home, about, legal) take their brand from the operator-
// editable public settings. Server-only: fetched once per request and cached.
// ============================================================

import { cache } from 'react';
import { getPublicSettings } from '@/features/settings/api/service';
import type { PublicSettings } from '@/features/settings/api/types';

export interface SiteInfo {
  name: string;
  description: string;
  registerEnabled: boolean;
}

// Mirrors the backend seed, used when a setting is blank or the API is down.
const FALLBACK_SITE_NAME = 'Castor Admin';

// Every visitor's SSR call reaches the backend from the web server's single IP,
// and `/settings/public` is rate limited per IP. Serve it from the Next data
// cache so the home page costs the backend at most one request per window.
const PUBLIC_SETTINGS_REVALIDATE_SECONDS = 60;

function text(value: unknown): string {
  return typeof value === 'string' ? value.trim() : '';
}

export function resolveSiteInfo(settings: PublicSettings | null): SiteInfo {
  const name = text(settings?.['site.name']) || FALLBACK_SITE_NAME;
  return {
    name,
    description: text(settings?.['site.description']) || name,
    registerEnabled: settings?.['feature.register.enabled'] === true
  };
}

export const getSiteInfo = cache(async (): Promise<SiteInfo> => {
  try {
    return resolveSiteInfo(
      await getPublicSettings({ next: { revalidate: PUBLIC_SETTINGS_REVALIDATE_SECONDS } })
    );
  } catch {
    // The public site must render even while the API is unreachable.
    return resolveSiteInfo(null);
  }
});
