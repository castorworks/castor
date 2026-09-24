import { mutationOptions } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { deleteMenu, saveMenu } from './service';
import { menuKeys } from './queries';
import type { MenuPayload } from './types';
async function refreshMenus() {
  const qc = getQueryClient();
  await Promise.all([
    qc.invalidateQueries({ queryKey: menuKeys.all }),
    qc.invalidateQueries({ queryKey: ['account', 'navigation'] }),
    qc.invalidateQueries({ queryKey: ['roles'] })
  ]);
}
export const saveMenuMutation = mutationOptions({
  mutationFn: ({ id, data }: { id?: number; data: MenuPayload }) => saveMenu(id, data),
  onSuccess: refreshMenus
});
export const deleteMenuMutation = mutationOptions({
  mutationFn: deleteMenu,
  onSuccess: refreshMenus
});
