// ============================================================
// Resource Service — Data Access Layer
// ============================================================
// Connected to castor REST API via apiClient.
// ============================================================

import { apiClient, type CastorListResponse } from '@/lib/api-client';
import { buildCastorOrder } from '@/lib/castor-query';
import type {
  Resource,
  ResourceFilters,
  ResourcesPageResult,
  CreateResourcePayload,
  UpdateResourcePayload
} from './types';

/** 表格列 id → 可排序的数据库列，交给后端的字段白名单校验。 */
const RESOURCE_ORDER_FIELDS: Record<string, string> = {
  search: 'name',
  path: 'path',
  category: 'category',
  module: 'module',
  status: 'is_enabled'
};

/** searchText 的作用范围，与后端 permission.Resource 的列名一致。 */
const RESOURCE_SEARCH_FIELDS = 'name,code,path';

/** 无显式排序时保持资源目录的人工展示顺序。 */
const RESOURCE_DEFAULT_ORDER = 'sort_order asc,id asc';

/** Fetch paginated resource list */
export async function getResources(
  filters: ResourceFilters,
  options?: RequestInit
): Promise<ResourcesPageResult> {
  const params = new URLSearchParams();
  if (filters.page) params.set('page', String(filters.page));
  if (filters.pageSize) params.set('pageSize', String(filters.pageSize));
  if (filters.category) params.set('category-eq', filters.category);
  if (filters.module) params.set('module-eq', filters.module);
  if (filters.isEnabled !== undefined) params.set('is_enabled-eq', String(filters.isEnabled));
  if (filters.search) {
    params.set('searchText', filters.search);
    params.set('searchFields', RESOURCE_SEARCH_FIELDS);
  }
  params.set(
    'order',
    buildCastorOrder(filters.sort, RESOURCE_ORDER_FIELDS) ?? RESOURCE_DEFAULT_ORDER
  );

  const res = await apiClient<CastorListResponse<Resource>>(
    `/v1/admin/resources?${params.toString()}`,
    options
  );
  return {
    list: res.list ?? [],
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
