'use client';
import { Badge } from '@/components/ui/badge';
import { DataTableColumnHeader } from '@/components/ui/table/data-table-column-header';
import type { Resource } from '../../api/types';
import { Column, ColumnDef } from '@tanstack/react-table';
import { Icons } from '@/components/icons';
import { CellAction } from './cell-action';
import { useSeedTranslation } from '@/lib/use-seed-translation';
import { useTranslations } from 'next-intl';

function ResourceNameCell({ resource }: { resource: Resource }) {
  const { ts } = useSeedTranslation();
  return (
    <div className='flex flex-col'>
      <span className='font-medium'>{ts(resource.name)}</span>
      <span className='text-muted-foreground text-xs'>{resource.code}</span>
    </div>
  );
}

type ModuleTranslationKey = Parameters<ReturnType<typeof useTranslations<'resources.modules'>>>[0];

export function useResourceColumns(moduleOptions?: string[]): ColumnDef<Resource>[] {
  const t = useTranslations('resources.table');
  const tm = useTranslations('resources.modules');
  const getModuleLabel = (module: string) => {
    const key = module as ModuleTranslationKey;
    return tm.has(key) ? tm(key) : module;
  };
  const moduleFilterOptions = (moduleOptions ?? []).map((m) => ({
    value: m,
    label: getModuleLabel(m)
  }));

  return [
    {
      id: 'search',
      accessorFn: (row) => row.name,
      header: ({ column }: { column: Column<Resource, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('resource')} />
      ),
      cell: ({ row }) => <ResourceNameCell resource={row.original} />,
      meta: {
        label: t('resource'),
        placeholder: t('search'),
        variant: 'text' as const,
        icon: Icons.text
      },
      enableColumnFilter: true
    },
    {
      accessorKey: 'path',
      header: ({ column }: { column: Column<Resource, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('path')} />
      ),
      cell: ({ row }) => (
        <code className='rounded bg-muted px-1.5 py-0.5 text-xs'>{row.original.path}</code>
      )
    },
    {
      id: 'actions',
      header: t('actions'),
      cell: ({ row }) => (
        <div className='flex flex-wrap gap-1'>
          {row.original.actions.map((action) => (
            <Badge key={action} variant='outline' className='text-xs'>
              {t(`actionLabels.${action}` as 'actionLabels.GET')}
            </Badge>
          ))}
        </div>
      )
    },
    {
      accessorKey: 'category',
      header: ({ column }: { column: Column<Resource, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('category')} />
      ),
      cell: ({ row }) => {
        const cat = row.original.category;
        const variant = cat === 'admin' ? 'default' : 'secondary';
        return <Badge variant={variant}>{t(cat as 'admin')}</Badge>;
      },
      enableColumnFilter: true,
      meta: {
        label: t('category'),
        variant: 'select' as const,
        options: [
          { value: 'admin', label: t('admin') },
          { value: 'user', label: t('user') }
        ]
      }
    },
    {
      accessorKey: 'module',
      header: ({ column }: { column: Column<Resource, unknown> }) => (
        <DataTableColumnHeader column={column} title={t('module')} />
      ),
      cell: ({ row }) => (
        <span className='text-sm'>
          {row.original.module ? getModuleLabel(row.original.module) : '—'}
        </span>
      ),
      enableColumnFilter: moduleFilterOptions.length > 0,
      ...(moduleFilterOptions.length > 0
        ? {
            meta: {
              label: t('module'),
              variant: 'select' as const,
              options: moduleFilterOptions
            }
          }
        : {})
    },
    {
      id: 'status',
      header: t('status'),
      cell: ({ row }) => {
        if (!row.original.isEnabled) {
          return <Badge variant='destructive'>{t('disabled')}</Badge>;
        }
        if (row.original.isSystem) {
          return <Badge variant='secondary'>{t('system')}</Badge>;
        }
        return <Badge variant='default'>{t('active')}</Badge>;
      },
      enableColumnFilter: true,
      meta: {
        label: t('status'),
        variant: 'select' as const,
        options: [
          { value: 'enabled', label: t('activeSystem') },
          { value: 'disabled', label: t('disabled') }
        ]
      }
    },
    {
      id: 'actions_col',
      cell: ({ row }) => <CellAction data={row.original} />
    }
  ];
}
