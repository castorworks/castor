'use client';

import { useLocale, useTranslations } from 'next-intl';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@/components/ui/select';
import { Icons } from '@/components/icons';
import { locales, localeNames, LOCALE_COOKIE } from '@/i18n/config';

/**
 * Sets the locale cookie and reloads the page.
 * The cookie is read by next-intl's request config to determine the active locale.
 */
function setLocaleCookie(locale: string) {
  // Set cookie with max-age of 1 year, same-site lax for SSR compatibility
  document.cookie = `${LOCALE_COOKIE}=${locale};path=/;max-age=31536000;SameSite=Lax`;

  // Update the Accept-Language header for future API calls by reloading
  window.location.reload();
}

export function LanguageSwitcher() {
  const locale = useLocale();
  const t = useTranslations('language');

  return (
    <div className='flex items-center gap-2'>
      <Label htmlFor='language-selector' className='sr-only'>
        {t('label')}
      </Label>
      <Select value={locale} onValueChange={setLocaleCookie}>
        <SelectTrigger id='language-selector' className='w-[110px] justify-start'>
          <span className='text-muted-foreground'>
            <Icons.globe />
          </span>
          <SelectValue placeholder={t('label')} />
        </SelectTrigger>
        <SelectContent align='end'>
          {locales.map((loc) => (
            <SelectItem key={loc} value={loc}>
              {localeNames[loc]}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
