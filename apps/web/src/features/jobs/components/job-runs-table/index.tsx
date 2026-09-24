'use client';

import { DataTable } from '@/components/ui/table/data-table';
import { DataTableToolbar } from '@/components/ui/table/data-table-toolbar';
import { useDataTable } from '@/hooks/use-data-table';
import { useSuspenseQuery } from '@tanstack/react-query';
import { parseAsInteger, parseAsString, useQueryStates } from 'nuqs';
import { getSortingStateParser } from '@/lib/parsers';
import { jobRunsQueryOptions } from '../../api/queries';
import { useJobRunColumns, columnIds } from './columns';

export function JobRunsTable() {
  const columns = useJobRunColumns();
  const [params] = useQueryStates({
    page: parseAsInteger.withDefault(1),
    perPage: parseAsInteger.withDefault(10),
    job: parseAsString,
    trigger: parseAsString,
    runStatus: parseAsString,
    sort: getSortingStateParser(columnIds).withDefault([])
  });

  const filters = {
    page: params.page,
    pageSize: params.perPage,
    ...(params.job && { jobKey: params.job }),
    ...(params.trigger && { trigger: params.trigger }),
    ...(params.runStatus && { status: params.runStatus }),
    ...(params.sort.length > 0 && { sort: JSON.stringify(params.sort) })
  };

  const { data } = useSuspenseQuery(jobRunsQueryOptions(filters));

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
