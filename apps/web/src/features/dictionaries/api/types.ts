// ============================================================
// Dictionary types — aligned with castor backend DTOs
// ============================================================

export interface DictType {
  id: number;
  code: string;
  name: string;
  description: string;
  sortOrder: number;
  isSystem: boolean;
  isEnabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface DictItem {
  id: number;
  typeCode: string;
  label: string;
  value: string;
  description: string;
  extra: string;
  color: string;
  icon: string;
  parentId: number | null;
  isDefault: boolean;
  isEnabled: boolean;
  sortOrder: number;
  createdAt: string;
  updatedAt: string;
}

export interface CreateDictTypePayload {
  code: string;
  name: string;
  description?: string;
  sortOrder?: number;
}

export interface UpdateDictTypePayload {
  name?: string;
  description?: string;
  isEnabled?: boolean;
  sortOrder?: number;
}

export interface CreateDictItemPayload {
  typeCode: string;
  label: string;
  value: string;
  description?: string;
  extra?: string;
  color?: string;
  icon?: string;
  parentId?: number | null;
  isDefault?: boolean;
  sortOrder?: number;
}

export interface UpdateDictItemPayload {
  label?: string;
  value?: string;
  description?: string;
  extra?: string;
  color?: string;
  icon?: string;
  parentId?: number | null;
  isDefault?: boolean;
  isEnabled?: boolean;
  sortOrder?: number;
}
