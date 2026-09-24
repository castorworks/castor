import { apiClient } from '@/lib/api-client';
import type { Department, DepartmentPayload } from './types';

/** The whole department tree as a flat list (sorted by sortOrder) */
export async function getDepartments(options?: RequestInit): Promise<Department[]> {
  return (await apiClient<Department[] | null>('/v1/admin/departments', options)) ?? [];
}

export function saveDepartment(id: number | undefined, data: DepartmentPayload) {
  return apiClient<Department>(id ? `/v1/admin/departments/${id}` : '/v1/admin/departments', {
    method: id ? 'PUT' : 'POST',
    body: JSON.stringify(data)
  });
}

export function deleteDepartment(id: number) {
  return apiClient<void>(`/v1/admin/departments/${id}`, { method: 'DELETE' });
}
