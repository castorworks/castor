// ============================================================
// System Setting types — aligned with castor backend DTOs
// ============================================================

export type SettingType = 'STRING' | 'NUMBER' | 'BOOL' | 'JSON' | 'ARRAY' | 'SECRET';
export type LoginMethod = 'password' | 'email' | 'mobile';

export interface SettingOption {
  label: string;
  value: string;
}

export interface Setting {
  id: number;
  key: string;
  name: string;
  value: string;
  type: SettingType;
  category: string;
  description: string;
  defaultVal: string;
  options?: SettingOption[];
  isPublic: boolean;
  isSystem: boolean;
  sortOrder: number;
  createdAt: string;
  updatedAt: string;
}

export interface SettingFilters {
  category?: string;
}

export interface CreateSettingPayload {
  key: string;
  name: string;
  value?: string;
  type?: SettingType;
  category?: string;
  description?: string;
  defaultVal?: string;
  isPublic?: boolean;
  sortOrder?: number;
}

export interface UpdateSettingPayload {
  name?: string;
  value?: string;
  description?: string;
  isPublic?: boolean;
  sortOrder?: number;
}

export interface BatchUpdateSettingPayload {
  settings: { key: string; value: string }[];
}

export interface PublicSettings {
  [key: string]: unknown;
  'security.login.allowedMethods'?: LoginMethod[];
}
