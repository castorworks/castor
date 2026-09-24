// ============================================================
// Asset Service — Data Access Layer
// ============================================================
// Connected to castor REST API via apiClient.
// ============================================================

import { apiClient, apiUpload, type CastorListResponse } from '@/lib/api-client';
import { buildCastorOrder } from '@/lib/castor-query';
import type {
  Asset,
  AssetAttachment,
  AssetDetail,
  AssetFilters,
  AssetUploadResult,
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
  if (filters.scope) {
    params.set(filters.scope.includes(',') ? 'scope-in' : 'scope-eq', filters.scope);
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

/** Get a single asset by ID, including the business records that reference it */
export async function getAsset(id: number, options?: RequestInit): Promise<AssetDetail> {
  const res = await apiClient<AssetDetail>(`/v1/admin/assets/${id}`, options);
  return { ...res, references: res.references ?? [] };
}

/** Upload a file into the asset library (multipart/form-data) */
export async function createAsset(formData: FormData): Promise<AssetUploadResult> {
  return apiUpload<AssetUploadResult>('/v1/admin/assets', formData);
}

/**
 * Upload a business attachment through the owning module's own upload route
 * (e.g. `/v1/admin/notifications/attachments`), so its permission — not the asset
 * library's — decides who may upload.
 */
export async function uploadAttachment(path: string, file: File): Promise<AssetAttachment> {
  const formData = new FormData();
  formData.append('file', file);
  const res = await apiUpload<AssetAttachment>(path, formData);
  // 显示名用上传者本地的文件名：去重命中时资产自己的名字属于最早的上传者。
  return { ...res, name: file.name };
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
