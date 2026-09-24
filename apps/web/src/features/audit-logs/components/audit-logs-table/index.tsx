'use client';

import { DataTable } from '@/components/ui/table/data-table';
import { DataTableToolbar } from '@/components/ui/table/data-table-toolbar';
import { useDataTable } from '@/hooks/use-data-table';
import { usePermission } from '@/hooks/use-permission';
import { ExportButton } from '@/components/data-io/export-button';
import { exportAuditLogs } from '../../api/service';

import { useSuspenseQuery } from '@tanstack/react-query';
import { parseAsInteger, parseAsString, useQueryStates } from 'nuqs';
import { getSortingStateParser } from '@/lib/parsers';
import { auditLogsQueryOptions } from '../../api/queries';
import { useAuditLogColumns, columnIds } from './columns';

export function AuditLogsTable() {
  const columns = useAuditLogColumns();
  const [params] = useQueryStates({
    page: parseAsInteger.withDefault(1),
    perPage: parseAsInteger.withDefault(10),
    logType: parseAsString,
    operator: parseAsString,
    target: parseAsString,
    success: parseAsString,
    sort: getSortingStateParser(columnIds).withDefault([])
  });

  const filters = {
    page: params.page,
    pageSize: params.perPage,
    ...(params.logType && { logType: params.logType }),
    ...(params.operator && { operator: params.operator }),
    ...(params.target && { target: params.target }),
    ...(params.success && { success: params.success }),
    ...(params.sort.length > 0 && { sort: JSON.stringify(params.sort) })
  };

  const { data } = useSuspenseQuery(auditLogsQueryOptions(filters));

  const canExport = usePermission('/api/v1/admin/audit-logs/export:GET');

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
        {canExport && <ExportButton onExport={(format) => exportAuditLogs(filters, format)} />}
      </DataTableToolbar>
    </DataTable>
  );
}

export function AuditLogsTableSkeleton() {
  return (
    <div className='flex flex-1 animate-pulse flex-col gap-4'>
      <div className='bg-muted h-10 w-full rounded' />
      <div className='bg-muted h-96 w-full rounded-lg' />
      <div className='bg-muted h-10 w-full rounded' />
    </div>
  );
}
