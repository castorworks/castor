import { describe, expect, it } from 'vitest';
import {
  DEFAULT_PASSWORD_POLICY,
  MAX_PASSWORD_BYTES,
  checkPassword,
  passwordSchema,
  resolvePasswordPolicy
} from './password-policy';

// Cases mirror TestPasswordPolicyValidate in apps/api so both sides agree.
describe('checkPassword', () => {
  const simple = { minLength: 8, requireComplexity: false };
  const complex = { minLength: 8, requireComplexity: true };

  it.each([
    ['meets length', simple, 'abcdefgh', null],
    ['too short', simple, 'abcdefg', 'tooShort'],
    ['empty with zero minimum', { minLength: 0, requireComplexity: false }, '', 'tooShort'],
    ['counts characters not bytes', simple, '密码密码密码密码', null],
    ['seven multibyte characters are too short', simple, '密码密码密码密', 'tooShort'],
    ['bcrypt limit', simple, 'a'.repeat(MAX_PASSWORD_BYTES), null],
    ['over bcrypt limit', simple, 'a'.repeat(MAX_PASSWORD_BYTES + 1), 'tooLong'],
    ['two classes are too weak', complex, 'password123', 'tooWeak'],
    ['three classes', complex, 'Password123', null],
    ['symbols count as a class', complex, 'password-123', null],
    ['spaces are not a class', complex, 'pass word 123', 'tooWeak'],
    ['length checked before complexity', complex, 'Ab1', 'tooShort']
  ] as const)('%s', (_name, policy, password, expected) => {
    expect(checkPassword(password, policy)).toBe(expected);
  });
});

describe('resolvePasswordPolicy', () => {
  it('falls back to the backend defaults', () => {
    expect(resolvePasswordPolicy(undefined)).toEqual(DEFAULT_PASSWORD_POLICY);
    expect(
      resolvePasswordPolicy({
        'security.password.minLength': 'x',
        'security.password.requireComplexity': 'maybe'
      })
    ).toEqual(DEFAULT_PASSWORD_POLICY);
  });

  it('reads the public settings', () => {
    expect(
      resolvePasswordPolicy({
        'security.password.minLength': 12,
        'security.password.requireComplexity': false
      })
    ).toEqual({ minLength: 12, requireComplexity: false });
  });
});

describe('passwordSchema', () => {
  it('reports the translated message of the broken rule', () => {
    const schema = passwordSchema(DEFAULT_PASSWORD_POLICY, {
      tooShort: 'short',
      tooLong: 'long',
      tooWeak: 'weak'
    });
    expect(schema.safeParse('Password123').success).toBe(true);
    const result = schema.safeParse('password123');
    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.message).toBe('weak');
  });
});
