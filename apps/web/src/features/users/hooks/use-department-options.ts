'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { usePermission } from '@/hooks/use-permission';
import { departmentsQueryOptions } from '@/features/departments/api/queries';
import { flattenDepartments, indentedLabel } from '@/features/departments/tree';
import { NO_DEPARTMENT } from '../schemas/user';

/**
 * Department choices for the user form: only departments inside the caller's data
 * scope, indented by depth. "No department" is offered only when every department is
 * in scope (the backend rejects it for narrower scopes, which would hide the user).
 * `available` is false when the caller cannot list departments at all.
 */
export function useDepartmentOptions(): {
  available: boolean;
  options: { value: string; label: string }[];
} {
  const t = useTranslations('users.form');
  const canList = usePermission('/api/v1/admin/departments:GET');
  const { data: departments = [] } = useQuery({ ...departmentsQueryOptions(), enabled: canList });
  const allInScope = departments.every((d) => d.inScope);
  return {
    available: canList,
    options: [
      ...(allInScope ? [{ value: NO_DEPARTMENT, label: t('noDepartment') }] : []),
      ...flattenDepartments(departments)
        .filter(({ department }) => department.inScope)
        .map(({ department, depth }) => ({
          value: String(department.id),
          label: indentedLabel(department.name, depth)
        }))
    ]
  };
}
