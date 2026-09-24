// ============================================================
// User types — aligned with castor backend DTOs
// ============================================================

export interface User {
  id: number;
  username: string;
  name: string;
  avatar: string;
  accountSource: string; // INTERNAL | FACEBOOK | QQ etc.
  email: string;
  mobile: string;
  emailVerified: boolean;
  mobileVerified: boolean;
  enable: boolean;
  locked: boolean;
  accountExpireDate: string | null; // ISO 8601 or null (never expires)
  credentialExpireDate: string | null; // ISO 8601 or null (never expires)
  departmentId: number | null;
  /** Two-factor authentication is on; only filled in by the admin user list and detail. */
  totpEnabled?: boolean;
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
  /** Department id; the list includes its sub-departments. */
  department?: string;
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
  email?: string;
  mobile?: string;
  departmentId?: number | null;
};

/** Payload for updating a user (PUT /admin/users/:id) */
export type UpdateUserPayload = {
  name?: string;
  avatar?: string;
  email?: string;
  mobile?: string;
  password?: string;
  enable?: boolean;
  locked?: boolean;
  accountExpireDate?: string | null;
  credentialExpireDate?: string | null;
  /** 0 removes the user from their department. */
  departmentId?: number;
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

/** A problem with one field of one row of an import file. */
export interface UserImportError {
  /** Line number in the file (the header is line 1). */
  line: number;
  /** Language-neutral field key: username, name, email, mobile, departmentCode. */
  field: string;
  code: string;
  /** Localized by the backend. */
  message: string;
}

export interface UserImportResult {
  created: number;
  errors: UserImportError[] | null;
}
