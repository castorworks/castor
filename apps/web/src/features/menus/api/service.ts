import { apiClient } from '@/lib/api-client';
import type { Menu, MenuPayload, MenuCatalog, Navigation } from './types';
export function getMenus(options?: RequestInit) {
  return apiClient<MenuCatalog>('/v1/admin/menus', options);
}
export function getNavigation() {
  return apiClient<Navigation>('/v1/account/navigation');
}
export function saveMenu(id: number | undefined, data: MenuPayload) {
  return apiClient<Menu>(id ? `/v1/admin/menus/${id}` : '/v1/admin/menus', {
    method: id ? 'PUT' : 'POST',
    body: JSON.stringify(data)
  });
}
export function deleteMenu(id: number) {
  return apiClient<void>(`/v1/admin/menus/${id}`, { method: 'DELETE' });
}
