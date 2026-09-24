import { describe, expect, it } from 'vitest';
import { getBindContactSchema, getUpdatePasswordSchema } from './account';

const messages = {
  contactRequired: 'contact required',
  invalidEmail: 'invalid email',
  invalidMobile: 'invalid mobile',
  codeRequired: 'code required'
};

function firstError(result: { success: boolean; error?: { issues: { message: string }[] } }) {
  return result.error?.issues[0]?.message;
}

describe('getBindContactSchema', () => {
  it('accepts a valid email with a code', () => {
    const result = getBindContactSchema('EMAIL', messages).safeParse({
      contact: 'user@example.com',
      code: '123456'
    });
    expect(result.success).toBe(true);
  });

  it('rejects a malformed email', () => {
    const result = getBindContactSchema('EMAIL', messages).safeParse({
      contact: 'not-an-email',
      code: '123456'
    });
    expect(result.success).toBe(false);
    expect(firstError(result)).toBe('invalid email');
  });

  it('requires a contact value', () => {
    const result = getBindContactSchema('MOBILE', messages).safeParse({ contact: '', code: '1' });
    expect(firstError(result)).toBe('contact required');
  });

  it('accepts international mobile numbers', () => {
    const schema = getBindContactSchema('MOBILE', messages);
    expect(schema.safeParse({ contact: '13800138000', code: '1' }).success).toBe(true);
    expect(schema.safeParse({ contact: '+81 90-1234-5678', code: '1' }).success).toBe(true);
  });

  it('rejects a malformed mobile number', () => {
    const result = getBindContactSchema('MOBILE', messages).safeParse({
      contact: 'abc',
      code: '123456'
    });
    expect(firstError(result)).toBe('invalid mobile');
  });

  it('requires a verification code', () => {
    const result = getBindContactSchema('EMAIL', messages).safeParse({
      contact: 'user@example.com',
      code: ''
    });
    expect(firstError(result)).toBe('code required');
  });
});

describe('getUpdatePasswordSchema', () => {
  const schema = getUpdatePasswordSchema(
    { minLength: 8, requireComplexity: true },
    {
      currentRequired: 'current required',
      mismatch: 'mismatch',
      password: { tooShort: 'too short', tooLong: 'too long', tooWeak: 'too weak' }
    }
  );

  it('accepts a new password that meets the policy', () => {
    expect(
      schema.safeParse({
        currentPassword: 'old',
        newPassword: 'New-password1',
        confirmPassword: 'New-password1'
      }).success
    ).toBe(true);
  });

  it('applies the configured policy instead of a fixed length', () => {
    const result = schema.safeParse({
      currentPassword: 'old',
      newPassword: 'newpassword',
      confirmPassword: 'newpassword'
    });
    expect(firstError(result)).toBe('too weak');
  });

  it('requires the confirmation to match', () => {
    const result = schema.safeParse({
      currentPassword: 'old',
      newPassword: 'New-password1',
      confirmPassword: 'New-password2'
    });
    expect(firstError(result)).toBe('mismatch');
  });
});
