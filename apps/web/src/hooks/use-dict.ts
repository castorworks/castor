'use client';

import { useContext } from 'react';
import { DictContext } from '@/components/layout/dict-provider';
import type { DictLookup } from '@/lib/dict';

/**
 * Dictionary lookup for the current locale: `dict.label('asset_status', value)`,
 * `dict.options('asset_status')`, … Column factories are hooks (`useXxxColumns`),
 * so call this there and close over the result in `cell` / `meta.options`.
 *
 * Throws outside `<DictProvider>` (mounted by the dashboard layout). Falling back
 * to an empty lookup would render raw enum values with no error, which is the
 * failure this module exists to prevent.
 */
export function useDict(): DictLookup {
  const lookup = useContext(DictContext);
  if (!lookup) {
    throw new Error('useDict() must be used under <DictProvider> (see app/dashboard/layout.tsx)');
  }
  return lookup;
}
