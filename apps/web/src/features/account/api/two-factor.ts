// ============================================================
// Two-factor authentication (TOTP) — the signed-in user's own settings
// ============================================================

import { mutationOptions, queryOptions } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';
import { getQueryClient } from '@/lib/query-client';
import { encryptPassword } from '@/features/users/lib/rsa';

export interface TwoFactorStatus {
  enabled: boolean;
  enabledAt: string | null;
  recoveryCodesRemaining: number;
}

export interface TwoFactorSetup {
  /** Base32 secret for manual entry. */
  secret: string;
  otpauthUrl: string;
  /** PNG data URL of the otpauth QR code. */
  qrCode: string;
}

export interface RecoveryCodes {
  recoveryCodes: string[];
}

export const twoFactorKeys = {
  status: ['account', 'two-factor'] as const
};

export const twoFactorStatusQueryOptions = () =>
  queryOptions({
    queryKey: twoFactorKeys.status,
    queryFn: () => apiClient<TwoFactorStatus>('/v1/account/totp')
  });

async function refreshStatus() {
  await getQueryClient().invalidateQueries({ queryKey: twoFactorKeys.status });
}

export function beginTwoFactorSetup() {
  return apiClient<TwoFactorSetup>('/v1/account/totp/setup', { method: 'POST' });
}

export function enableTwoFactor(code: string) {
  return apiClient<RecoveryCodes>('/v1/account/totp/enable', {
    method: 'POST',
    body: JSON.stringify({ code: code.trim() })
  });
}

/** `password` is the current password in plain text; it is RSA-encrypted here (empty stays empty). */
export async function disableTwoFactor(password: string, code: string) {
  await apiClient<void>('/v1/account/totp/disable', {
    method: 'POST',
    body: JSON.stringify({
      password: password ? await encryptPassword(password) : '',
      code: code.trim()
    })
  });
}

export function regenerateRecoveryCodes(code: string) {
  return apiClient<RecoveryCodes>('/v1/account/totp/recovery-codes', {
    method: 'POST',
    body: JSON.stringify({ code: code.trim() })
  });
}

export const beginTwoFactorSetupMutation = mutationOptions({ mutationFn: beginTwoFactorSetup });

export const enableTwoFactorMutation = mutationOptions({
  mutationFn: enableTwoFactor,
  onSuccess: refreshStatus
});

export const disableTwoFactorMutation = mutationOptions({
  mutationFn: ({ password, code }: { password: string; code: string }) =>
    disableTwoFactor(password, code),
  onSuccess: refreshStatus
});

export const regenerateRecoveryCodesMutation = mutationOptions({
  mutationFn: regenerateRecoveryCodes,
  onSuccess: refreshStatus
});

/** Text file offered for saving recovery codes. */
export function recoveryCodesFile(codes: string[], site: string): string {
  return [site, '', ...codes, ''].join('\n');
}
