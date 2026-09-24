'use client';
import { Badge } from '@/components/ui/badge';
import { DataTableColumnHeader } from '@/components/ui/table/data-table-column-header';
import type { Column, ColumnDef } from '@tanstack/react-table';
import { Icons } from '@/components/icons';
import { formatDateTime } from '@/lib/format';
import { useTranslations, useLocale } from 'next-intl';
import type { Session } from '../../api/types';
import { describeUserAgent } from '../../lib/user-agent';
import { CellAction } from './cell-action';

export const columnIds = [
  'user',
  'ip',
  'device',
  'createdAt',
  'lastActiveAt',
  'expiresAt',
  'actions_col'
];

export function useSessionColumns(): ColumnDef<Session>[] {
  const t = useTranslations('sessions');
  const locale = useLocale();

  return [
    {
      id: 'user',
      accessorFn: (row) => row.username,
      header: ({ column }: { column: Column<Session, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('columns.user')} />
      ),
      cell: ({ row }) => (
        <div className='flex items-center gap-2'>
          <span className='font-medium'>@{row.original.username}</span>
          {row.original.current && <Badge variant='info'>{t('current')}</Badge>}
        </div>
      ),
      meta: {
        label: t('columns.user'),
        placeholder: t('columns.searchByUsername'),
        variant: 'text' as const,
        icon: Icons.search
      },
      enableColumnFilter: true
    },
    {
      id: 'ip',
      accessorFn: (row) => row.ipAddr,
      header: t('columns.ip'),
      cell: ({ row }) => <span className='font-mono text-sm'>{row.original.ipAddr || '—'}</span>,
      meta: {
        label: t('columns.ip'),
        placeholder: t('columns.searchByIp'),
        variant: 'text' as const,
        icon: Icons.search
      },
      enableColumnFilter: true,
      enableSorting: false
    },
    {
      id: 'device',
      header: t('columns.device'),
      cell: ({ row }) => (
        <div className='flex flex-col gap-1'>
          <span className='text-sm' title={row.original.userAgent}>
            {describeUserAgent(row.original.userAgent) || t('unknownDevice')}
          </span>
          {row.original.remember && (
            <span className='text-muted-foreground text-xs'>{t('remembered')}</span>
          )}
        </div>
      ),
      enableSorting: false
    },
    {
      accessorKey: 'createdAt',
      header: ({ column }: { column: Column<Session, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('columns.signedInAt')} />
      ),
      cell: ({ row }) => (
        <span className='text-sm' suppressHydrationWarning>
          {formatDateTime(row.original.createdAt, { locale })}
        </span>
      )
    },
    {
      accessorKey: 'lastActiveAt',
      header: ({ column }: { column: Column<Session, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('columns.lastActiveAt')} />
      ),
      cell: ({ row }) => (
        <span className='text-sm' suppressHydrationWarning>
          {formatDateTime(row.original.lastActiveAt, { locale })}
        </span>
      )
    },
    {
      accessorKey: 'expiresAt',
      header: ({ column }: { column: Column<Session, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('columns.expiresAt')} />
      ),
      cell: ({ row }) => (
        <span className='text-muted-foreground text-sm' suppressHydrationWarning>
          {formatDateTime(row.original.expiresAt, { locale })}
        </span>
      )
    },
    {
      id: 'actions_col',
      cell: ({ row }) => <CellAction data={row.original} />
    }
  ];
}
