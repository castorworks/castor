// ============================================================
// Role Service — Data Access Layer
// ============================================================
// Connected to castor REST API via apiClient.
// ============================================================

import { apiClient } from '@/lib/api-client';
import type {
  Role,
  RoleWithPermissions,
  CreateRolePayload,
  RolePermissionCatalog,
  PermissionGrant,
  SeparationConstraint,
  ConstraintPayload,
  UpdateRolePayload
} from './types';

/** Fetch all roles */
export async function getRoles(options?: RequestInit): Promise<Role[]> {
  return apiClient<Role[]>('/v1/admin/roles', options);
}

/** Get a single role by code with permissions */
export async function getRole(code: string, options?: RequestInit): Promise<RoleWithPermissions> {
  return apiClient<RoleWithPermissions>(`/v1/admin/roles/${encodeURIComponent(code)}`, options);
}

/** Create a new role */
export async function createRole(data: CreateRolePayload): Promise<Role> {
  return apiClient<Role>('/v1/admin/roles', {
    method: 'POST',
    body: JSON.stringify(data)
  });
}

/** Update an existing role */
export async function updateRole(code: string, data: UpdateRolePayload): Promise<Role> {
  return apiClient<Role>(`/v1/admin/roles/${encodeURIComponent(code)}`, {
    method: 'PUT',
    body: JSON.stringify(data)
  });
}

/** Delete a role */
export async function deleteRole(code: string): Promise<void> {
  await apiClient<void>(`/v1/admin/roles/${encodeURIComponent(code)}`, {
    method: 'DELETE'
  });
}

/** Toggle role enabled status */
export async function toggleRoleEnabled(code: string, isEnabled: boolean): Promise<void> {
  await apiClient<void>(`/v1/admin/roles/${encodeURIComponent(code)}`, {
    method: 'PUT',
    body: JSON.stringify({ isEnabled })
  });
}

/** Get permissions for a role */
export async function getRolePermissions(code: string): Promise<RolePermissionCatalog> {
  return apiClient<RolePermissionCatalog>(
    `/v1/admin/roles/${encodeURIComponent(code)}/permissions`
  );
}

/** Set permissions for a role (replace all) */
export async function setRolePermissions(code: string, grants: PermissionGrant[]): Promise<void> {
  await apiClient<void>(`/v1/admin/roles/${encodeURIComponent(code)}/permissions`, {
    method: 'PUT',
    body: JSON.stringify({ grants })
  });
}

export async function getRoleHierarchy(code: string): Promise<Role[]> {
  return apiClient<Role[]>(`/v1/admin/roles/${encodeURIComponent(code)}/hierarchy`);
}

export async function setRoleHierarchy(code: string, juniorRoleCodes: string[]): Promise<void> {
  await apiClient<void>(`/v1/admin/roles/${encodeURIComponent(code)}/hierarchy`, {
    method: 'PUT',
    body: JSON.stringify({ juniorRoleCodes })
  });
}

/** Get users in a role */
export async function getRoleUsers(code: string): Promise<string[]> {
  return apiClient<string[]>(`/v1/admin/roles/${encodeURIComponent(code)}/users`);
}

export async function getConstraints(): Promise<SeparationConstraint[]> {
  return apiClient<SeparationConstraint[]>('/v1/admin/authorization/constraints');
}

export async function createConstraint(data: ConstraintPayload): Promise<SeparationConstraint> {
  return apiClient<SeparationConstraint>('/v1/admin/authorization/constraints', {
    method: 'POST',
    body: JSON.stringify(data)
  });
}

export async function updateConstraint(
  id: number,
  data: ConstraintPayload
): Promise<SeparationConstraint> {
  return apiClient<SeparationConstraint>(`/v1/admin/authorization/constraints/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data)
  });
}

export async function deleteConstraint(id: number): Promise<void> {
  await apiClient<void>(`/v1/admin/authorization/constraints/${id}`, { method: 'DELETE' });
}
