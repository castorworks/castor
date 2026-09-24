'use client';

import { createContext, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useLocale } from 'next-intl';
import { enabledDictsQueryOptions } from '@/features/dictionaries/api/queries';
import { createDictLookup, type DictLookup } from '@/lib/dict';

export const DictContext = createContext<DictLookup | null>(null);

/**
 * Holds the one subscription to the enabled-dictionaries query and shares the
 * lookup through context, so a table with hundreds of badges costs one query
 * observer instead of one per cell.
 *
 * The dashboard layout prefetches the query on the server and wraps this in a
 * HydrationBoundary: labels are in the first HTML, and an operator's edit in
 * dictionary management re-renders every consumer once the query is invalidated.
 */
export function DictProvider({ children }: { children: React.ReactNode }) {
  const locale = useLocale();
  const { data } = useQuery(enabledDictsQueryOptions());
  const lookup = useMemo(() => createDictLookup(data, locale), [data, locale]);
  return <DictContext.Provider value={lookup}>{children}</DictContext.Provider>;
}
