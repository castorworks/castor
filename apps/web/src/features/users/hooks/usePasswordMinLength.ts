'use client';

import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';

const DEFAULT_MIN_LENGTH = 8;
const SETTING_KEY = 'security.password.minLength';

type PublicSettings = Record<string, unknown>;

async function fetchPublicSettings(): Promise<PublicSettings> {
  return apiClient<PublicSettings>('/v1/settings/public');
}

/**
 * Fetches the configured password minimum length from public settings.
 * Falls back to 8 if the fetch fails or the value is not a valid positive integer.
 */
export function usePasswordMinLength(): number {
  const { data } = useQuery({
    queryKey: ['settings', 'public', 'passwordMinLength'],
    queryFn: fetchPublicSettings,
    staleTime: 10 * 60 * 1000, // 10 minutes — this setting rarely changes
    gcTime: 30 * 60 * 1000 // keep in cache for 30 minutes
  });

  if (!data) return DEFAULT_MIN_LENGTH;

  const raw = data[SETTING_KEY];

  if (typeof raw === 'number' && Number.isInteger(raw) && raw > 0) {
    return raw;
  }

  if (typeof raw === 'string') {
    const parsed = Number.parseInt(raw, 10);
    if (Number.isFinite(parsed) && parsed > 0) {
      return parsed;
    }
  }

  return DEFAULT_MIN_LENGTH;
}
