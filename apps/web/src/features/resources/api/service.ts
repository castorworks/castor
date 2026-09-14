// ============================================================
// Resource Service — Data Access Layer
// ============================================================
// Connected to castor REST API via apiClient.
// ============================================================

import { apiClient } from '@/lib/api-client';
import type {
  Resource,
  ResourceFilters,
  ResourcesPageResult,
  CreateResourcePayload,
  UpdateResourcePayload
} from './types';

interface ResourceListResponse {
  items: Resource[];
  total: number;
  page: number;
  pageSize: number;
  totalPages?: number;
}

/** Fetch paginated resource list */
export async function getResources(
  filters: ResourceFilters,
  options?: RequestInit
): Promise<ResourcesPageResult> {
  const params = new URLSearchParams();
  if (filters.page) params.set('page', String(filters.page));
  if (filters.pageSize) params.set('pageSize', String(filters.pageSize));
  if (filters.category) params.set('category', filters.category);
  if (filters.module) params.set('module', filters.module);
  if (filters.search) params.set('search', filters.search);
  if (filters.isEnabled !== undefined) params.set('isEnabled', String(filters.isEnabled));

  const query = params.toString();
  const res = await apiClient<ResourceListResponse>(
    `/v1/admin/resources${query ? `?${query}` : ''}`,
    options
  );
  return {
    list: res.items ?? [],
    total: res.total ?? 0,
    page: res.page ?? filters.page ?? 1,
    pageSize: res.pageSize ?? filters.pageSize ?? 10,
    totalPages:
      res.totalPages ?? Math.ceil((res.total ?? 0) / (res.pageSize ?? filters.pageSize ?? 10))
  };
}

/** Get a single resource by ID */
export async function getResource(id: number): Promise<Resource> {
  return apiClient<Resource>(`/v1/admin/resources/${id}`);
}

/** Create a new resource */
export async function createResource(data: CreateResourcePayload): Promise<Resource> {
  return apiClient<Resource>('/v1/admin/resources', {
    method: 'POST',
    body: JSON.stringify(data)
  });
}

/** Update an existing resource */
export async function updateResource(id: number, data: UpdateResourcePayload): Promise<Resource> {
  return apiClient<Resource>(`/v1/admin/resources/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data)
  });
}

/** Delete a resource */
export async function deleteResource(id: number): Promise<void> {
  await apiClient<void>(`/v1/admin/resources/${id}`, {
    method: 'DELETE'
  });
}

/** Get all unique modules */
export async function getResourceModules(options?: RequestInit): Promise<string[]> {
  return apiClient<string[]>('/v1/admin/resources/modules', options);
}
