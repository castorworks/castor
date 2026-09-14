// ============================================================
// Resource types — aligned with castor backend DTOs
// ============================================================

export interface Resource {
  id: number;
  code: string;
  name: string;
  description: string;
  path: string;
  actions: string[]; // ['GET', 'POST', 'PUT', 'DELETE', 'PATCH']
  category: string; // 'admin' | 'user'
  module: string;
  sortOrder: number;
  isSystem: boolean;
  isEnabled: boolean;
}

export interface ResourceFilters {
  page?: number;
  pageSize?: number;
  category?: string;
  module?: string;
  search?: string;
  sort?: string;
  isEnabled?: boolean;
}

export interface ResourcesPageResult {
  list: Resource[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface CreateResourcePayload {
  code: string;
  name: string;
  description?: string;
  path: string;
  actions: string[];
  category: string;
  module?: string;
  sortOrder?: number;
}

export interface UpdateResourcePayload {
  name: string;
  description?: string;
  path: string;
  actions: string[];
  category: string;
  module?: string;
  sortOrder?: number;
  isEnabled?: boolean;
}
