'use client';

import { useQuery } from '@tanstack/react-query';
import { allPublicDictsQueryOptions } from '@/features/dictionaries/api/queries';

/**
 * Invisible component that triggers the public dictionaries query on mount.
 * Place in the dashboard layout so dict data is available in the cache
 * before any module tries to read it.
 */
export function DictPrefetch() {
  useQuery(allPublicDictsQueryOptions());
  return null;
}
