'use client';
import { Badge } from '@/components/ui/badge';
import { DataTableColumnHeader } from '@/components/ui/table/data-table-column-header';
import type { User } from '../../api/types';
import { Column, ColumnDef } from '@tanstack/react-table';
import { Icons } from '@/components/icons';
import { formatDateTime } from '@/lib/format';
import { CellAction } from './cell-action';
import { getDictLabel, getDictOptions } from '@/lib/dict';
import { useTranslations, useLocale } from 'next-intl';

export const userColumnIds = ['search', 'accountSource', 'status', 'createdAt', 'actions'];

export function useUserColumns(): ColumnDef<User>[] {
  const t = useTranslations('users');
  const locale = useLocale();

  return [
    {
      id: 'search',
      accessorFn: (row) => row.name || row.username,
      header: ({ column }: { column: Column<User, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('table.user')} />
      ),
      cell: ({ row }) => (
        <div className='flex items-center gap-3'>
          <div className='flex flex-col'>
            <span className='font-medium'>{row.original.name || row.original.username}</span>
            <span className='text-muted-foreground text-xs'>@{row.original.username}</span>
          </div>
        </div>
      ),
      meta: {
        label: t('table.user'),
        placeholder: t('table.searchUsers'),
        variant: 'text' as const,
        icon: Icons.text
      },
      enableColumnFilter: true
    },
    {
      id: 'accountSource',
      accessorKey: 'accountSource',
      header: ({ column }: { column: Column<User, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('table.source')} />
      ),
      cell: ({ row }) => {
        const source = row.original.accountSource;
        return <span className='text-sm'>{getDictLabel('account_source', source)}</span>;
      },
      enableColumnFilter: true,
      meta: {
        label: t('table.source'),
        variant: 'multiSelect' as const,
        get options() {
          return getDictOptions('account_source');
        }
      }
    },
    {
      id: 'status',
      header: ({ column }: { column: Column<User, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('table.status')} />
      ),
      cell: ({ row }) => {
        const { enable, locked } = row.original;
        if (!enable) {
          return <Badge variant='destructive'>{t('table.disabled')}</Badge>;
        }
        if (locked) {
          return <Badge variant='secondary'>{t('table.locked')}</Badge>;
        }
        return <Badge variant='default'>{t('table.active')}</Badge>;
      },
      enableColumnFilter: true,
      meta: {
        label: t('table.status'),
        variant: 'select' as const,
        options: [
          { value: 'active', label: t('table.active') },
          { value: 'disabled', label: t('table.disabled') },
          { value: 'locked', label: t('table.locked') }
        ]
      }
    },
    {
      accessorKey: 'createdAt',
      header: ({ column }: { column: Column<User, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('table.created')} />
      ),
      cell: ({ row }) => {
        const date = formatDateTime(row.original.createdAt, { locale });
        return (
          <span className='text-sm' suppressHydrationWarning>
            {date}
          </span>
        );
      }
    },
    {
      id: 'actions',
      cell: ({ row }) => <CellAction data={row.original} />
    }
  ];
}
