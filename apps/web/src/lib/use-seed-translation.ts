import { useTranslations } from 'next-intl';

/**
 * Hook to translate seed data names from the backend.
 *
 * Backend seed data stores i18n keys (e.g. "seedRoles.admin") in name/description fields.
 * This hook attempts to resolve the key via next-intl. If the key is not found
 * (i.e. it's a user-created entry with a plain text name), it falls back to the raw value.
 */
export function useSeedTranslation() {
  const t = useTranslations();
  type TranslationKey = Parameters<typeof t>[0];

  /**
   * Translate a seed data field. If the value looks like an i18n key (contains a dot
   * and starts with "seed"), resolve it. Otherwise return as-is.
   */
  function ts(value: string | undefined | null): string {
    if (!value) return '';
    // Only attempt translation for seed keys (prefixed with "seed")
    if (value.startsWith('seed') && value.includes('.')) {
      try {
        return t(value as TranslationKey);
      } catch {
        // Key not found in translations, return raw value
        return value;
      }
    }
    return value;
  }

  return { ts };
}
