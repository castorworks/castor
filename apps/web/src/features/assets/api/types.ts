// ============================================================
// Asset types — aligned with castor backend DTOs
// ============================================================

export type AssetCategory = 'IMAGE' | 'VIDEO' | 'AUDIO' | 'DOCUMENT' | 'ARCHIVE' | 'OTHER';
export type AssetStatus = 'PENDING' | 'ACTIVE' | 'ARCHIVED' | 'DELETED';

export interface Asset {
  id: number;
  name: string;
  filename: string;
  objectKey: string;
  extension: string;
  mimeType: string;
  size: number;
  sizeFormatted: string;
  category: AssetCategory;
  status: AssetStatus;
  isPublic: boolean;
  description?: string;
  tags?: string;
  folderPath?: string;
  createdBy?: number;
  createdAt: string;
  updatedAt: string;
}

export interface AssetStats {
  totalCount: number;
  totalSize: number;
  totalSizeFormatted: string;
  categoryStats: Record<AssetCategory, number>;
  statusStats: Record<AssetStatus, number>;
}

export interface AssetFilters {
  page?: number;
  pageSize?: number;
  search?: string;
  category?: string;
  status?: string;
  folderPath?: string;
  sort?: string;
}

export interface AssetsPageResult {
  list: Asset[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface CreateAssetPayload {
  file: File;
  name?: string;
  description?: string;
  tags?: string;
  folderPath?: string;
  isPublic?: boolean;
}

export interface UpdateAssetPayload {
  name?: string;
  description?: string;
  tags?: string;
  folderPath?: string;
  isPublic?: boolean;
}

export interface UpdateAssetStatusPayload {
  status: AssetStatus;
}

export interface MoveAssetPayload {
  folderId?: number;
  folderPath?: string;
}

export interface BatchDeletePayload {
  ids: number[];
}

export interface BatchStatusPayload {
  ids: number[];
  status: AssetStatus;
}
