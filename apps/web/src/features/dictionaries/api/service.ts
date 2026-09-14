// ============================================================
// Dictionary Service — Data Access Layer
// ============================================================
// Connected to castor REST API via apiClient.
// ============================================================

import { apiClient } from '@/lib/api-client';
import type {
  DictType,
  DictItem,
  CreateDictTypePayload,
  UpdateDictTypePayload,
  CreateDictItemPayload,
  UpdateDictItemPayload
} from './types';

// ============================================================
// Dictionary Types
// ============================================================

/** Fetch all dictionary types */
export async function getDictTypes(options?: RequestInit): Promise<DictType[]> {
  return apiClient<DictType[]>('/v1/admin/dict-types', options);
}

/** Get a dictionary type by ID */
export async function getDictType(id: number): Promise<DictType> {
  return apiClient<DictType>(`/v1/admin/dict-types/${id}`);
}

/** Create a dictionary type */
export async function createDictType(data: CreateDictTypePayload): Promise<DictType> {
  return apiClient<DictType>('/v1/admin/dict-types', {
    method: 'POST',
    body: JSON.stringify(data)
  });
}

/** Update a dictionary type */
export async function updateDictType(id: number, data: UpdateDictTypePayload): Promise<DictType> {
  return apiClient<DictType>(`/v1/admin/dict-types/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data)
  });
}

/** Delete a dictionary type */
export async function deleteDictType(id: number): Promise<void> {
  await apiClient<void>(`/v1/admin/dict-types/${id}`, {
    method: 'DELETE'
  });
}

/** Toggle a dictionary type enabled/disabled */
export async function toggleDictType(id: number, enabled: boolean): Promise<DictType> {
  return apiClient<DictType>(`/v1/admin/dict-types/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ isEnabled: enabled })
  });
}

// ============================================================
// Dictionary Items
// ============================================================

/** Fetch all dictionary items */
export async function getDictItems(options?: RequestInit): Promise<DictItem[]> {
  return apiClient<DictItem[]>('/v1/admin/dict-items', options);
}

/** Fetch dictionary items by type code */
export async function getDictItemsByType(
  typeCode: string,
  options?: RequestInit
): Promise<DictItem[]> {
  return apiClient<DictItem[]>(`/v1/admin/dict-items/type/${typeCode}`, options);
}

/** Get a dictionary item by ID */
export async function getDictItem(id: number): Promise<DictItem> {
  return apiClient<DictItem>(`/v1/admin/dict-items/${id}`);
}

/** Create a dictionary item */
export async function createDictItem(data: CreateDictItemPayload): Promise<DictItem> {
  return apiClient<DictItem>('/v1/admin/dict-items', {
    method: 'POST',
    body: JSON.stringify(data)
  });
}

/** Update a dictionary item */
export async function updateDictItem(id: number, data: UpdateDictItemPayload): Promise<DictItem> {
  return apiClient<DictItem>(`/v1/admin/dict-items/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data)
  });
}

/** Delete a dictionary item */
export async function deleteDictItem(id: number): Promise<void> {
  await apiClient<void>(`/v1/admin/dict-items/${id}`, {
    method: 'DELETE'
  });
}

/** Toggle a dictionary item enabled/disabled */
export async function toggleDictItem(id: number, enabled: boolean): Promise<DictItem> {
  return apiClient<DictItem>(`/v1/admin/dict-items/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ isEnabled: enabled })
  });
}

// ============================================================
// Public Dictionary API (no auth required for cached consumption)
// ============================================================

/** Fetch all public dictionaries grouped by type code */
export async function getAllPublicDicts(
  options?: RequestInit
): Promise<Record<string, DictItem[]>> {
  return apiClient<Record<string, DictItem[]>>('/v1/dictionaries', options);
}

/** Fetch public dictionary items by type code */
export async function getPublicDictItems(
  typeCode: string,
  options?: RequestInit
): Promise<DictItem[]> {
  return apiClient<DictItem[]>(`/v1/dictionaries/${typeCode}`, options);
}
