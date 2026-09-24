'use client';

import { useAnyPermission, usePermission } from '@/hooks/use-permission';
import type { ReactNode } from 'react';

interface PermissionGuardProps {
  /** Permission string in "resource:action" format, e.g. "/v1/admin/users:PUT" */
  permission: string;
  /** Children to render when permission is granted */
  children: ReactNode;
  /** Fallback to render when permission is denied (hidden by default) */
  fallback?: ReactNode;
  /** If true, render children as disabled instead of hiding them */
  disabled?: boolean;
}

/**
 * Conditionally renders children based on user permissions.
 *
 * @example
 * // Hide button if user cannot create users
 * <PermissionGuard permission="/v1/admin/users:POST">
 *   <Button>Add User</Button>
 * </PermissionGuard>
 *
 * @example
 * // Disable button instead of hiding
 * <PermissionGuard permission="/v1/admin/users:DELETE" disabled>
 *   <Button>Delete</Button>
 * </PermissionGuard>
 */
export function PermissionGuard({
  permission,
  children,
  fallback,
  disabled
}: PermissionGuardProps) {
  const hasPermission = usePermission(permission);

  if (disabled && !hasPermission) {
    return <span className='pointer-events-none opacity-50'>{children}</span>;
  }

  if (!hasPermission) {
    return fallback ? <>{fallback}</> : null;
  }

  return <>{children}</>;
}

/**
 * Conditionally renders children if user has ANY of the given permissions.
 */
export function AnyPermissionGuard({
  permissions,
  children,
  fallback
}: {
  permissions: string[];
  children: ReactNode;
  fallback?: ReactNode;
}) {
  const hasAny = useAnyPermission(permissions);

  if (!hasAny) {
    return fallback ? <>{fallback}</> : null;
  }

  return <>{children}</>;
}
