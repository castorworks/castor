import type { MenuCatalog } from '@/features/menus/api/types';
// ============================================================
// Role types — aligned with castor backend DTOs
// ============================================================

export interface Role {
  id: number;
  code: string;
  name: string;
  description: string;
  isSystem: boolean;
  isEnabled: boolean;
  userCount?: number;
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
