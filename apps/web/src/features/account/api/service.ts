// ============================================================
// Account Service — Data Access Layer
// ============================================================
// User account operations: info, name, password, avatar, roles
// ============================================================

import { apiClient, apiUpload } from '@/lib/api-client';
import { encryptPassword } from '@/lib/rsa';
import type {
  AccountInfo,
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

/** Fetch user roles */
export async function getUserRoles(options?: RequestInit): Promise<UserRolesResponse> {
  return apiClient<UserRolesResponse>('/v1/account/roles', options);
}
