import { queryOptions } from '@tanstack/react-query';
import { getConstraints, getRolePermissions, getRole, getRoleHierarchy, getRoles } from './service';

export const roleKeys = {
  all: ['roles'] as const,
  list: () => [...roleKeys.all, 'list'] as const,
  detail: (code: string) => [...roleKeys.all, 'detail', code] as const,
  permissionsAll: () => [...roleKeys.all, 'permissions'] as const,
  permissions: (code: string) => [...roleKeys.permissionsAll(), code] as const,
  usersAll: () => [...roleKeys.all, 'users'] as const,
  users: (code: string) => [...roleKeys.usersAll(), code] as const,
  hierarchyAll: () => [...roleKeys.all, 'hierarchy'] as const,
  hierarchy: (code: string) => [...roleKeys.hierarchyAll(), code] as const,
  constraints: () => [...roleKeys.all, 'constraints'] as const
};

export const rolesQueryOptions = (options?: RequestInit) =>
  queryOptions({
    queryKey: roleKeys.list(),
    queryFn: () => getRoles(options)
  });

export const roleQueryOptions = (code: string, options?: RequestInit) =>
  queryOptions({
    queryKey: roleKeys.detail(code),
    queryFn: () => getRole(code, options),
    enabled: !!code
  });

export const roleHierarchyQueryOptions = (code: string) =>
  queryOptions({
    queryKey: roleKeys.hierarchy(code),
    queryFn: () => getRoleHierarchy(code),
    enabled: !!code
  });

export const constraintsQueryOptions = () =>
  queryOptions({
    queryKey: roleKeys.constraints(),
    queryFn: getConstraints
  });

export const rolePermissionsQueryOptions = (code: string) =>
  queryOptions({ queryKey: roleKeys.permissions(code), queryFn: () => getRolePermissions(code) });
