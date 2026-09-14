import { queryOptions } from '@tanstack/react-query';
import { getPublicSettings, getSettings } from './service';
import type { SettingFilters } from './types';

export const settingKeys = {
  all: ['settings'] as const,
  list: (filters: SettingFilters) => [...settingKeys.all, 'list', filters] as const,
  detail: (id: number) => [...settingKeys.all, 'detail', id] as const,
  public: () => [...settingKeys.all, 'public'] as const
};

export const settingsQueryOptions = (filters: SettingFilters = {}, options?: RequestInit) =>
  queryOptions({
    queryKey: settingKeys.list(filters),
    queryFn: () => getSettings(filters, options)
  });

export const publicSettingsQueryOptions = (options?: RequestInit) =>
  queryOptions({
    queryKey: settingKeys.public(),
    queryFn: () => getPublicSettings(options)
  });
