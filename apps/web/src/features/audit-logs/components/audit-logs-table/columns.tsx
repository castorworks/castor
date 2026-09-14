'use client';

import { Badge } from '@/components/ui/badge';
import { DataTableColumnHeader } from '@/components/ui/table/data-table-column-header';
import { Icons } from '@/components/icons';
import { formatDateTime } from '@/lib/format';
import type { AuditLog } from '../../api/types';
import { Column, ColumnDef } from '@tanstack/react-table';
import { getDictLabel, getDictOptions } from '@/lib/dict';
import { useLocale, useTranslations } from 'next-intl';

export const columnIds = [
  'logType',
  'operator',
  'target',
  'details',
  'success',
  'ipAddr',
  'createdAt'
];

export function useAuditLogColumns(): ColumnDef<AuditLog>[] {
  const locale = useLocale();
  const t = useTranslations('auditLogs');

  return [
    {
      id: 'logType',
      accessorKey: 'logType',
      header: ({ column }: { column: Column<AuditLog, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('columns.type')} />
      ),
      cell: ({ row }) => {
        const type = row.original.logType;
        return <Badge variant='outline'>{getDictLabel('audit_log_type', type)}</Badge>;
      },
      enableColumnFilter: true,
      meta: {
        label: t('columns.type'),
        variant: 'multiSelect' as const,
        get options() {
          return getDictOptions('audit_log_type');
        }
      }
    },
    {
      id: 'operator',
      accessorKey: 'operator',
      header: ({ column }: { column: Column<AuditLog, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('columns.operator')} />
      ),
      cell: ({ row }) => (
        <div className='flex flex-col'>
          <span className='font-medium'>@{row.original.operator || 'unknown'}</span>
          <span className='text-muted-foreground text-xs'>
            ID: {row.original.operatorId || '-'}
          </span>
        </div>
      ),
      enableColumnFilter: true,
      meta: {
        label: t('columns.operator'),
        placeholder: t('table.operatorSearch'),
        variant: 'text' as const,
        icon: Icons.search
      }
    },
    {
      id: 'target',
      accessorKey: 'target',
      header: ({ column }: { column: Column<AuditLog, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('columns.target')} />
      ),
      cell: ({ row }) => (
        <code className='rounded bg-muted px-1.5 py-0.5 text-xs'>{row.original.target || '-'}</code>
      ),
      enableColumnFilter: true,
      meta: {
        label: t('columns.target'),
        placeholder: t('table.targetSearch'),
        variant: 'text' as const
      }
    },
    {
      id: 'details',
      accessorKey: 'details',
      header: ({ column }: { column: Column<AuditLog, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('columns.details')} />
      ),
      cell: ({ row }) => (
        <span className='block max-w-md truncate text-sm'>{row.original.details || '-'}</span>
      )
    },
    {
      id: 'success',
      accessorKey: 'success',
      header: t('columns.result'),
      cell: ({ row }) => {
        if (row.original.success) {
          return (
            <Badge variant='default' className='bg-green-600 text-white hover:bg-green-600/90'>
              <Icons.check className='mr-1 h-3 w-3' /> {t('success')}
            </Badge>
          );
        }
        return (
          <Badge variant='destructive'>
            <Icons.close className='mr-1 h-3 w-3' /> {t('failed')}
          </Badge>
        );
      },
      enableColumnFilter: true,
      meta: {
        label: t('columns.result'),
        variant: 'select' as const,
        options: [
          { value: 'true', label: t('success') },
          { value: 'false', label: t('failed') }
        ]
      }
    },
    {
      id: 'ipAddr',
      accessorKey: 'ipAddr',
      header: ({ column }: { column: Column<AuditLog, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('columns.ip')} />
      ),
      cell: ({ row }) => (
        <code className='text-muted-foreground text-xs'>{row.original.ipAddr}</code>
      )
    },
    {
      id: 'createdAt',
      accessorKey: 'createdAt',
      header: ({ column }: { column: Column<AuditLog, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('columns.time')} />
      ),
      cell: ({ row }) => {
        const date = formatDateTime(row.original.createdAt, { locale });
        return (
          <span className='text-sm' suppressHydrationWarning>
            {date}
          </span>
        );
      }
    }
  ];
}

/** @deprecated Use useAuditLogColumns() hook instead */
export const columns = undefined as unknown as ColumnDef<AuditLog>[];
