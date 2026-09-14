// ============================================================
// User Service — Data Access Layer
// ============================================================
// Connected to castor REST API via apiClient.
// Only this file changes when switching backend.
// ============================================================

import { apiClient, type CastorListResponse } from '@/lib/api-client';
import type {
  User,
  UserFilters,
  UsersPageResult,
  CreateUserPayload,
  UpdateUserPayload,
  RoleSummary,
  AccessSnapshot
} from './types';
import { encryptPassword } from '../lib/rsa';

const USER_ORDER_FIELDS: Record<string, string> = {
  id: 'id',
  search: 'username',
  user: 'username',
  username: 'username',
  name: 'name',
  accountSource: 'account_source',
  createdAt: 'created_at',
  updatedAt: 'updated_at'
};

type SortingStateItem = {
  id: string;
  desc?: boolean;
};

function parseSort(sort: string): SortingStateItem | undefined {
  try {
    const parsed = JSON.parse(sort) as SortingStateItem[];
    return parsed[0];
  } catch {
    return undefined;
  }
}

export function buildUserListQuery(filters: UserFilters): URLSearchParams {
  const params = new URLSearchParams();
  if (filters.page) params.set('page', String(filters.page));
  if (filters.pageSize) params.set('pageSize', String(filters.pageSize));
  if (filters.search) {
    params.set('searchText', filters.search);
    params.set('searchFields', 'username,name');
  }
  if (filters.accountSource) params.set('account_source-in', filters.accountSource);
  if (filters.status) {
    // status filter: "active" | "disabled" | "locked"
    if (filters.status.includes('disabled')) {
      params.set('enable-eq', 'false');
    } else if (filters.status.includes('locked')) {
      params.set('locked-eq', 'true');
    } else if (filters.status.includes('active')) {
      params.set('enable-eq', 'true');
      params.set('locked-eq', 'false');
    }
  }
  if (filters.sort) {
    const sort = parseSort(filters.sort);
    const field = sort ? USER_ORDER_FIELDS[sort.id] : undefined;
    if (field) {
      params.set('order', `${field} ${sort?.desc ? 'desc' : 'asc'}`);
    }
  }

  return params;
}

/** Fetch paginated user list */
export async function getUsers(
  filters: UserFilters,
  options?: RequestInit
): Promise<UsersPageResult> {
  const query = buildUserListQuery(filters).toString();
  const res = await apiClient<CastorListResponse<User>>(
    `/v1/admin/users${query ? `?${query}` : ''}`,
    options
  );
  return {
    list: res.list ?? [],
    total: res.total ?? 0,
    page: res.page ?? filters.page ?? 1,
    pageSize: res.pageSize ?? filters.pageSize ?? 10,
    totalPages: res.totalPages ?? 0
  };
}

/** Get a single user by ID */
export async function getUser(id: number): Promise<User> {
  return apiClient<User>(`/v1/admin/users/${id}`);
}

/** Create a new user */
export async function createUser(data: CreateUserPayload): Promise<User> {
  const encryptedPassword = await encryptPassword(data.password);
  return apiClient<User>('/v1/admin/users', {
    method: 'POST',
    body: JSON.stringify({ ...data, password: encryptedPassword })
  });
}

/** Update an existing user */
export async function updateUser(id: number, data: UpdateUserPayload): Promise<User> {
  let payload = data;
  if (data.password) {
    const encryptedPassword = await encryptPassword(data.password);
    payload = { ...data, password: encryptedPassword };
  }
  return apiClient<User>(`/v1/admin/users/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload)
  });
}

/** Delete a user */
export async function deleteUser(id: number): Promise<void> {
  await apiClient<void>(`/v1/admin/users/${id}`, {
    method: 'DELETE'
  });
}

/** Get roles for a user */
export async function getUserRoles(username: string): Promise<string[]> {
  return apiClient<string[]>(`/v1/admin/authorization/users/${encodeURIComponent(username)}/roles`);
}

/** Add a role to a user */
export async function addUserRole(username: string, role: string): Promise<void> {
  await apiClient<void>(`/v1/admin/authorization/users/${encodeURIComponent(username)}/roles`, {
    method: 'POST',
    body: JSON.stringify({ role })
  });
}

/** Remove a role from a user */
export async function removeUserRole(username: string, role: string): Promise<void> {
  await apiClient<void>(
    `/v1/admin/authorization/users/${encodeURIComponent(username)}/roles/${encodeURIComponent(role)}`,
    {
      method: 'DELETE'
    }
  );
}

/** Get effective permissions for a user */
export async function getUserPermissions(username: string): Promise<AccessSnapshot> {
  return apiClient<AccessSnapshot>(
    `/v1/admin/authorization/users/${encodeURIComponent(username)}/permissions`
  );
}

/** Get effective permissions for the current account */
export async function getAccountPermissions(): Promise<AccessSnapshot> {
  return apiClient<AccessSnapshot>('/v1/account/permissions');
}

export async function setAccountActiveRoles(roleCodes: string[]): Promise<AccessSnapshot> {
  return apiClient<AccessSnapshot>('/v1/account/session/roles', {
    method: 'PUT',
    body: JSON.stringify({ roleCodes })
  });
}

/** Get all roles (from the roles endpoint) */
export async function getAllRoles(): Promise<RoleSummary[]> {
  return apiClient<RoleSummary[]>('/v1/admin/roles');
}
