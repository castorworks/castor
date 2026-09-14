'use client';

import { useQuery } from '@tanstack/react-query';
import { allPublicDictsQueryOptions } from '@/features/dictionaries/api/queries';
import type { DictItem } from '@/features/dictionaries/api/types';
import { colorToVariant, type BadgeVariant } from '@/lib/dict';

export { colorToVariant, type BadgeVariant } from '@/lib/dict';

/**
 * Get all dictionary items for a given type code.
 * Data is cached globally with a 5-minute staleTime — no redundant requests.
 */
export function useDictItems(typeCode: string): DictItem[] {
  const { data } = useQuery(allPublicDictsQueryOptions());
  return data?.[typeCode] ?? [];
}

/**
 * Get select options (value + label) for a given type code.
 * Useful for table filter dropdowns and form selects.
 */
export function useDictOptions(typeCode: string): { value: string; label: string }[] {
  const items = useDictItems(typeCode);
  return items.map((item) => ({ value: item.value, label: item.label }));
}

/**
 * Get a single dict item's label by type code and value.
 * Returns the raw value as fallback if not found.
 */
export function useDictLabel(typeCode: string, value: string): string {
  const items = useDictItems(typeCode);
  return items.find((item) => item.value === value)?.label ?? value;
}

/**
 * Get the badge variant for a specific dict value.
 */
export function useDictVariant(typeCode: string, value: string): BadgeVariant {
  const items = useDictItems(typeCode);
  const item = items.find((i) => i.value === value);
  return colorToVariant(item?.color);
}
