'use client';

import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { publicSettingsQueryOptions } from '@/features/settings/api/queries';
import {
  MAX_PASSWORD_BYTES,
  resolvePasswordPolicy,
  type PasswordMessages,
  type PasswordPolicy
} from '@/lib/password-policy';

/**
 * The password policy from the public settings, with translated validation
 * messages and a one-line hint for password fields. Until the settings load the
 * backend defaults apply; the backend re-validates every submission anyway.
 */
export function usePasswordPolicy(): {
  policy: PasswordPolicy;
  messages: PasswordMessages;
  hint: string;
} {
  const t = useTranslations('passwordPolicy');
  const { data } = useQuery({ ...publicSettingsQueryOptions(), staleTime: 10 * 60 * 1000 });
  const policy = resolvePasswordPolicy(data);
  const { minLength, requireComplexity } = policy;
  return useMemo(
    () => ({
      policy: { minLength, requireComplexity },
      messages: {
        tooShort: t('tooShort', { min: minLength }),
        tooLong: t('tooLong', { max: MAX_PASSWORD_BYTES }),
        tooWeak: t('tooWeak')
      },
      hint: t('hint', { min: minLength, complex: String(requireComplexity) })
    }),
    [t, minLength, requireComplexity]
  );
}
