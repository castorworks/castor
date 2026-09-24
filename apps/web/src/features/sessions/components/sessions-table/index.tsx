'use client';

import { DataTable } from '@/components/ui/table/data-table';
import { DataTableToolbar } from '@/components/ui/table/data-table-toolbar';
import { useDataTable } from '@/hooks/use-data-table';
import { useSuspenseQuery } from '@tanstack/react-query';
import { parseAsInteger, parseAsString, useQueryStates } from 'nuqs';
import { getSortingStateParser } from '@/lib/parsers';
import { sessionsQueryOptions } from '../../api/queries';
import { useSessionColumns, columnIds } from './columns';

export function SessionsTable() {
  const columns = useSessionColumns();
  const [params] = useQueryStates({
    page: parseAsInteger.withDefault(1),
    perPage: parseAsInteger.withDefault(10),
    user: parseAsString,
    ip: parseAsString,
    sort: getSortingStateParser(columnIds).withDefault([])
  });

  const filters = {
    page: params.page,
    pageSize: params.perPage,
    ...(params.user && { username: params.user }),
    ...(params.ip && { ipAddr: params.ip }),
    ...(params.sort.length > 0 && { sort: JSON.stringify(params.sort) })
  };

  const { data } = useSuspenseQuery(sessionsQueryOptions(filters));

  const { table } = useDataTable({
    data: data.list,
    columns,
    rowCount: data.total,
    shallow: true,
    debounceMs: 500
  });

  return (
    <DataTable table={table}>
      <DataTableToolbar table={table} />
    </DataTable>
  );
}
