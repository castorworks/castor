// ============================================================
// Asset types — aligned with castor backend DTOs
// ============================================================

export type AssetCategory = 'IMAGE' | 'VIDEO' | 'AUDIO' | 'DOCUMENT' | 'ARCHIVE' | 'OTHER';
export type AssetStatus = 'ACTIVE' | 'ARCHIVED';
/** LIBRARY：运营在资产库中管理；ATTACHMENT：业务模块上传，随最后一个引用解除而回收。 */
export type AssetScope = 'LIBRARY' | 'ATTACHMENT';

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
  scope: AssetScope;
  isPublic: boolean;
  description?: string;
  tags?: string;
  folderPath?: string;
  createdBy?: number;
  createdAt: string;
  updatedAt: string;
}

/**
 * 业务记录上的一个附件（后端 `AssetAttachmentResp`）。`name` 是引用方登记的显示名，
 * 不是资产自己的名字：去重会让多个上传共用一条资产记录。
 */
export interface AssetAttachment {
  objectKey: string;
  name: string;
  extension: string;
  mimeType: string;
  size: number;
  sizeFormatted: string;
  category: AssetCategory;
}

/** 业务记录对资产的一次引用，如 { ownerType: 'user', ownerId: 42, field: 'avatar' }。 */
export interface AssetReference {
  ownerType: string;
  ownerId: number;
  field: string;
}

/** 资产详情：附带引用方，解释资产为何不可删除。 */
export interface AssetDetail extends Asset {
  references: AssetReference[];
}

/** 上传响应：命中去重时返回的是已有资产，其名称、可见性不受本次请求影响。 */
export interface AssetUploadResult extends Asset {
  isDuplicate: boolean;
}

export interface AssetStats {
  totalCount: number;
  totalSize: number;
  totalSizeFormatted: string;
  categoryStats: Record<AssetCategory, number>;
  statusStats: Record<AssetStatus, number>;
}

/**
 * 「资产管理」列表缺省的范围：只列运营自己管理的资产库。业务附件（头像等）数量大、
 * 生命周期由业务决定，需要排查时再显式筛选出来。
 */
export const DEFAULT_ASSET_SCOPE: AssetScope = 'LIBRARY';

export interface AssetFilters {
  page?: number;
  pageSize?: number;
  search?: string;
  category?: string;
  status?: string;
  scope?: string;
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
  folderPath: string;
}

export interface BatchDeletePayload {
  ids: number[];
}

export interface BatchStatusPayload {
  ids: number[];
  status: AssetStatus;
}
