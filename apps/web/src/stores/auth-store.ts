// ============================================================
// Auth Zustand Store
// ============================================================
// Manages authentication state: token, user info, roles, and actions.
// Integrates with castor JWT authentication.
// ============================================================

import { create } from 'zustand';
import { apiClient, CastorApiError } from '@/lib/api-client';
import { getQueryClient } from '@/lib/query-client';
import { encryptPassword } from '@/lib/rsa';
import type { AccessSnapshot, EffectivePermission, RoleSummary } from '@/features/users/api/types';

// ============================================================
// Types
// ============================================================

export interface AuthUser {
  id: number;
  username: string;
  name: string;
  avatar: string;
  assignedRoles: RoleSummary[];
  authorizedRoles: RoleSummary[];
  activeRoles: RoleSummary[];
  permissions: EffectivePermission[];
  authorizationSessionId: string;
}

export interface LoginParams {
  username: string;
  credential: string;
  captchaId?: string;
  captchaCode?: string;
}

export interface LoginResponse {
  token?: string;
  expire: string;
  user: AuthUser;
}

export interface CaptchaResponse {
  id: string;
  img: string; // base64 data URI
}

export interface PublicKeyResponse {
  publicKey: string; // PEM-encoded RSA public key
}

// ============================================================
// State
// ============================================================

interface AuthState {
  // Current authenticated user (null = not authenticated)
  user: AuthUser | null;

  // Whether the initial auth check is complete
  isLoading: boolean;

  // Whether a login/logout action is in progress
  isSubmitting: boolean;

  // Last error message (cleared on next action)
  error: string | null;

  // Actions
  setUser: (user: AuthUser | null) => void;
  login: (params: LoginParams, method?: 'password' | 'email' | 'mobile') => Promise<void>;
  logout: () => Promise<void>;
  refreshToken: () => Promise<boolean>;
  fetchUserInfo: () => Promise<void>;
  setActiveRoles: (roleCodes: string[]) => Promise<void>;
  clearError: () => void;
}

// ============================================================
// Helpers
// ============================================================

/**
 * Encrypt a password using the castor RSA public key.
 * The backend uses RSA-OAEP with SHA-256, so the frontend matches that algorithm.
 */
export function rsaEncrypt(publicKeyPem: string, plainText: string): string {
  return encryptPassword(publicKeyPem, plainText);
}

/** Redirect to login path, preserving the current URL for post-login redirect */
function redirectToLogin() {
  if (typeof window === 'undefined') return;
  const currentPath = window.location.pathname + window.location.search;
  window.location.href = `/auth/sign-in?redirect=${encodeURIComponent(currentPath)}`;
}

// ============================================================
// Store
// ============================================================

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isLoading: true,
  isSubmitting: false,
  error: null,

  setUser: (user) => set({ user, error: null }),

  login: async (params, method = 'password') => {
    set({ isSubmitting: true, error: null });
    try {
      // Fetch public key for password encryption
      if (method === 'password') {
        const pubKey = await apiClient<PublicKeyResponse>('/v1/auth/public-key');
        params = {
          ...params,
          credential: rsaEncrypt(pubKey.publicKey, params.credential)
        };
      }

      const data = await apiClient<LoginResponse>(`/v1/auth/login?method=${method}`, {
        method: 'POST',
        body: JSON.stringify({
          username: params.username,
          credential: params.credential,
          captchaId: params.captchaId,
          captchaCode: params.captchaCode
        })
      });

      set({ user: data.user, isSubmitting: false, error: null });
    } catch (err) {
      const message = err instanceof CastorApiError ? err.message : 'Login failed';
      set({ error: message, isSubmitting: false });
      throw err;
    }
  },

  logout: async () => {
    set({ isSubmitting: true });
    try {
      await apiClient('/v1/auth/logout', { method: 'POST' });
    } catch {
      // Ignore logout errors
    } finally {
      set({ user: null, isSubmitting: false, error: null });
      redirectToLogin();
    }
  },

  refreshToken: async () => {
    try {
      await apiClient<{ token?: string; expire: string }>('/v1/auth/refresh-token', {
        method: 'POST'
      });
      return true;
    } catch {
      return false;
    }
  },

  fetchUserInfo: async () => {
    try {
      const user = await apiClient<AuthUser>('/v1/account/info');
      const access = await apiClient<AccessSnapshot>('/v1/account/permissions');
      user.assignedRoles = access.assignedRoles;
      user.authorizedRoles = access.authorizedRoles;
      user.activeRoles = access.activeRoles;
      user.permissions = access.permissions;
      user.authorizationSessionId = access.sessionId ?? '';
      set({ user, isLoading: false, error: null });
    } catch {
      set({ user: null, isLoading: false, error: null });
    }
  },

  setActiveRoles: async (roleCodes) => {
    const access = await apiClient<AccessSnapshot>('/v1/account/session/roles', {
      method: 'PUT',
      body: JSON.stringify({ roleCodes })
    });
    set((state) =>
      state.user
        ? {
            user: {
              ...state.user,
              assignedRoles: access.assignedRoles,
              authorizedRoles: access.authorizedRoles,
              activeRoles: access.activeRoles,
              permissions: access.permissions,
              authorizationSessionId: access.sessionId ?? state.user.authorizationSessionId
            }
          }
        : state
    );
    await Promise.all([
      getQueryClient().invalidateQueries({ queryKey: ['account'] }),
      getQueryClient().invalidateQueries({ queryKey: ['users', 'permissions', 'account'] })
    ]);
  },

  clearError: () => set({ error: null })
}));

// ============================================================
// Convenience Hooks & Selectors
// ============================================================

export function useUser() {
  return useAuthStore((s) => s.user);
}

export function useIsAuthenticated() {
  return useAuthStore((s) => s.user !== null);
}
