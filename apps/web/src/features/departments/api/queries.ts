import { queryOptions } from '@tanstack/react-query';
import { getDepartments } from './service';

export const departmentKeys = {
  all: ['departments'] as const
};

export const departmentsQueryOptions = (options?: RequestInit) =>
  queryOptions({
    queryKey: departmentKeys.all,
    queryFn: () => getDepartments(options),
    staleTime: 60 * 1000
  });
