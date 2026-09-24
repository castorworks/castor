import { getRequestConfig } from 'next-intl/server';
import { cookies, headers } from 'next/headers';
import { defaultLocale, type Locale, locales, LOCALE_COOKIE } from './config';

/** Parse Accept-Language header to find the best matching locale */
function detectLocaleFromAcceptLanguage(acceptLanguage: string | null): Locale | null {
  if (!acceptLanguage) return null;

  // Parse quality-weighted locales: "zh-CN,zh;q=0.9,en;q=0.8"
  const parsed = acceptLanguage
    .split(',')
    .map((entry) => {
      const [tag, qStr] = entry.trim().split(';q=');
      const lang = tag.split('-')[0]; // Strip region suffix (e.g., "zh-CN" -> "zh")
      const q = qStr ? parseFloat(qStr) : 1.0;
      return { lang, q };
    })
    .toSorted((a, b) => b.q - a.q);

  for (const { lang } of parsed) {
    if (locales.includes(lang as Locale)) {
      return lang as Locale;
    }
  }
  return null;
}

export default getRequestConfig(async () => {
  const cookieStore = await cookies();
  const rawLocale = cookieStore.get(LOCALE_COOKIE)?.value;

  // 1. Cookie 中已保存的语言选择
  if (rawLocale && locales.includes(rawLocale as Locale)) {
    return {
      locale: rawLocale as Locale,
      messages: (await import(`../../messages/${rawLocale}.json`)).default
    };
  }

  // 2. 首次访问：从 Accept-Language header 检测浏览器语言
  const acceptLanguage = (await headers()).get('accept-language');
  const detectedLocale = detectLocaleFromAcceptLanguage(acceptLanguage);
  const locale = detectedLocale ?? defaultLocale;

  return {
    locale,
    messages: (await import(`../../messages/${locale}.json`)).default
  };
});
