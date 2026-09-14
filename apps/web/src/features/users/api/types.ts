// ============================================================
// User types — aligned with castor backend DTOs
// ============================================================

export interface User {
  id: number;
  username: string;
  name: string;
  avatar: string;
  accountSource: string; // INTERNAL | FACEBOOK | QQ etc.
  enable: boolean;
  locked: boolean;
  accountExpireDate: string | null; // ISO 8601 or null (never expires)
  credentialExpireDate: string | null; // ISO 8601 or null (never expires)
  createdAt: string;
  updatedAt: string;
}

/** Query filters passed to the service layer */
export type UserFilters = {
  page?: number;
  pageSize?: number;
  search?: string;
  accountSource?: string;
  status?: string;
  sort?: string;
};

/** Shape returned by the service after converting castor pagination */
export interface UsersPageResult {
  list: User[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

/** Payload for creating a user (POST /admin/users) */
export type CreateUserPayload = {
  username: string;
  name?: string;
  password: string;
};

/** Payload for updating a user (PUT /admin/users/:id) */
export type UpdateUserPayload = {
  name?: string;
  avatar?: string;
  password?: string;
  enable?: boolean;
  locked?: boolean;
  accountExpireDate?: string | null;
  credentialExpireDate?: string | null;
};

export interface RoleSummary {
  id: number;
  code: string;
  name: string;
  description: string;
  isSystem: boolean;
  isEnabled: boolean;
}

export interface EffectivePermission {
  roleId: number;
  roleCode: string;
  resourceId: number;
  resourceCode: string;
  resourcePath: string;
  action: string;
}

export interface AccessSnapshot {
  sessionId?: string;
  username: string;
  assignedRoles: RoleSummary[];
  authorizedRoles: RoleSummary[];
  activeRoles: RoleSummary[];
  permissions: EffectivePermission[];
}
