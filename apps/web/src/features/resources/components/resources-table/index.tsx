'use client';

import { DataTable } from '@/components/ui/table/data-table';
import { DataTableToolbar } from '@/components/ui/table/data-table-toolbar';
import { useDataTable } from '@/hooks/use-data-table';
import { useSuspenseQuery } from '@tanstack/react-query';
import { parseAsInteger, parseAsString, useQueryStates } from 'nuqs';
import { getSortingStateParser } from '@/lib/parsers';
import { buildResourceFilters } from '../../api/filters';
import { resourcesQueryOptions } from '../../api/queries';
import { useResourceColumns } from './columns';

const columnIdsBase = ['search', 'path', 'actions', 'category', 'module', 'status', 'actions_col'];

interface ResourcesTableProps {
  moduleOptions?: string[];
}

export function ResourcesTable({ moduleOptions }: ResourcesTableProps) {
  const [params] = useQueryStates({
    page: parseAsInteger.withDefault(1),
    perPage: parseAsInteger.withDefault(10),
    search: parseAsString,
    category: parseAsString,
    module: parseAsString,
    status: parseAsString,
    sort: getSortingStateParser(columnIdsBase).withDefault([])
  });

  const filters = buildResourceFilters({
    ...params,
    sort: params.sort.length > 0 ? JSON.stringify(params.sort) : null
  });

  const { data } = useSuspenseQuery(resourcesQueryOptions(filters));

  const columns = useResourceColumns(moduleOptions);

  const { table } = useDataTable({
    data: data.list,
    columns,
    rowCount: data.total,
    shallow: true,
    debounceMs: 500,
    initialState: {
      columnPinning: { right: ['actions_col'] }
    }
  });

  return (
    <DataTable table={table}>
      <DataTableToolbar table={table} />
    </DataTable>
  );
}

export function ResourcesTableSkeleton() {
  return (
    <div className='flex flex-1 animate-pulse flex-col gap-4'>
      <div className='bg-muted h-10 w-full rounded' />
      <div className='bg-muted h-96 w-full rounded-lg' />
      <div className='bg-muted h-10 w-full rounded' />
    </div>
  );
}
