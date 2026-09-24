import { z } from 'zod';
import { locales } from '@/i18n/config';
import type { I18nText } from '@/lib/i18n-text';

export const dictionaryValidationMessages = {
  codeRequired: 'dictionary.validation.codeRequired',
  codeFormat: 'dictionary.validation.codeFormat',
  nameRequired: 'dictionary.validation.nameRequired',
  labelRequired: 'dictionary.validation.labelRequired',
  valueRequired: 'dictionary.validation.valueRequired'
} as const;

type DictionaryValidationMessages = Record<keyof typeof dictionaryValidationMessages, string>;

const defaultMessages: DictionaryValidationMessages = dictionaryValidationMessages;

/**
 * 四语言文案在表单里摊平成 en/zh/ja/ko 四个字段，提交时再组回 I18nText。
 * 与菜单表单保持一致：四种语言都必填，少填一种就会让那种语言的用户
 * 在徽章和下拉框里看到别的语言。
 */
const languageFields = (message: string) =>
  Object.fromEntries(
    locales.map((locale) => [locale, z.string().trim().min(1, message)])
  ) as Record<(typeof locales)[number], z.ZodString>;

/** 把表单里摊平的语言字段收拢成 I18nText。 */
export function toI18nText(values: Record<string, unknown>): I18nText {
  return Object.fromEntries(
    locales.map((locale) => [locale, String(values[locale] ?? '')])
  ) as I18nText;
}

/** 把 I18nText 摊平成表单默认值。 */
export function fromI18nText(text: I18nText | undefined): Record<string, string> {
  return Object.fromEntries(locales.map((locale) => [locale, text?.[locale] ?? '']));
}

export function getCreateDictTypeSchema(messages: DictionaryValidationMessages = defaultMessages) {
  return z.object({
    code: z
      .string()
      .min(1, messages.codeRequired)
      .regex(/^[a-z0-9_]+$/, messages.codeFormat),
    ...languageFields(messages.nameRequired),
    description: z.string().optional(),
    isPublic: z.boolean().optional(),
    sortOrder: z.number().optional()
  });
}

export const createDictTypeSchema = getCreateDictTypeSchema();

export function getCreateDictItemSchema(messages: DictionaryValidationMessages = defaultMessages) {
  return z.object({
    ...languageFields(messages.labelRequired),
    value: z.string().min(1, messages.valueRequired),
    description: z.string().optional(),
    color: z.string().optional(),
    icon: z.string().optional(),
    isDefault: z.boolean().optional(),
    sortOrder: z.number().optional()
  });
}

export const createDictItemSchema = getCreateDictItemSchema();

export type CreateDictTypeFormValues = z.infer<typeof createDictTypeSchema>;
export type CreateDictItemFormValues = z.infer<typeof createDictItemSchema>;
