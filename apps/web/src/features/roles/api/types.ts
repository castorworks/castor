import type { MenuCatalog } from '@/features/menus/api/types';
// ============================================================
// Role types — aligned with castor backend DTOs
// ============================================================

/** Which users' data members of a role can see (dictionary `role_data_scope`). */
export const DATA_SCOPES = ['ALL', 'DEPT_AND_CHILDREN', 'DEPT', 'CUSTOM', 'SELF'] as const;
export type DataScope = (typeof DATA_SCOPES)[number];

export interface Role {
  id: number;
  code: string;
  name: string;
  description: string;
  isSystem: boolean;
  isEnabled: boolean;
  userCount?: number;
  dataScope: DataScope;
  /** Only for CUSTOM */
  departmentIds: number[] | null;
}

export interface RoleDetail extends Role {
  users: string[];
  permissions: RolePermission[];
}

export interface RolePermission {
  resourceId: number;
  resourceCode: string;
  resourceName: string;
  resourcePath: string;
  actions: string[];
  isSystem: boolean;
}

export interface RoleWithPermissions extends Role {
  permissions: RolePermission[];
}

export interface CreateRolePayload {
  code: string;
  name: string;
  description?: string;
}

export interface UpdateRolePayload {
  name?: string;
  description?: string;
  isEnabled?: boolean;
}

export interface PermissionGrant {
  resourceId: number;
  actions: string[];
}

export type ConstraintType = 'SSD' | 'DSD';

export interface SeparationConstraint {
  id: number;
  code: string;
  name: string;
  description: string;
  type: ConstraintType;
  cardinality: number;
  roleIds: number[];
  isEnabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface ConstraintPayload {
  code: string;
  name: string;
  description?: string;
  type: ConstraintType;
  cardinality: number;
  roleIds: number[];
  isEnabled: boolean;
}

export interface RolePermissionCatalog extends MenuCatalog {
  grants: RolePermission[];
}
