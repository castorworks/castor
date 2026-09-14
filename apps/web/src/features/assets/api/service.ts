// ============================================================
// Asset Service — Data Access Layer
// ============================================================
// Connected to castor REST API via apiClient.
// ============================================================

import { apiClient, apiUpload, type CastorListResponse } from '@/lib/api-client';
import { buildCastorOrder } from '@/lib/castor-query';
import type {
  Asset,
  AssetFilters,
  AssetStats,
  AssetsPageResult,
  UpdateAssetPayload,
  UpdateAssetStatusPayload,
  MoveAssetPayload,
  BatchDeletePayload,
  BatchStatusPayload
} from './types';

const ASSET_ORDER_FIELDS: Record<string, string> = {
  asset: 'name',
  category: 'category',
  status: 'status',
  folderPath: 'folder_path',
  createdAt: 'created_at'
};

/** Fetch paginated asset list */
export async function getAssets(
  filters: AssetFilters,
  options?: RequestInit
): Promise<AssetsPageResult> {
  const params = new URLSearchParams();
  if (filters.page) params.set('page', String(filters.page));
  if (filters.pageSize) params.set('pageSize', String(filters.pageSize));
  if (filters.search) params.set('searchText', filters.search);
  if (filters.search) params.set('searchFields', 'name,filename,description,tags');
  if (filters.category) {
    if (filters.category.includes(',')) {
      params.set('category-in', filters.category);
    } else {
      params.set('category-eq', filters.category);
    }
  }
  if (filters.status) {
    if (filters.status.includes(',')) {
      params.set('status-in', filters.status);
    } else {
      params.set('status-eq', filters.status);
    }
  }
  if (filters.folderPath) params.set('folder_path-like', filters.folderPath);
  const order = buildCastorOrder(filters.sort, ASSET_ORDER_FIELDS);
  if (order) params.set('order', order);

  const query = params.toString();
  const res = await apiClient<CastorListResponse<Asset>>(
    `/v1/admin/assets${query ? `?${query}` : ''}`,
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

/** Get asset statistics */
export async function getAssetStats(options?: RequestInit): Promise<AssetStats> {
  return apiClient<AssetStats>('/v1/admin/assets/stats', options);
}

/** Get assets by folder path */
export async function getAssetsByFolder(
  folderPath: string,
  options?: RequestInit
): Promise<AssetsPageResult> {
  const params = new URLSearchParams({ folderPath });
  const res = await apiClient<CastorListResponse<Asset>>(
    `/v1/admin/assets/folder?${params.toString()}`,
    options
  );
  return {
    list: res.list ?? [],
    total: res.total ?? 0,
    page: res.page ?? 1,
    pageSize: res.pageSize ?? 10,
    totalPages: res.totalPages ?? Math.ceil((res.total ?? 0) / (res.pageSize ?? 10))
  };
}

/** Get assets by category */
export async function getAssetsByCategory(
  category: string,
  options?: RequestInit
): Promise<AssetsPageResult> {
  const res = await apiClient<CastorListResponse<Asset>>(
    `/v1/admin/assets/category/${category}`,
    options
  );
  return {
    list: res.list ?? [],
    total: res.total ?? 0,
    page: res.page ?? 1,
    pageSize: res.pageSize ?? 10,
    totalPages: res.totalPages ?? Math.ceil((res.total ?? 0) / (res.pageSize ?? 10))
  };
}

/** Get assets by status */
export async function getAssetsByStatus(
  status: string,
  options?: RequestInit
): Promise<AssetsPageResult> {
  const res = await apiClient<CastorListResponse<Asset>>(
    `/v1/admin/assets/status/${status}`,
    options
  );
  return {
    list: res.list ?? [],
    total: res.total ?? 0,
    page: res.page ?? 1,
    pageSize: res.pageSize ?? 10,
    totalPages: res.totalPages ?? Math.ceil((res.total ?? 0) / (res.pageSize ?? 10))
  };
}

/** Get a single asset by ID */
export async function getAsset(id: number): Promise<Asset> {
  return apiClient<Asset>(`/v1/admin/assets/${id}`);
}

/** Upload a new asset (multipart/form-data) */
export async function createAsset(formData: FormData): Promise<Asset> {
  return apiUpload<Asset>('/v1/admin/assets', formData);
}

/** Update asset metadata */
export async function updateAsset(id: number, data: UpdateAssetPayload): Promise<Asset> {
  return apiClient<Asset>(`/v1/admin/assets/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data)
  });
}

/** Update asset status */
export async function updateAssetStatus(
  id: number,
  data: UpdateAssetStatusPayload
): Promise<Asset> {
  return apiClient<Asset>(`/v1/admin/assets/${id}/status`, {
    method: 'PUT',
    body: JSON.stringify(data)
  });
}

/** Move asset to a different folder */
export async function moveAsset(id: number, data: MoveAssetPayload): Promise<Asset> {
  return apiClient<Asset>(`/v1/admin/assets/${id}/move`, {
    method: 'PUT',
    body: JSON.stringify(data)
  });
}

/** Delete a single asset */
export async function deleteAsset(id: number): Promise<void> {
  await apiClient<void>(`/v1/admin/assets/${id}`, {
    method: 'DELETE'
  });
}

/** Batch delete assets */
export async function batchDeleteAssets(data: BatchDeletePayload): Promise<void> {
  await apiClient<void>('/v1/admin/assets/batch/delete', {
    method: 'POST',
    body: JSON.stringify(data)
  });
}

/** Batch update asset status */
export async function batchUpdateAssetStatus(data: BatchStatusPayload): Promise<void> {
  await apiClient<void>('/v1/admin/assets/batch/status', {
    method: 'PUT',
    body: JSON.stringify(data)
  });
}

/** Get download URL for an asset */
export function getAssetDownloadUrl(objectKey: string): string {
  return `/api/v1/admin/assets/download/${encodeURIComponent(objectKey)}`;
}
