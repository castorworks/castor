import { useTranslations } from 'next-intl';

/**
 * Localized name and description of a job registered in backend code
 * (`jobs.catalog.{key}`). A job added in code before its translations shows its key.
 */
export function useJobText() {
  const t = useTranslations('jobs.catalog');
  return {
    name: (key: string) => (t.has(`${key}.name`) ? t(`${key}.name`) : key),
    description: (key: string) => (t.has(`${key}.description`) ? t(`${key}.description`) : '')
  };
}
