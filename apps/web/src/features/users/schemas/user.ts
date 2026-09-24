import * as z from 'zod';
import { passwordSchema, type PasswordMessages, type PasswordPolicy } from '@/lib/password-policy';

/** Accepts international numbers: optional leading +, 5–20 digits with spaces or dashes */
export const MOBILE_PATTERN = /^\+?\d[\d\s-]{4,19}$/;

export interface UserFormMessages {
  usernameTooShort: string;
  invalidEmail: string;
  invalidMobile: string;
  password: PasswordMessages;
}

/** Select value meaning "not in any department" (Radix selects cannot hold ''). */
export const NO_DEPARTMENT = 'none';

/** Optional contact fields shared by the create and update schemas */
function contactFields(messages: UserFormMessages) {
  return {
    email: z.union([z.literal(''), z.email(messages.invalidEmail)]).optional(),
    mobile: z
      .union([z.literal(''), z.string().regex(MOBILE_PATTERN, messages.invalidMobile)])
      .optional()
  };
}

/** Factory for the user creation schema; the password follows the configured policy */
export function createUserSchema(messages: UserFormMessages, policy: PasswordPolicy) {
  return z.object({
    username: z.string().min(2, messages.usernameTooShort),
    name: z.string().optional(),
    password: passwordSchema(policy, messages.password),
    department: z.string().optional(),
    ...contactFields(messages)
  });
}

/** Factory for the user update schema; an empty password keeps the current one */
export function updateUserSchema(messages: UserFormMessages, policy: PasswordPolicy) {
  return z.object({
    name: z.string().optional(),
    avatar: z.string().optional(),
    ...contactFields(messages),
    password: z.union([z.literal(''), passwordSchema(policy, messages.password)]).optional(),
    department: z.string().optional(),
    enable: z.boolean().optional(),
    locked: z.boolean().optional(),
    accountExpireDate: z.string().nullable().optional(),
    credentialExpireDate: z.string().nullable().optional()
  });
}

export type CreateUserFormValues = z.infer<ReturnType<typeof createUserSchema>>;
export type UpdateUserFormValues = z.infer<ReturnType<typeof updateUserSchema>>;
