// ============================================================
// Password policy — mirrors the backend PasswordPolicy.Validate
// ============================================================
// The backend is the authority; this copy only lets forms explain the policy
// and reject an unacceptable password before the round trip. The policy comes
// from the public settings security.password.minLength / requireComplexity.
// ============================================================

import * as z from 'zod';

export interface PasswordPolicy {
  minLength: number;
  requireComplexity: boolean;
}

export const PASSWORD_MIN_LENGTH_SETTING = 'security.password.minLength';
export const PASSWORD_COMPLEXITY_SETTING = 'security.password.requireComplexity';

/** Same fallbacks the backend uses when a setting is missing or invalid. */
export const DEFAULT_PASSWORD_POLICY: PasswordPolicy = { minLength: 8, requireComplexity: true };

/** bcrypt only verifies the first 72 bytes, so the backend rejects longer passwords. */
export const MAX_PASSWORD_BYTES = 72;

const COMPLEXITY_CLASSES = 3;

export type PasswordIssue = 'tooShort' | 'tooLong' | 'tooWeak';

export type PasswordMessages = Record<PasswordIssue, string>;

export function resolvePasswordPolicy(
  settings: Record<string, unknown> | undefined
): PasswordPolicy {
  const rawLength = settings?.[PASSWORD_MIN_LENGTH_SETTING];
  const length = typeof rawLength === 'string' ? Number.parseInt(rawLength, 10) : rawLength;
  const rawComplexity = settings?.[PASSWORD_COMPLEXITY_SETTING];
  return {
    minLength:
      typeof length === 'number' && Number.isInteger(length) && length > 0
        ? length
        : DEFAULT_PASSWORD_POLICY.minLength,
    requireComplexity:
      rawComplexity === true || rawComplexity === 'true'
        ? true
        : rawComplexity === false || rawComplexity === 'false'
          ? false
          : DEFAULT_PASSWORD_POLICY.requireComplexity
  };
}

function characterClasses(password: string): number {
  let lower = false;
  let upper = false;
  let digit = false;
  let symbol = false;
  for (const char of password) {
    if (/\p{Ll}/u.test(char)) lower = true;
    else if (/\p{Lu}/u.test(char)) upper = true;
    else if (/\p{Nd}/u.test(char)) digit = true;
    else if (!/\s/u.test(char)) symbol = true;
  }
  return [lower, upper, digit, symbol].filter(Boolean).length;
}

/** Returns the first rule the password breaks, in the backend's order, or null. */
export function checkPassword(password: string, policy: PasswordPolicy): PasswordIssue | null {
  if (password === '' || Array.from(password).length < policy.minLength) return 'tooShort';
  if (new TextEncoder().encode(password).length > MAX_PASSWORD_BYTES) return 'tooLong';
  if (policy.requireComplexity && characterClasses(password) < COMPLEXITY_CLASSES) return 'tooWeak';
  return null;
}

/** zod string schema enforcing the policy; messages are translated by the caller. */
export function passwordSchema(policy: PasswordPolicy, messages: PasswordMessages) {
  return z.string().superRefine((value, ctx) => {
    const issue = checkPassword(value, policy);
    if (issue) ctx.addIssue({ code: 'custom', message: messages[issue] });
  });
}
