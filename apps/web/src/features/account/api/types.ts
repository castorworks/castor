import type { AccessSnapshot } from '@/features/users/api/types';

// ============================================================
// Account types — aligned with castor backend DTOs
// ============================================================

export interface AccountInfo {
  id: number;
  username: string;
  name: string;
  avatar: string;
  accountSource: string;
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

export type UserRolesResponse = AccessSnapshot;
