import { describe, expect, it } from 'vitest';
import { createUserSchema, updateUserSchema } from './user';

const messages = {
  usernameTooShort: 'username too short',
  invalidEmail: 'invalid email',
  invalidMobile: 'invalid mobile',
  password: { tooShort: 'too short', tooLong: 'too long', tooWeak: 'too weak' }
};
const policy = { minLength: 8, requireComplexity: true };

describe('user contact fields', () => {
  it('treats email and mobile as optional on create', () => {
    const result = createUserSchema(messages, policy).safeParse({
      username: 'alice',
      password: 'Secret123'
    });
    expect(result.success).toBe(true);
  });

  it('allows empty contacts to clear the value on update', () => {
    const result = updateUserSchema(messages, policy).safeParse({ email: '', mobile: '' });
    expect(result.success).toBe(true);
  });

  it('rejects malformed contacts', () => {
    const email = createUserSchema(messages, policy).safeParse({
      username: 'alice',
      password: 'Secret123',
      email: 'nope'
    });
    expect(email.success).toBe(false);

    const mobile = updateUserSchema(messages, policy).safeParse({ mobile: 'abc' });
    expect(mobile.success).toBe(false);
  });

  it('accepts valid contacts', () => {
    const result = createUserSchema(messages, policy).safeParse({
      username: 'alice',
      password: 'Secret123',
      email: 'alice@example.com',
      mobile: '+1 415-555-0100'
    });
    expect(result.success).toBe(true);
  });
});

describe('user password', () => {
  it('applies the configured policy with translated messages', () => {
    const result = createUserSchema(messages, policy).safeParse({
      username: 'alice',
      password: 'secret123'
    });
    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.message).toBe('too weak');
  });

  it('keeps the current password when the update leaves it blank', () => {
    expect(updateUserSchema(messages, policy).safeParse({ password: '' }).success).toBe(true);
    expect(updateUserSchema(messages, policy).safeParse({ password: 'short' }).success).toBe(false);
  });
});
