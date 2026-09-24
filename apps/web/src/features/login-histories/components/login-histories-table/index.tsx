'use client';

import { DataTable } from '@/components/ui/table/data-table';
import { DataTableToolbar } from '@/components/ui/table/data-table-toolbar';
import { useDataTable } from '@/hooks/use-data-table';
import { usePermission } from '@/hooks/use-permission';
import { ExportButton } from '@/components/data-io/export-button';
import { exportLoginHistories } from '../../api/service';

import { useSuspenseQuery } from '@tanstack/react-query';
import { parseAsInteger, parseAsString, useQueryStates } from 'nuqs';
import { getSortingStateParser } from '@/lib/parsers';
import { loginHistoriesQueryOptions } from '../../api/queries';
import { useLoginHistoryColumns, columnIds } from './columns';

export function LoginHistoriesTable() {
  const columns = useLoginHistoryColumns();
  const [params] = useQueryStates({
    page: parseAsInteger.withDefault(1),
    perPage: parseAsInteger.withDefault(10),
    user: parseAsString,
    loginMethod: parseAsString,
    result: parseAsString,
    sort: getSortingStateParser(columnIds).withDefault([])
  });

  const filters = {
    page: params.page,
    pageSize: params.perPage,
    ...(params.user && { username: params.user }),
    ...(params.loginMethod && { loginMethod: params.loginMethod }),
    ...(params.result && { success: params.result }),
    ...(params.sort.length > 0 && { sort: JSON.stringify(params.sort) })
  };

  const { data } = useSuspenseQuery(loginHistoriesQueryOptions(filters));

  const canExport = usePermission('/api/v1/admin/login-histories/export:GET');

  const { table } = useDataTable({
    data: data.list,
    columns,
    rowCount: data.total,
    shallow: true,
    debounceMs: 500
  });

  return (
    <DataTable table={table}>
      <DataTableToolbar table={table}>
        {canExport && <ExportButton onExport={(format) => exportLoginHistories(filters, format)} />}
      </DataTableToolbar>
    </DataTable>
  );
}

export function LoginHistoriesTableSkeleton() {
  return (
    <div className='flex flex-1 animate-pulse flex-col gap-4'>
      <div className='bg-muted h-10 w-full rounded' />
      <div className='bg-muted h-96 w-full rounded-lg' />
      <div className='bg-muted h-10 w-full rounded' />
    </div>
  );
}
