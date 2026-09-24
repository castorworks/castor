'use client';

import { DataTable } from '@/components/ui/table/data-table';
import { DataTableToolbar } from '@/components/ui/table/data-table-toolbar';
import { useDataTable } from '@/hooks/use-data-table';
import { getSortingStateParser } from '@/lib/parsers';
import { useSuspenseQuery } from '@tanstack/react-query';
import { parseAsInteger, parseAsString, useQueryStates } from 'nuqs';
import { adminNotificationsQueryOptions } from '../../api/queries';
import type { Notification } from '../../api/types';
import { notificationColumnIds, useAdminNotificationColumns } from './columns';

export function AdminNotificationsTable() {
  const columns = useAdminNotificationColumns();
  const [params] = useQueryStates({
    page: parseAsInteger.withDefault(1),
    perPage: parseAsInteger.withDefault(10),
    title: parseAsString,
    type: parseAsString,
    level: parseAsString,
    sort: getSortingStateParser(notificationColumnIds).withDefault([])
  });

  const filters = {
    page: params.page,
    pageSize: params.perPage,
    ...(params.title && { title: params.title }),
    ...(params.type && { type: params.type }),
    ...(params.level && { level: params.level }),
    ...(params.sort.length > 0 && { sort: JSON.stringify(params.sort) })
  };

  const { data } = useSuspenseQuery(adminNotificationsQueryOptions(filters));

  const { table } = useDataTable<Notification>({
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
      <DataTableToolbar table={table} />
    </DataTable>
  );
}

export function AdminNotificationsTableSkeleton() {
  return (
    <div className='flex flex-1 animate-pulse flex-col gap-4'>
      <div className='bg-muted h-10 w-full rounded' />
      <div className='bg-muted h-96 w-full rounded-lg' />
      <div className='bg-muted h-10 w-full rounded' />
    </div>
  );
}
