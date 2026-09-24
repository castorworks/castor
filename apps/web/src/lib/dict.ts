/**
 * Dictionaries turn enum values (`ACTIVE`, `LOGIN`, …) into what the UI shows:
 * a four-language label, a tag color and an icon, all editable by operators.
 * The values themselves stay owned by backend constants; a dictionary never
 * decides what is valid, only how it looks.
 *
 * Read them through `useDict()` (`@/hooks/use-dict`) or `<DictBadge>`. Both go
 * through the QueryClient from React context, so they see the data the
 * dashboard layout prefetched on the server and re-render when it changes.
 * Do not read the query cache directly: on the server `getQueryClient()` returns
 * a fresh, empty client, which renders raw values and breaks hydration.
 */

import type { DictItem, DictMap } from '@/features/dictionaries/api/types';
import { localizedText } from '@/lib/i18n-text';

/**
 * Dictionary types that code may reference. Mirrors `DefaultDictTypes` in
 * `apps/api/internal/domain/dictionary/entity.go` (enforced by `dict.test.ts`).
 * Types an operator creates at runtime are data, not part of this contract:
 * to use a new dictionary in code, seed it in the backend and list it here.
 */
export const DICT_TYPES = [
  'gender',
  'status',
  'priority',
  'boolean',
  'asset_status',
  'asset_category',
  'account_source',
  'notification_type',
  'notification_level',
  'login_method',
  'audit_log_type',
  'asset_scope',
  'role_data_scope',
  'job_run_status',
  'job_trigger',
  'notification_email_status'
] as const;

export type DictTypeCode = (typeof DICT_TYPES)[number];

export interface DictOption {
  value: string;
  label: string;
}

export interface DictLookup {
  /** Enabled items of a type, in display order. */
  items(type: DictTypeCode): DictItem[];
  /** Localized label; falls back to the raw value (disabled or historical values). */
  label(type: DictTypeCode, value: string): string;
  /** Tag color name for `tagColorBg()`, if the item has one. */
  color(type: DictTypeCode, value: string): string | undefined;
  /** `Icons` key, if the item has one. */
  icon(type: DictTypeCode, value: string): string | undefined;
  /** Localized options for selects and table filters. */
  options(type: DictTypeCode): DictOption[];
}

/**
 * Options of a dictionary, or `fallback` when it is empty — the dictionary failed
 * to load or an operator disabled the whole type. For form selects, which must
 * stay usable either way.
 */
export function dictOptionsOr(
  dict: DictLookup,
  type: DictTypeCode,
  fallback: DictOption[]
): DictOption[] {
  const options = dict.options(type);
  return options.length > 0 ? options : fallback;
}

/** Build a lookup bound to one locale. Pure, so it can be unit-tested without React. */
export function createDictLookup(dicts: DictMap | undefined, locale: string): DictLookup {
  const items = (type: DictTypeCode): DictItem[] => dicts?.[type] ?? [];
  const find = (type: DictTypeCode, value: string) =>
    items(type).find((item) => item.value === value);

  return {
    items,
    label: (type, value) => {
      const item = find(type, value);
      return (item && localizedText(item.label, locale)) || value;
    },
    color: (type, value) => find(type, value)?.color || undefined,
    icon: (type, value) => find(type, value)?.icon || undefined,
    options: (type) =>
      items(type).map((item) => ({
        value: item.value,
        label: localizedText(item.label, locale) || item.value
      }))
  };
}
