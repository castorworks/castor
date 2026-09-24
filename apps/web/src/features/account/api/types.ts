import type { AccessSnapshot } from '@/features/users/api/types';

// ============================================================
// Account types — aligned with castor backend DTOs
// ============================================================

/** Contact channel usable for account recovery */
export type ContactType = 'EMAIL' | 'MOBILE';

export interface AccountInfo {
  id: number;
  username: string;
  name: string;
  avatar: string;
  accountSource: string;
  email: string;
  mobile: string;
  emailVerified: boolean;
  /** The user turned notification emails off (in-app notifications are unaffected). */
  muteNotificationEmails: boolean;
  mobileVerified: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface UpdateNamePayload {
  name: string;
}

export interface UpdatePasswordPayload {
  currentPassword: string;
  newPassword: string;
}

/** Request a verification code for the contact the user is binding */
export interface SendContactCodePayload {
  contactType: ContactType;
  contact: string;
}

/** Verify the code and bind the contact to the current account */
export interface BindContactPayload {
  contactType: ContactType;
  contact: string;
  code: string;
}

/** Public verification code request (sign-in or password reset) */
export interface SendAuthCodePayload {
  codeType: ContactType;
  username: string;
  purpose?: 'auth' | 'reset';
  captchaId?: string;
  captchaCode?: string;
}

/** Password reset with a verification code (identifier = email or mobile) */
export interface ResetPasswordPayload {
  username: string;
  confirmCode: string;
  newPassword: string;
}

/** Changing a password that has expired: the user cannot sign in, so it proves the current one */
export interface ExpiredPasswordPayload {
  username: string;
  currentPassword: string;
  newPassword: string;
  captchaId?: string;
  captchaCode?: string;
}

export type UserRolesResponse = AccessSnapshot;
