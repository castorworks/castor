// ============================================================
// Auth Utilities
// ============================================================
// Server-side helpers for reading JWT cookies and verifying tokens.
// Used by Next.js middleware (proxy.ts) and server components.
// ============================================================

import { cookies } from 'next/headers';

/**
 * Read the JWT token from the httpOnly cookie.
 * Must be called in a server context (server component, API route, middleware).
 */
export async function getAuthToken(): Promise<string | undefined> {
  const cookieStore = await cookies();
  return cookieStore.get('jwt')?.value;
}

/**
 * Determine the login redirect URL, preserving the current path.
 */
export function getLoginRedirectUrl(path: string): string {
  return `/auth/sign-in?redirect=${encodeURIComponent(path)}`;
}

/**
 * Check if the current user has any of the specified roles.
 * Note: Role information is stored in the JWT claims and must be
 * decoded on the server, or fetched from the API.
 */
export function hasAnyRole(roles: string[], requiredRoles: string[]): boolean {
  if (requiredRoles.length === 0) return true;
  return roles.some((r) => requiredRoles.includes(r));
}

/**
 * Parse the JWT payload (without verification — for client-side use only).
 * Do NOT trust this for security decisions; always verify server-side.
 * Works in both browser and Node.js (SSR) environments.
 */
export function parseJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
    const payload =
      typeof globalThis.atob === 'function'
        ? globalThis.atob(base64)
        : Buffer.from(base64, 'base64').toString('utf-8');
    return JSON.parse(payload);
  } catch {
    return null;
  }
}
