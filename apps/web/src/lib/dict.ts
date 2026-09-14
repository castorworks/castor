/**
 * Dictionary utilities for use in non-hook contexts (e.g., column definitions).
 *
 * These read from the TanStack Query cache synchronously.
 * The cache is populated by the `allPublicDictsQueryOptions` query
 * which should be triggered early in the app lifecycle.
 */

import { getQueryClient } from '@/lib/query-client';
import { dictKeys } from '@/features/dictionaries/api/queries';
import type { DictItem } from '@/features/dictionaries/api/types';

export type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'outline';

const COLOR_TO_VARIANT: Record<string, BadgeVariant> = {
  green: 'default',
  red: 'destructive',
  orange: 'secondary',
  gray: 'outline',
  blue: 'default',
  purple: 'secondary'
};

/** Map a color string to a Badge variant */
export function colorToVariant(color: string | undefined | null): BadgeVariant {
  if (!color) return 'outline';
  return COLOR_TO_VARIANT[color] ?? 'outline';
}

/** Read all public dicts from the query cache (synchronous) */
function getAllDictsFromCache(): Record<string, DictItem[]> | undefined {
  return getQueryClient().getQueryData<Record<string, DictItem[]>>(dictKeys.public);
}

/** Get dict items for a type code from cache */
export function getDictItems(typeCode: string): DictItem[] {
  return getAllDictsFromCache()?.[typeCode] ?? [];
}

/** Get a dict item's label by type code and value */
export function getDictLabel(typeCode: string, value: string): string {
  const items = getDictItems(typeCode);
  return items.find((item) => item.value === value)?.label ?? value;
}

/** Get a dict item's badge variant by type code and value */
export function getDictVariant(typeCode: string, value: string): BadgeVariant {
  const items = getDictItems(typeCode);
  const item = items.find((i) => i.value === value);
  return colorToVariant(item?.color);
}

/** Get a dict item's icon key by type code and value */
export function getDictIcon(typeCode: string, value: string): string | undefined {
  const items = getDictItems(typeCode);
  return items.find((i) => i.value === value)?.icon || undefined;
}

/** Get select options for a type code (for table filters / form selects) */
export function getDictOptions(typeCode: string): { value: string; label: string }[] {
  return getDictItems(typeCode).map((item) => ({ value: item.value, label: item.label }));
}
