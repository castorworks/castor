'use client';

import { Badge } from '@/components/ui/badge';
import { DataTableColumnHeader } from '@/components/ui/table/data-table-column-header';
import { Icons } from '@/components/icons';
import { formatDateTime } from '@/lib/format';
import { getDictLabel, getDictOptions, getDictVariant } from '@/lib/dict';
import type { Notification } from '../../api/types';
import { Column, ColumnDef } from '@tanstack/react-table';
import { useLocale, useTranslations } from 'next-intl';
import { CellAction } from './cell-action';

export const notificationColumnIds = [
  'title',
  'type',
  'level',
  'scope',
  'delivery',
  'expireAt',
  'createdAt',
  'actions'
];

function getNotificationOptions(typeCode: string, fallback: { value: string; label: string }[]) {
  const options = getDictOptions(typeCode);
  return options.length > 0 ? options : fallback;
}

export function useAdminNotificationColumns(): ColumnDef<Notification>[] {
  const t = useTranslations('notifications.admin');
  const locale = useLocale();

  return [
    {
      id: 'title',
      accessorFn: (row) => row.title,
      header: ({ column }: { column: Column<Notification, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('table.title')} />
      ),
      cell: ({ row }) => (
        <div className='flex min-w-64 max-w-md flex-col gap-1'>
          <span className='truncate font-medium'>{row.original.title}</span>
          <span className='text-muted-foreground line-clamp-1 text-xs'>
            {row.original.content || t('table.noContent')}
          </span>
        </div>
      ),
      meta: {
        label: t('table.title'),
        placeholder: t('table.searchTitle'),
        variant: 'text' as const,
        icon: Icons.search
      },
      enableColumnFilter: true
    },
    {
      id: 'type',
      accessorKey: 'type',
      header: ({ column }: { column: Column<Notification, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('table.type')} />
      ),
      cell: ({ row }) => (
        <Badge variant={getDictVariant('notification_type', row.original.type)}>
          {getDictLabel('notification_type', row.original.type)}
        </Badge>
      ),
      enableColumnFilter: true,
      meta: {
        label: t('table.type'),
        variant: 'multiSelect' as const,
        get options() {
          return getNotificationOptions('notification_type', [
            { value: 'SYSTEM', label: t('options.types.system') },
            { value: 'ANNOUNCE', label: t('options.types.announce') },
            { value: 'MESSAGE', label: t('options.types.message') },
            { value: 'ALERT', label: t('options.types.alert') },
            { value: 'TASK', label: t('options.types.task') }
          ]);
        }
      }
    },
    {
      id: 'level',
      accessorKey: 'level',
      header: ({ column }: { column: Column<Notification, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('table.level')} />
      ),
      cell: ({ row }) => (
        <Badge variant={getDictVariant('notification_level', row.original.level)}>
          {getDictLabel('notification_level', row.original.level)}
        </Badge>
      ),
      enableColumnFilter: true,
      meta: {
        label: t('table.level'),
        variant: 'multiSelect' as const,
        get options() {
          return getNotificationOptions('notification_level', [
            { value: 'INFO', label: t('options.levels.info') },
            { value: 'SUCCESS', label: t('options.levels.success') },
            { value: 'WARNING', label: t('options.levels.warning') },
            { value: 'ERROR', label: t('options.levels.error') }
          ]);
        }
      }
    },
    {
      id: 'scope',
      header: ({ column }: { column: Column<Notification, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('table.scope')} />
      ),
      cell: ({ row }) => {
        const isGlobal = row.original.isGlobal;
        return (
          <Badge variant={isGlobal ? 'default' : 'secondary'}>
            {isGlobal ? (
              <Icons.globe className='mr-1 h-3 w-3' />
            ) : (
              <Icons.teams className='mr-1 h-3 w-3' />
            )}
            {isGlobal ? t('table.global') : t('table.targeted')}
          </Badge>
        );
      },
      enableSorting: false
    },
    {
      id: 'delivery',
      header: t('table.delivery'),
      cell: ({ row }) => {
        const n = row.original;
        return (
          <div className='grid min-w-48 grid-cols-2 gap-x-3 gap-y-1 text-xs'>
            <span className='text-muted-foreground'>{t('table.recipients')}</span>
            <span className='text-right tabular-nums'>{n.recipientCount}</span>
            <span className='text-muted-foreground'>{t('table.read')}</span>
            <span className='text-right tabular-nums'>{n.readCount}</span>
            <span className='text-muted-foreground'>{t('table.unread')}</span>
            <span className='text-right tabular-nums'>{n.unreadCount}</span>
            <span className='text-muted-foreground'>{t('table.dismissed')}</span>
            <span className='text-right tabular-nums'>{n.dismissedCount}</span>
          </div>
        );
      },
      enableSorting: false
    },
    {
      id: 'expireAt',
      accessorKey: 'expireAt',
      header: ({ column }: { column: Column<Notification, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('table.expireAt')} />
      ),
      cell: ({ row }) => {
        const date = row.original.expireAt
          ? formatDateTime(row.original.expireAt, { locale })
          : t('table.neverExpires');
        return (
          <span className='text-sm' suppressHydrationWarning>
            {date}
          </span>
        );
      }
    },
    {
      id: 'createdAt',
      accessorKey: 'createdAt',
      header: ({ column }: { column: Column<Notification, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('table.createdAt')} />
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
      size: 50,
      minSize: 50,
      maxSize: 50,
      cell: ({ row }) => <CellAction data={row.original} />
    }
  ];
}
