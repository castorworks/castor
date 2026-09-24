// ============================================================
// Account Service — Data Access Layer
// ============================================================
// User account operations: info, name, password, avatar, roles
// ============================================================

import { apiClient, apiUpload } from '@/lib/api-client';
import { encryptPassword } from '@/lib/rsa';
import type {
  AccountInfo,
  BindContactPayload,
  ExpiredPasswordPayload,
  ResetPasswordPayload,
  SendAuthCodePayload,
  SendContactCodePayload,
  UpdateNamePayload,
  UpdatePasswordPayload,
  UserRolesResponse
} from './types';

type PublicKeyResponse = {
  publicKey: string;
};

/** Fetch current user account info */
export async function getAccountInfo(options?: RequestInit): Promise<AccountInfo> {
  return apiClient<AccountInfo>('/v1/account/info', options);
}

/** Update user display name */
export async function updateAccountName(
  data: UpdateNamePayload,
  options?: RequestInit
): Promise<void> {
  await apiClient<void>('/v1/account/name', {
    method: 'PUT',
    body: JSON.stringify(data),
    ...options
  });
}

/** Update user password (requires current password) */
export async function updateAccountPassword(
  data: UpdatePasswordPayload,
  options?: RequestInit
): Promise<void> {
  const { publicKey } = await apiClient<PublicKeyResponse>('/v1/auth/public-key');
  await apiClient<void>('/v1/account/password', {
    method: 'PUT',
    body: JSON.stringify({
      currentPassword: encryptPassword(publicKey, data.currentPassword),
      newPassword: encryptPassword(publicKey, data.newPassword)
    }),
    ...options
  });
}

/** Upload user avatar */
export async function uploadAccountAvatar(
  formData: FormData,
  options?: Omit<RequestInit, 'body'>
): Promise<void> {
  await apiUpload<void>('/v1/account/avatar', formData, options);
}

/** Send a verification code to the contact being bound (authenticated) */
export async function sendAccountContactCode(
  data: SendContactCodePayload,
  options?: RequestInit
): Promise<void> {
  await apiClient<void>('/v1/account/contact/code', {
    method: 'POST',
    body: JSON.stringify(data),
    ...options
  });
}

/** Verify the code and bind the contact to the current account */
export async function bindAccountContact(
  data: BindContactPayload,
  options?: RequestInit
): Promise<void> {
  await apiClient<void>('/v1/account/contact', {
    method: 'PUT',
    body: JSON.stringify(data),
    ...options
  });
}

/** Send a public verification code (sign-in or password reset) */
export async function sendAuthCode(
  data: SendAuthCodePayload,
  options?: RequestInit
): Promise<void> {
  await apiClient<void>('/v1/auth/code', {
    method: 'POST',
    body: JSON.stringify({ ...data, purpose: data.purpose ?? 'auth' }),
    ...options
  });
}

/** Reset a forgotten password using a verification code */
export async function resetAccountPassword(
  data: ResetPasswordPayload,
  options?: RequestInit
): Promise<void> {
  const { publicKey } = await apiClient<PublicKeyResponse>('/v1/auth/public-key');
  await apiClient<void>('/v1/account/password/reset', {
    method: 'PUT',
    body: JSON.stringify({
      username: data.username,
      confirmCode: data.confirmCode,
      newPassword: encryptPassword(publicKey, data.newPassword)
    }),
    ...options
  });
}

/** Fetch user roles */
export async function getUserRoles(options?: RequestInit): Promise<UserRolesResponse> {
  return apiClient<UserRolesResponse>('/v1/account/roles', options);
}

/** Replace an expired password (unauthenticated; both passwords are RSA-encrypted) */
export async function changeExpiredPassword(
  data: ExpiredPasswordPayload,
  options?: RequestInit
): Promise<void> {
  const { publicKey } = await apiClient<PublicKeyResponse>('/v1/auth/public-key');
  await apiClient<void>('/v1/auth/password/expired', {
    method: 'PUT',
    body: JSON.stringify({
      username: data.username,
      currentPassword: encryptPassword(publicKey, data.currentPassword),
      newPassword: encryptPassword(publicKey, data.newPassword),
      captchaId: data.captchaId,
      captchaCode: data.captchaCode
    }),
    ...options
  });
}

/** Turn notification emails on or off for the signed-in user */
export async function updateNotificationPreferences(
  muteNotificationEmails: boolean
): Promise<void> {
  await apiClient<void>('/v1/account/notification-preferences', {
    method: 'PUT',
    body: JSON.stringify({ muteNotificationEmails })
  });
}
