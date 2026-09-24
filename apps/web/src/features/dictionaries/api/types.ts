// ============================================================
// Dictionary types — aligned with castor backend DTOs
// ============================================================

import type { I18nText } from '@/lib/i18n-text';

export interface DictType {
  id: number;
  code: string;
  name: I18nText;
  description: string;
  sortOrder: number;
  isSystem: boolean;
  /** Whether the type is served by the unauthenticated `/v1/dictionaries` endpoint. */
  isPublic: boolean;
  isEnabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface DictItem {
  id: number;
  typeCode: string;
  label: I18nText;
  value: string;
  description: string;
  /** A tag color name from `@/lib/tag-color`, or '' for none. */
  color: string;
  /** A key of `Icons`, or '' for none. */
  icon: string;
  /** Seeded items are referenced by code: their value is locked and they cannot be deleted. */
  isSystem: boolean;
  isDefault: boolean;
  isEnabled: boolean;
  sortOrder: number;
  createdAt: string;
  updatedAt: string;
}

/** Dictionary items grouped by type code, as served to signed-in clients. */
export type DictMap = Record<string, DictItem[]>;

export interface CreateDictTypePayload {
  code: string;
  name: I18nText;
  description?: string;
  isPublic?: boolean;
  sortOrder?: number;
}

export interface UpdateDictTypePayload {
  name?: I18nText;
  description?: string;
  isPublic?: boolean;
  isEnabled?: boolean;
  sortOrder?: number;
}

export interface CreateDictItemPayload {
  typeCode: string;
  label: I18nText;
  value: string;
  description?: string;
  color?: string;
  icon?: string;
  isDefault?: boolean;
  sortOrder?: number;
}

export interface UpdateDictItemPayload {
  label?: I18nText;
  value?: string;
  description?: string;
  color?: string;
  icon?: string;
  isDefault?: boolean;
  isEnabled?: boolean;
  sortOrder?: number;
}
