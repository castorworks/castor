'use client';
import { Badge } from '@/components/ui/badge';
import { DataTableColumnHeader } from '@/components/ui/table/data-table-column-header';
import type { Asset } from '../../api/types';
import { Column, ColumnDef } from '@tanstack/react-table';
import { Checkbox } from '@/components/ui/checkbox';
import { Icons } from '@/components/icons';
import { formatDateTime } from '@/lib/format';
import { CellAction } from './cell-action';
import { getDictLabel, getDictVariant, getDictIcon, getDictOptions } from '@/lib/dict';
import { useLocale, useTranslations } from 'next-intl';

export const columnIds = [
  'select',
  'asset',
  'category',
  'status',
  'public',
  'folderPath',
  'createdAt',
  'actions_col'
];

export function useAssetColumns(): ColumnDef<Asset>[] {
  const locale = useLocale();
  const t = useTranslations('assets.table');

  return [
    {
      id: 'select',
      size: 40,
      minSize: 40,
      maxSize: 40,
      header: ({ table }) => (
        <div className='flex h-8 items-center'>
          <Checkbox
            checked={
              table.getIsAllPageRowsSelected() ||
              (table.getIsSomePageRowsSelected() && 'indeterminate')
            }
            onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
            aria-label='Select all'
          />
        </div>
      ),
      cell: ({ row }) => (
        <div className='flex h-8 items-center'>
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(!!value)}
            aria-label='Select row'
          />
        </div>
      ),
      enableSorting: false,
      enableColumnFilter: false,
      enableHiding: false
    },
    {
      id: 'asset',
      accessorFn: (row) => row.name,
      header: ({ column }: { column: Column<Asset, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('asset')} />
      ),
      cell: ({ row }) => {
        const asset = row.original;
        const iconKey = getDictIcon('asset_category', asset.category) as
          | keyof typeof Icons
          | undefined;
        const IconComponent = iconKey ? (Icons[iconKey] ?? Icons.page) : Icons.page;
        return (
          <div className='flex items-center gap-3'>
            <IconComponent className='h-4 w-4 text-muted-foreground' />
            <div className='flex flex-col'>
              <span className='font-medium'>{asset.name}</span>
              <span className='text-muted-foreground text-xs'>
                {asset.sizeFormatted} &middot; {asset.extension.toUpperCase()}
              </span>
            </div>
          </div>
        );
      },
      meta: {
        label: t('asset'),
        placeholder: t('search'),
        variant: 'text' as const,
        icon: Icons.search
      },
      enableColumnFilter: true
    },
    {
      id: 'category',
      accessorKey: 'category',
      header: ({ column }: { column: Column<Asset, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('category')} />
      ),
      cell: ({ row }) => {
        const cat = row.original.category;
        return (
          <Badge variant={getDictVariant('asset_category', cat)}>
            {getDictLabel('asset_category', cat)}
          </Badge>
        );
      },
      enableColumnFilter: true,
      meta: {
        label: t('category'),
        variant: 'multiSelect' as const,
        get options() {
          return getDictOptions('asset_category');
        }
      }
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: ({ column }: { column: Column<Asset, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('status')} />
      ),
      cell: ({ row }) => {
        const status = row.original.status;
        return (
          <Badge variant={getDictVariant('asset_status', status)}>
            {getDictLabel('asset_status', status)}
          </Badge>
        );
      },
      enableColumnFilter: true,
      meta: {
        label: t('status'),
        variant: 'multiSelect' as const,
        get options() {
          return getDictOptions('asset_status');
        }
      }
    },
    {
      id: 'public',
      header: () => <div className='flex h-8 items-center'>{t('visibility')}</div>,
      cell: ({ row }) => {
        if (row.original.isPublic) {
          return (
            <Badge variant='outline'>
              <Icons.share className='mr-1 h-3 w-3' /> {t('public')}
            </Badge>
          );
        }
        return (
          <Badge variant='secondary'>
            <Icons.lock className='mr-1 h-3 w-3' /> {t('private')}
          </Badge>
        );
      }
    },
    {
      accessorKey: 'folderPath',
      header: ({ column }: { column: Column<Asset, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('folder')} />
      ),
      cell: ({ row }) => (
        <span className='text-muted-foreground text-sm'>{row.original.folderPath || '—'}</span>
      )
    },
    {
      accessorKey: 'createdAt',
      header: ({ column }: { column: Column<Asset, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('created')} />
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
      size: 50,
      minSize: 50,
      maxSize: 50,
      cell: ({ row }) => <CellAction data={row.original} />
    }
  ];
}

/** @deprecated Use useAssetColumns() hook instead */
export const columns = undefined as unknown as ColumnDef<Asset>[];
