import { queryOptions } from '@tanstack/react-query';
import {
  getUsers,
  getUserRoles,
  getUserPermissions,
  getAccountPermissions,
  getAllRoles
} from './service';
import type { User, UserFilters } from './types';

export type { User };

export const userKeys = {
  all: ['users'] as const,
  list: (filters: UserFilters) => [...userKeys.all, 'list', filters] as const,
  detail: (id: number) => [...userKeys.all, 'detail', id] as const,
  rolesAll: () => [...userKeys.all, 'roles'] as const,
  roles: (username: string) => [...userKeys.rolesAll(), username] as const,
  permissionsAll: () => [...userKeys.all, 'permissions'] as const,
  permissions: (username: string) => [...userKeys.permissionsAll(), username] as const,
  accountPermissions: () => [...userKeys.permissionsAll(), 'account'] as const,
  allRoles: () => [...userKeys.all, 'all-roles'] as const
};

export const usersQueryOptions = (filters: UserFilters, options?: RequestInit) =>
  queryOptions({
    queryKey: userKeys.list(filters),
    queryFn: () => getUsers(filters, options)
  });

export const userRolesQueryOptions = (username: string) =>
  queryOptions({
    queryKey: userKeys.roles(username),
    queryFn: () => getUserRoles(username),
    enabled: !!username
  });

export const userPermissionsQueryOptions = (username: string) =>
  queryOptions({
    queryKey: userKeys.permissions(username),
    queryFn: () => getUserPermissions(username),
    enabled: !!username
  });

export const accountPermissionsQueryOptions = () =>
  queryOptions({
    queryKey: userKeys.accountPermissions(),
    queryFn: () => getAccountPermissions()
  });

export const allRolesQueryOptions = () =>
  queryOptions({
    queryKey: userKeys.allRoles(),
    queryFn: () => getAllRoles()
  });
