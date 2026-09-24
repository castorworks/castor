'use client';

import { useAuthStore } from '@/stores/auth-store';
import type { EffectivePermission } from '@/features/users/api/types';

function splitPermission(permission: string): { resource: string; action: string } | null {
  const separatorIndex = permission.lastIndexOf(':');
  if (separatorIndex <= 0 || separatorIndex === permission.length - 1) return null;

  return {
    resource: permission.slice(0, separatorIndex),
    action: permission.slice(separatorIndex + 1)
  };
}

function hasPermission(
  permissions: EffectivePermission[] | undefined,
  permission: string
): boolean {
  const parsed = splitPermission(permission);
  if (!parsed || !permissions?.length) return false;

  return permissions.some((p) => p.resourcePath === parsed.resource && p.action === parsed.action);
}

/**
 * Hook to check if the current user has a specific permission.
 * Uses the format "resource:action" e.g. "/api/v1/admin/users:GET".
 *
 * @example
 * const canEdit = usePermission('/api/v1/admin/users:PUT');
 * const canView = usePermission('/api/v1/admin/users:GET');
 */
export function usePermission(permission: string): boolean {
  const permissions = useAuthStore((s) => s.user?.permissions);
  return hasPermission(permissions, permission);
}

/**
 * Hook to check if the current user has ANY of the given permissions.
 */
export function useAnyPermission(permissions: string[]): boolean {
  const grants = useAuthStore((s) => s.user?.permissions);
  return permissions.some((permission) => hasPermission(grants, permission));
}

/**
 * Hook to check if the current user has ALL of the given permissions.
 */
export function useAllPermissions(permissions: string[]): boolean {
  const grants = useAuthStore((s) => s.user?.permissions);
  return permissions.every((permission) => hasPermission(grants, permission));
}
