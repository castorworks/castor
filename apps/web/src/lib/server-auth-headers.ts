import { cookies } from 'next/headers';
import { apiClient } from './api-client';
import type { AccessSnapshot } from '@/features/users/api/types';

export async function getServerAuthHeaders(): Promise<HeadersInit> {
  const token = (await cookies()).get('jwt')?.value;
  if (!token) return {};

  return {
    Authorization: `Bearer ${token}`,
    Cookie: `jwt=${token}`
  };
}

export function hasServerAuthHeaders(headers: HeadersInit): boolean {
  return headers instanceof Headers
    ? headers.has('Authorization')
    : Array.isArray(headers)
      ? headers.some(([key]) => key.toLowerCase() === 'authorization')
      : 'Authorization' in headers || 'authorization' in headers;
}

export async function getServerAccountAccess(): Promise<AccessSnapshot | null> {
  const headers = await getServerAuthHeaders();
  if (!hasServerAuthHeaders(headers)) return null;

  try {
    return await apiClient<AccessSnapshot>('/v1/account/permissions', {
      headers,
      cache: 'no-store'
    });
  } catch {
    return null;
  }
}

export async function serverHasPermission(permission: string): Promise<boolean> {
  const separatorIndex = permission.lastIndexOf(':');
  if (separatorIndex <= 0) return false;
  const access = await getServerAccountAccess();
  if (!access) return false;
  const resourcePath = permission.slice(0, separatorIndex);
  const action = permission.slice(separatorIndex + 1);
  return access.permissions.some(
    (grant) => grant.resourcePath === resourcePath && grant.action === action
  );
}
