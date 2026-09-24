// ============================================================
// System Setting Service — Data Access Layer
// ============================================================
// Connected to castor REST API via apiClient.
// ============================================================

import { apiClient } from '@/lib/api-client';
import type {
  Setting,
  SettingFilters,
  CreateSettingPayload,
  UpdateSettingPayload,
  BatchUpdateSettingPayload,
  PublicSettings
} from './types';

/** Fetch all settings (optionally filtered by category) */
export async function getSettings(
  filters: SettingFilters = {},
  options?: RequestInit
): Promise<Setting[]> {
  const params = new URLSearchParams();
  if (filters.category) params.set('category', filters.category);

  const query = params.toString();
  return apiClient<Setting[]>(`/v1/admin/settings${query ? `?${query}` : ''}`, options);
}

/** Get a single setting by ID */
export async function getSetting(id: number): Promise<Setting> {
  return apiClient<Setting>(`/v1/admin/settings/${id}`);
}

/** Get a setting by key */
export async function getSettingByKey(key: string): Promise<Setting> {
  return apiClient<Setting>(`/v1/admin/settings/key/${key}`);
}

/** Create a new setting */
export async function createSetting(data: CreateSettingPayload): Promise<Setting> {
  return apiClient<Setting>('/v1/admin/settings', {
    method: 'POST',
    body: JSON.stringify(data)
  });
}

/** Update an existing setting */
export async function updateSetting(id: number, data: UpdateSettingPayload): Promise<Setting> {
  return apiClient<Setting>(`/v1/admin/settings/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data)
  });
}

/** Batch update settings */
export async function batchUpdateSettings(data: BatchUpdateSettingPayload): Promise<void> {
  await apiClient<void>('/v1/admin/settings/batch', {
    method: 'PUT',
    body: JSON.stringify(data)
  });
}

/** Delete a setting */
export async function deleteSetting(id: number): Promise<void> {
  await apiClient<void>(`/v1/admin/settings/${id}`, {
    method: 'DELETE'
  });
}

/** Fetch public settings available before authentication */
export async function getPublicSettings(options?: RequestInit): Promise<PublicSettings> {
  return apiClient<PublicSettings>('/v1/settings/public', options);
}
