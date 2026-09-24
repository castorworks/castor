import { mutationOptions } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { deleteOIDCProvider, saveOIDCProvider, unlinkIdentity } from './service';
import { ssoKeys } from './queries';
import type { OIDCProviderPayload } from './types';

async function refresh() {
  await getQueryClient().invalidateQueries({ queryKey: ssoKeys.all });
}

export const saveOIDCProviderMutation = mutationOptions({
  mutationFn: ({ id, data }: { id?: number; data: OIDCProviderPayload }) =>
    saveOIDCProvider(id, data),
  onSuccess: refresh
});

export const deleteOIDCProviderMutation = mutationOptions({
  mutationFn: deleteOIDCProvider,
  onSuccess: refresh
});

export const unlinkIdentityMutation = mutationOptions({
  mutationFn: unlinkIdentity,
  onSuccess: refresh
});
