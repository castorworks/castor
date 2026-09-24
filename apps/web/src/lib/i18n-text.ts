import type { Locale } from '@/i18n/config';

/**
 * 一段四语言文案，对应后端的 shared.I18nText。
 *
 * 用于运营可编辑、因而无法放进 messages 文件的数据：菜单标题、字典标签。
 * 固定界面文案仍然走 next-intl 的翻译 key。
 */
export type I18nText = Record<Locale, string>;

/** 取指定语言的文案，缺失时回退英文，再缺则回退空串。 */
export function localizedText(text: I18nText | undefined, locale: string): string {
  if (!text) return '';
  return text[locale as Locale] || text.en || '';
}
