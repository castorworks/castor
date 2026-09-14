import { mutationOptions } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { updateAccountName, updateAccountPassword, uploadAccountAvatar } from './service';
import { accountKeys } from './queries';
import type { UpdateNamePayload, UpdatePasswordPayload } from './types';

export const updateAccountNameMutation = mutationOptions({
  mutationFn: (data: UpdateNamePayload) => updateAccountName(data),
  onSuccess: () => {
    getQueryClient().invalidateQueries({ queryKey: accountKeys.all });
  }
});

export const updateAccountPasswordMutation = mutationOptions({
  mutationFn: (data: UpdatePasswordPayload) => updateAccountPassword(data),
  onSuccess: () => {
    // Password change doesn't invalidate cache, just triggers toast in component
  }
});

export const uploadAccountAvatarMutation = mutationOptions({
  mutationFn: (formData: FormData) => uploadAccountAvatar(formData),
  onSuccess: () => {
    getQueryClient().invalidateQueries({ queryKey: accountKeys.all });
  }
});
