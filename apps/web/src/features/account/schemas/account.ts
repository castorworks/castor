import { z } from 'zod';
import { passwordSchema, type PasswordMessages, type PasswordPolicy } from '@/lib/password-policy';
import type { ContactType } from '../api/types';

export interface UpdateNameMessages {
  nameRequired: string;
  nameTooLong: string;
}

/** Factory for the display-name form schema; messages are translated by the caller */
export function getUpdateNameSchema(messages: UpdateNameMessages) {
  return z.object({
    name: z.string().min(1, messages.nameRequired).max(50, messages.nameTooLong)
  });
}

export interface UpdatePasswordMessages {
  currentRequired: string;
  mismatch: string;
  password: PasswordMessages;
}

/** Factory for the change-password form schema; the new password follows the configured policy */
export function getUpdatePasswordSchema(policy: PasswordPolicy, messages: UpdatePasswordMessages) {
  return z
    .object({
      currentPassword: z.string().min(1, messages.currentRequired),
      newPassword: passwordSchema(policy, messages.password),
      confirmPassword: z.string()
    })
    .refine((data) => data.newPassword === data.confirmPassword, {
      message: messages.mismatch,
      path: ['confirmPassword']
    });
}

export type UpdateNameFormValues = z.infer<ReturnType<typeof getUpdateNameSchema>>;
export type UpdatePasswordFormValues = z.infer<ReturnType<typeof getUpdatePasswordSchema>>;

export interface BindContactMessages {
  contactRequired: string;
  invalidEmail: string;
  invalidMobile: string;
  codeRequired: string;
}

/** Accepts international numbers: optional leading +, 5–20 digits with spaces or dashes */
export const MOBILE_PATTERN = /^\+?\d[\d\s-]{4,19}$/;

/** Factory for the bind-contact form schema; messages are translated by the caller */
export function getBindContactSchema(contactType: ContactType, messages: BindContactMessages) {
  const contact =
    contactType === 'EMAIL'
      ? z.string().min(1, messages.contactRequired).pipe(z.email(messages.invalidEmail))
      : z.string().min(1, messages.contactRequired).regex(MOBILE_PATTERN, messages.invalidMobile);

  return z.object({
    contact,
    code: z.string().min(1, messages.codeRequired)
  });
}

export type BindContactFormValues = z.infer<ReturnType<typeof getBindContactSchema>>;
