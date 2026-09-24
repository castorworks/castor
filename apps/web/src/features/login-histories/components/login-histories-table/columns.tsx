'use client';
import { Badge } from '@/components/ui/badge';
import { DataTableColumnHeader } from '@/components/ui/table/data-table-column-header';
import type { LoginHistory } from '../../api/types';
import { Column, ColumnDef } from '@tanstack/react-table';
import { Icons } from '@/components/icons';
import { formatDateTime } from '@/lib/format';
import { CellAction } from './cell-action';
import { useDict } from '@/hooks/use-dict';
import { useTranslations, useLocale } from 'next-intl';

export const columnIds = ['user', 'loginMethod', 'result', 'createdAt', 'actions_col'];

export function useLoginHistoryColumns(): ColumnDef<LoginHistory>[] {
  const t = useTranslations('loginHistory');
  const locale = useLocale();
  const dict = useDict();

  return [
    {
      id: 'user',
      accessorFn: (row) => row.username,
      header: ({ column }: { column: Column<LoginHistory, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('columns.user')} />
      ),
      cell: ({ row }) => (
        <div className='flex flex-col'>
          <span className='font-medium'>@{row.original.username}</span>
          <span className='text-muted-foreground text-xs'>{row.original.ipAddr}</span>
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
      accessorKey: 'loginMethod',
      header: ({ column }: { column: Column<LoginHistory, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('columns.method')} />
      ),
      cell: ({ row }) => {
        const method = row.original.loginMethod;
        return <span className='text-sm'>{dict.label('login_method', method)}</span>;
      },
      enableColumnFilter: true,
      meta: {
        label: t('columns.method'),
        variant: 'multiSelect' as const,
        get options() {
          return dict.options('login_method');
        }
      }
    },
    {
      id: 'result',
      header: t('columns.result'),
      cell: ({ row }) => {
        if (row.original.success) {
          return (
            <Badge variant='success'>
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
      accessorKey: 'createdAt',
      header: ({ column }: { column: Column<LoginHistory, unknown> }) => (
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
    },
    {
      id: 'actions_col',
      cell: ({ row }) => <CellAction data={row.original} />
    }
  ];
}

/** @deprecated Use useLoginHistoryColumns() hook instead */
export const columns = undefined as unknown as ColumnDef<LoginHistory>[];
