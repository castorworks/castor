import { queryOptions } from '@tanstack/react-query';
import { getAccountInfo, getUserRoles } from './service';

export const accountKeys = {
  all: ['account'] as const,
  info: () => [...accountKeys.all, 'info'] as const,
  roles: () => [...accountKeys.all, 'roles'] as const
};

export function accountInfoQueryOptions(requestInit?: RequestInit) {
  return queryOptions({
    queryKey: accountKeys.info(),
    queryFn: () => getAccountInfo(requestInit)
  });
}

export function userRolesQueryOptions(requestInit?: RequestInit) {
  return queryOptions({
    queryKey: accountKeys.roles(),
    queryFn: () => getUserRoles(requestInit)
  });
}
