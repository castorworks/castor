import { mutationOptions } from '@tanstack/react-query';
import { getQueryClient } from '@/lib/query-client';
import { deleteDepartment, saveDepartment } from './service';
import { departmentKeys } from './queries';
import type { DepartmentPayload } from './types';

async function refreshDepartments() {
  const qc = getQueryClient();
  await Promise.all([
    qc.invalidateQueries({ queryKey: departmentKeys.all }),
    // 部门名出现在用户列表与角色数据范围里
    qc.invalidateQueries({ queryKey: ['users'] }),
    qc.invalidateQueries({ queryKey: ['roles'] })
  ]);
}

export const saveDepartmentMutation = mutationOptions({
  mutationFn: ({ id, data }: { id?: number; data: DepartmentPayload }) => saveDepartment(id, data),
  onSuccess: refreshDepartments
});

export const deleteDepartmentMutation = mutationOptions({
  mutationFn: deleteDepartment,
  onSuccess: refreshDepartments
});
