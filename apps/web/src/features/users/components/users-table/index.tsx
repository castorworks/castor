'use client';

import { DataTable } from '@/components/ui/table/data-table';
import { DataTableToolbar } from '@/components/ui/table/data-table-toolbar';
import { useDataTable } from '@/hooks/use-data-table';
import { usePermission } from '@/hooks/use-permission';
import { ExportButton } from '@/components/data-io/export-button';
import { exportUsers } from '../../api/service';

import { useSuspenseQuery } from '@tanstack/react-query';
import { parseAsInteger, parseAsString, useQueryStates } from 'nuqs';
import { getSortingStateParser } from '@/lib/parsers';
import { usersQueryOptions } from '../../api/queries';
import { useUserColumns, userColumnIds } from './columns';

const columnIds = userColumnIds;

export function UsersTable() {
  const columns = useUserColumns();
  const [params] = useQueryStates({
    page: parseAsInteger.withDefault(1),
    perPage: parseAsInteger.withDefault(10),
    search: parseAsString,
    accountSource: parseAsString,
    status: parseAsString,
    department: parseAsString,
    sort: getSortingStateParser(columnIds).withDefault([])
  });

  const filters = {
    page: params.page,
    pageSize: params.perPage,
    ...(params.search && { search: params.search }),
    ...(params.accountSource && { accountSource: params.accountSource }),
    ...(params.status && { status: params.status }),
    ...(params.department && { department: params.department }),
    ...(params.sort.length > 0 && { sort: JSON.stringify(params.sort) })
  };

  const { data } = useSuspenseQuery(usersQueryOptions(filters));

  const canExport = usePermission('/api/v1/admin/users/export:GET');

  const { table } = useDataTable({
    data: data.list,
    columns,
    rowCount: data.total,
    shallow: true,
    debounceMs: 500,
    initialState: {
      columnPinning: { right: ['actions'] }
    }
  });

  return (
    <DataTable table={table}>
      <DataTableToolbar table={table}>
        {canExport && <ExportButton onExport={(format) => exportUsers(filters, format)} />}
      </DataTableToolbar>
    </DataTable>
  );
}

export function UsersTableSkeleton() {
  return (
    <div className='flex flex-1 animate-pulse flex-col gap-4'>
      <div className='bg-muted h-10 w-full rounded' />
      <div className='bg-muted h-96 w-full rounded-lg' />
      <div className='bg-muted h-10 w-full rounded' />
    </div>
  );
}
