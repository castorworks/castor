import { z } from 'zod';
import { locales } from '@/i18n/config';

export interface ProviderFormMessages {
  codeInvalid: string;
  nameRequired: string;
  issuerInvalid: string;
  clientIdRequired: string;
  clientSecretRequired: string;
  scopesInvalid: string;
}

/** Mirrors the backend rules; `requireSecret` is true when creating (no stored secret yet). */
export function providerSchema(messages: ProviderFormMessages, requireSecret: boolean) {
  return z.object({
    code: z
      .string()
      .trim()
      .regex(/^[a-z0-9][a-z0-9-]{0,31}$/, messages.codeInvalid),
    ...(Object.fromEntries(
      locales.map((locale) => [locale, z.string().trim().min(1, messages.nameRequired).max(100)])
    ) as Record<(typeof locales)[number], z.ZodString>),
    issuer: z
      .string()
      .trim()
      .refine((value) => /^https?:\/\/[^\s/]+/.test(value), messages.issuerInvalid),
    clientId: z.string().trim().min(1, messages.clientIdRequired),
    clientSecret: requireSecret ? z.string().min(1, messages.clientSecretRequired) : z.string(),
    scopes: z
      .string()
      .refine((value) => value.split(/\s+/).includes('openid'), messages.scopesInvalid),
    usernameClaim: z.string().trim().max(64),
    autoRegister: z.boolean(),
    isEnabled: z.boolean(),
    sortOrder: z.number().int()
  });
}

export type ProviderFormValues = z.infer<ReturnType<typeof providerSchema>>;
