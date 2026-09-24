import { mutationOptions } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { createSetting, updateSetting, deleteSetting, batchUpdateSettings } from './service';
import { settingKeys } from './queries';
import type {
  CreateSettingPayload,
  UpdateSettingPayload,
  BatchUpdateSettingPayload
} from './types';

export const createSettingMutation = mutationOptions({
  mutationFn: (data: CreateSettingPayload) => createSetting(data),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: settingKeys.all });
  }
});

export const updateSettingMutation = mutationOptions({
  mutationFn: ({ id, values }: { id: number; values: UpdateSettingPayload }) =>
    updateSetting(id, values),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: settingKeys.all });
  }
});

export const batchUpdateSettingsMutation = mutationOptions({
  mutationFn: (data: BatchUpdateSettingPayload) => batchUpdateSettings(data),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: settingKeys.all });
  }
});

export const deleteSettingMutation = mutationOptions({
  mutationFn: (id: number) => deleteSetting(id),
  onSuccess: async () => {
    await getQueryClient().invalidateQueries({ queryKey: settingKeys.all });
  }
});
