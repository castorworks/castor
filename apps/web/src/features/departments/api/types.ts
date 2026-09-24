// ============================================================
// Department types — aligned with castor backend dto.DepartmentResp
// ============================================================

export interface Department {
  id: number;
  parentId: number | null;
  code: string;
  name: string;
  sortOrder: number;
  isEnabled: boolean;
  memberCount: number;
  /** Inside the caller's data scope: only these can be edited, used as parents or assigned to users. */
  inScope: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface DepartmentPayload {
  parentId: number | null;
  code: string;
  name: string;
  sortOrder: number;
  isEnabled: boolean;
}
