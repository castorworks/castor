// ============================================================
// Auth Provider
// ============================================================
// Client component that initializes the auth store on mount.
// Wraps the app and fetches user info if a JWT cookie exists.
// Does NOT block rendering — the proxy.ts middleware handles
// redirect for unauthenticated users on protected routes.
// ============================================================

'use client';

import { useEffect } from 'react';
import { usePathname } from 'next/navigation';
import { useAuthStore } from '@/stores/auth-store';

const ACCOUNT_BOOTSTRAP_PREFIXES = ['/dashboard'];

export function shouldFetchUserInfo(pathname: string): boolean {
  return ACCOUNT_BOOTSTRAP_PREFIXES.some((prefix) => pathname.startsWith(prefix));
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  useEffect(() => {
    if (!shouldFetchUserInfo(pathname)) return;

    // 使用 getState() 获取稳定引用，避免订阅整个 store 导致不必要的重渲染
    // fetchUserInfo 是 Zustand store 中定义一次的稳定函数引用
    useAuthStore.getState().fetchUserInfo();
  }, [pathname]);

  return <>{children}</>;
}
