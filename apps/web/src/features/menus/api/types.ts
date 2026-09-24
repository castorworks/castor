import type { Resource } from '@/features/resources/api/types';
export type MenuKind = 'directory' | 'page' | 'action';
export interface MenuPermission {
  resourceId: number;
  action: string;
}
export interface MenuPayload {
  parentId: number | null;
  code: string;
  kind: MenuKind;
  titles: Record<'en' | 'zh' | 'ja' | 'ko', string>;
  path: string;
  icon: string;
  sortOrder: number;
  isEnabled: boolean;
  accessMode: 'authenticated' | 'permission';
  permissions: MenuPermission[];
}
export interface Menu extends MenuPayload {
  id: number;
}
export interface MenuCatalog {
  menus: Menu[];
  resources: Resource[];
}
export interface MenuRoute {
  path: string;
  allowed: boolean;
}
export interface Navigation {
  items: Menu[];
  routes: MenuRoute[];
}
